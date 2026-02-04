package provider

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	_ "modernc.org/sqlite"
	"golang.org/x/crypto/bcrypt"
	"github.com/yourusername/open-z3950-gateway/pkg/z3950"
)

// SQLiteProvider implements the Provider interface for a SQLite database.
type SQLiteProvider struct {
	db      *sql.DB
	profile *z3950.MARCProfile
}

// NewSQLiteProvider creates a new provider using the given database file path.
// It initializes the database if the required table doesn't exist.
func NewSQLiteProvider(path string) (*SQLiteProvider, error) {
	if path == "" {
		return nil, fmt.Errorf("sqlite provider requires a non-empty database path")
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite db at %s: %w", path, err)
	}

	// Check if the 'bibliography' table exists. If not, we need to seed the DB.
	var tableName string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='bibliography'").Scan(&tableName)

	if err == sql.ErrNoRows {
		// The table doesn't exist, so we run the seeder.
		fmt.Printf("Table 'bibliography' not found in %s. Seeding database...\n", path)
		if seedErr := seedDatabase(db); seedErr != nil {
			db.Close()
			return nil, fmt.Errorf("failed to seed database: %w", seedErr)
		}
	} else if err != nil {
		// A different error occurred during the check.
		db.Close()
		return nil, fmt.Errorf("failed to check for existing tables in database %s: %w", path, err)
	}

	// Ensure ill_requests table exists
	createILLTableSQL := `
	CREATE TABLE IF NOT EXISTS ill_requests (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		target_db TEXT,
		record_id TEXT,
		title TEXT,
		author TEXT,
		isbn TEXT,
		status TEXT,
		requestor TEXT,
		comments TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`
	if _, err := db.Exec(createILLTableSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create ill_requests table: %w", err)
	}

	// Ensure users table exists
	createUserTableSQL := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		role TEXT DEFAULT 'user',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`
	if _, err := db.Exec(createUserTableSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create users table: %w", err)
	}
	
	// Seed admin user if not exists
	var userCount int
	db.QueryRow("SELECT COUNT(*) FROM users WHERE username = 'admin'").Scan(&userCount)
	if userCount == 0 {
		hash, _ := bcrypt.GenerateFromPassword([]byte("admin"), bcrypt.DefaultCost)
		db.Exec("INSERT INTO users (username, password_hash, role) VALUES (?, ?, ?)", "admin", string(hash), "admin")
	}

	// Ensure targets table exists
	createTargetsTableSQL := `
	CREATE TABLE IF NOT EXISTS targets (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT UNIQUE NOT NULL,
		host TEXT NOT NULL,
		port INTEGER NOT NULL,
		database_name TEXT NOT NULL,
		encoding TEXT DEFAULT 'MARC21',
		auth_user TEXT,
		auth_pass TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`
	if _, err := db.Exec(createTargetsTableSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create targets table: %w", err)
	}

	// Seed default targets if empty
	var targetCount int
	db.QueryRow("SELECT COUNT(*) FROM targets").Scan(&targetCount)
	if targetCount == 0 {
		seedTargets := []string{
			"INSERT INTO targets (name, host, port, database_name, encoding) VALUES ('Library of Congress', 'lx2.loc.gov', 210, 'LCDB', 'MARC21')",
			"INSERT INTO targets (name, host, port, database_name, encoding) VALUES ('LCDB', 'lx2.loc.gov', 210, 'LCDB', 'MARC21')", // Alias for webapp default
			"INSERT INTO targets (name, host, port, database_name, encoding) VALUES ('Oxford University', 'library.ox.ac.uk', 210, 'MAIN_BIB', 'MARC21')",
		}
		for _, q := range seedTargets {
			db.Exec(q)
		}
	}

	// Ensure holdings table exists (Legacy table, might migrate to items later, but keep for now)
	createHoldingsTableSQL := `
	CREATE TABLE IF NOT EXISTS holdings (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		bib_id INTEGER,
		call_number TEXT,
		status TEXT,
		location TEXT,
		FOREIGN KEY(bib_id) REFERENCES bibliography(id)
	);
	`
	if _, err := db.Exec(createHoldingsTableSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create holdings table: %w", err)
	}

	// Ensure items table exists (New for Circulation)
	createItemsTableSQL := `
	CREATE TABLE IF NOT EXISTS items (
		id INTEGER PRIMARY KEY,
		bib_id INTEGER,
		barcode TEXT UNIQUE,
		call_number TEXT,
		status TEXT DEFAULT 'Available',
		location TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`
	if _, err := db.Exec(createItemsTableSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create items table: %w", err)
	}

	// Ensure loans table exists (New for Circulation)
	createLoansTableSQL := `
	CREATE TABLE IF NOT EXISTS loans (
		id INTEGER PRIMARY KEY,
		item_id INTEGER,
		patron_id TEXT,
		checkout_date DATETIME DEFAULT CURRENT_TIMESTAMP,
		due_date DATETIME,
		return_date DATETIME,
		status TEXT DEFAULT 'active'
	);
	`
	if _, err := db.Exec(createLoansTableSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create loans table: %w", err)
	}

	format := os.Getenv("ZSERVER_MARC_FORMAT")
	profile := &z3950.ProfileMARC21
	if format == "CNMARC" {
		profile = &z3950.ProfileCNMARC
	} else if format == "UNIMARC" {
		profile = &z3950.ProfileUNIMARC
	}

	// Migration for new columns (ignore errors if they exist)
	db.Exec("ALTER TABLE bibliography ADD COLUMN issn TEXT")
	db.Exec("ALTER TABLE bibliography ADD COLUMN subjects TEXT")
	db.Exec("ALTER TABLE ill_requests ADD COLUMN comments TEXT")

	return &SQLiteProvider{db: db, profile: profile}, nil
}

// seedDatabase creates the table and inserts sample data.
func seedDatabase(db *sql.DB) error {
	createTableSQL := `
	CREATE TABLE bibliography (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT,
		author TEXT,
		isbn TEXT,
		publisher TEXT,
		pub_year TEXT,
		issn TEXT,
		subjects TEXT,
		raw_record TEXT,
		raw_record_format TEXT
	);
	`
	if _, err := db.Exec(createTableSQL); err != nil {
		return err
	}

	insertSQL := `
	INSERT INTO bibliography (id, title, author, isbn, publisher, pub_year, issn, subjects) VALUES
	(1, 'The Go Programming Language', 'Alan A. A. Donovan, Brian W. Kernighan', '978-0134190440', 'Addison-Wesley', '2015', '', 'Programming, Go'),
	(2, 'Thinking in Go', 'Rob Pike', '0201548550', 'PublisherX', '2018', '', 'Programming, Philosophy'),
	(3, 'Go in Practice', 'Matt Butcher, Matt Farina', '978-1617291784', 'Manning', '2016', '', 'Programming, Practice'),
	(4, 'Black Hat Go', 'Tom Steele, Chris Patten, Dan Kottmann', '978-1593278651', 'No Starch Press', '2020', '', 'Security, Go');
	`
	_, err := db.Exec(insertSQL)
	return err
}

// buildSQL recursively builds WHERE clause and args from QueryNode
func buildSQL(node z3950.QueryNode) (string, []interface{}, error) {
	if node == nil {
		return "", nil, nil
	}

	switch n := node.(type) {
	case z3950.QueryClause:
		term := n.Term
		switch n.Attribute {
		case z3950.UseAttributeTitle:
			return "LOWER(title) LIKE ?", []interface{}{"%" + strings.ToLower(term) + "%"}, nil
		case z3950.UseAttributeAuthor:
			return "LOWER(author) LIKE ?", []interface{}{"%" + strings.ToLower(term) + "%"}, nil
		case z3950.UseAttributeISBN:
			return "REPLACE(REPLACE(TRIM(isbn), '-', ''), ' ', '') = ?", []interface{}{"" + CleanISBN(term)}, nil
		case z3950.UseAttributeISSN:
			return "issn LIKE ?", []interface{}{"%" + term + "%"}, nil
		case z3950.UseAttributeSubject:
			return "LOWER(subjects) LIKE ?", []interface{}{"%" + strings.ToLower(term) + "%"}, nil
		case z3950.UseAttributeDatePub:
			return "pub_year LIKE ?", []interface{}{"%" + term + "%"}, nil
		default:
			// Broad search
			likeTerm := "%" + strings.ToLower(term) + "%"
			return "(LOWER(title) LIKE ? OR LOWER(author) LIKE ? OR LOWER(subjects) LIKE ?)", []interface{}{"" + likeTerm, "" + likeTerm, "" + likeTerm}, nil
		}
	case z3950.QueryComplex:
		lSql, lArgs, err := buildSQL(n.Left)
		if err != nil { return "", nil, err }
		rSql, rArgs, err := buildSQL(n.Right)
		if err != nil { return "", nil, err }
		
		op := "AND"
		if n.Operator == "OR" { op = "OR" }
		if n.Operator == "AND-NOT" { op = "AND NOT" }
		
		return fmt.Sprintf("(%s %s %s)", lSql, op, rSql), append(lArgs, rArgs...), nil
	}
	return "", nil, fmt.Errorf("unknown query node type")
}

func (p *SQLiteProvider) Search(db string, query z3950.StructuredQuery) ([]string, error) {
	if query.Root == nil {
		return nil, nil
	}

	whereClause, args, err := buildSQL(query.Root)
	if err != nil {
		return nil, err
	}

	limit := 100
	if query.Limit > 0 {
		limit = query.Limit
	}
	offset := 0
	if query.Offset > 0 {
		offset = query.Offset
	}

	sqlStr := fmt.Sprintf(`SELECT CAST(id AS TEXT) FROM bibliography WHERE %s LIMIT ? OFFSET ?`, whereClause)
	args = append(args, limit, offset)

	rows, err := p.db.Query(sqlStr, args...)
	if err != nil {
		return nil, fmt.Errorf("dynamic query failed: %w. SQL: %s. Args: %v", err, sqlStr, args)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

func (p *SQLiteProvider) Fetch(db string, ids []string) ([]*z3950.MARCRecord, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i := range ids {
		placeholders[i] = "?"
		args[i] = ids[i]
	}

	query := fmt.Sprintf(`
		SELECT id, TRIM(title), TRIM(author), TRIM(isbn), TRIM(publisher), TRIM(pub_year), issn, subjects, raw_record, raw_record_format 
		FROM bibliography 
		WHERE CAST(id AS TEXT) IN (%s)`, strings.Join(placeholders,","))
	
rows, err := p.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []*z3950.MARCRecord
	targetFormat := os.Getenv("ZSERVER_MARC_FORMAT")
	if targetFormat == "" {
		targetFormat = "USMARC"
	}
	
	for rows.Next() {
		var id, title, author, isbn, publisher, pubYear, issn, subjects, rawRecord, rawFormat sql.NullString
		if err := rows.Scan(&id, &title, &author, &isbn, &publisher, &pubYear, &issn, &subjects, &rawRecord, &rawFormat); err != nil {
			continue
		}

		var rec *z3950.MARCRecord
		
		if rawRecord.Valid && rawRecord.String != "" {
			isJSON := rawFormat.Valid && rawFormat.String == "MARC_JSON"
			isMatch := !rawFormat.Valid || rawFormat.String == targetFormat
			if isJSON || isMatch {
				parsed, err := z3950.ParseMARC([]byte(rawRecord.String))
				if err == nil {
					rec = parsed
				}
			}
		}

		if rec == nil {
			t, a, i, pub, y, iss, sub, rid := "", "", "", "", "", "", "", ""
			if title.Valid { t = title.String }
			if author.Valid { a = author.String }
			if isbn.Valid { i = isbn.String }
			if publisher.Valid { pub = publisher.String }
			if pubYear.Valid { y = pubYear.String }
			if issn.Valid { iss = issn.String }
			if subjects.Valid { sub = subjects.String }
			if id.Valid { rid = id.String }
			bytes := z3950.BuildMARC(p.profile, rid, t, a, i, pub, y, iss, sub)
			rec, _ = z3950.ParseMARC(bytes)
		}

		if rec != nil {
			// Fetch holdings
			if id.Valid {
				// We query both 'items' (new) and 'holdings' (old) and merge them for now
				var holdings []z3950.Holding
				
				// 1. New items table
				iRows, err := p.db.Query("SELECT barcode, call_number, status, location FROM items WHERE bib_id = ?", id.String)
				if err == nil {
					defer iRows.Close()
					for iRows.Next() {
						var bc, cn, st, loc sql.NullString
						if err := iRows.Scan(&bc, &cn, &st, &loc); err == nil {
							holdings = append(holdings, z3950.Holding{
								CallNumber: bc.String + " " + cn.String, // Temp hack to show barcode
								Status:     st.String,
								Location:   loc.String,
							})
						}
					}
				}

				// 2. Old holdings table (legacy support)
				hRows, err := p.db.Query("SELECT call_number, status, location FROM holdings WHERE bib_id = ?", id.String)
				if err == nil {
					defer hRows.Close()
					for hRows.Next() {
						var cn, st, loc sql.NullString
						if err := hRows.Scan(&cn, &st, &loc); err == nil {
							holdings = append(holdings, z3950.Holding{
								CallNumber: cn.String,
								Status:     st.String,
								Location:   loc.String,
							})
						}
					}
				}
				rec.Holdings = holdings
			}
			records = append(records, rec)
		}
	}
	return records, nil
}

func (p *SQLiteProvider) Scan(db, field, startTerm string, opts z3950.ScanOptions) ([]ScanResult, error) {
	var sqlStr string
	if field == "author" {
		sqlStr = `SELECT TRIM(author), 1 FROM bibliography WHERE author >= ? ORDER BY author ASC LIMIT 10`
	} else if field == "subject" {
		sqlStr = `SELECT TRIM(subjects), 1 FROM bibliography WHERE subjects >= ? ORDER BY subjects ASC LIMIT 10`
	} else {
		// Default to title
		sqlStr = `SELECT TRIM(title), 1 FROM bibliography WHERE title >= ? ORDER BY title ASC LIMIT 10`
	}

	rows, err := p.db.Query(sqlStr, startTerm)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []ScanResult
	for rows.Next() {
		var term string
		var count int
		if err := rows.Scan(&term, &count); err == nil {
			results = append(results, ScanResult{Term: term, Count: count})
		}
	}
	return results, nil
}

func (p *SQLiteProvider) CreateILLRequest(req ILLRequest) error {
	sqlStr := `INSERT INTO ill_requests (target_db, record_id, title, author, isbn, status, requestor, comments) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := p.db.Exec(sqlStr, req.TargetDB, req.RecordID, req.Title, req.Author, req.ISBN, req.Status, req.Requestor, req.Comments)
	return err
}

func (p *SQLiteProvider) GetILLRequest(id int64) (*ILLRequest, error) {
	var r ILLRequest
	var comments sql.NullString
	err := p.db.QueryRow("SELECT id, target_db, record_id, title, author, isbn, status, requestor, comments FROM ill_requests WHERE id = ?", id).
		Scan(&r.ID, &r.TargetDB, &r.RecordID, &r.Title, &r.Author, &r.ISBN, &r.Status, &r.Requestor, &comments)
	if err != nil {
		return nil, err
	}
	if comments.Valid {
		r.Comments = comments.String
	}
	return &r, nil
}

func (p *SQLiteProvider) ListILLRequests() ([]ILLRequest, error) {
	rows, err := p.db.Query("SELECT id, target_db, record_id, title, author, isbn, status, requestor, comments FROM ill_requests ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var requests []ILLRequest
	for rows.Next() {
		var r ILLRequest
		var comments sql.NullString
		if err := rows.Scan(&r.ID, &r.TargetDB, &r.RecordID, &r.Title, &r.Author, &r.ISBN, &r.Status, &r.Requestor, &comments); err != nil {
			return nil, err
		}
		if comments.Valid {
			r.Comments = comments.String
		}
		requests = append(requests, r)
	}
	return requests, nil
}

func (p *SQLiteProvider) UpdateILLRequestStatus(id int64, status string) error {
	_, err := p.db.Exec("UPDATE ill_requests SET status = ? WHERE id = ?", status, id)
	return err
}

// --- Cataloging Implementation ---

func (p *SQLiteProvider) CreateRecord(db string, record *z3950.MARCRecord) (string, error) {
	marcData := z3950.BuildMARC(p.profile, "", 
		record.GetTitle(p.profile), 
		record.GetAuthor(p.profile), 
		record.GetISBN(p.profile), 
		record.GetPublisher(p.profile), 
		record.GetPubYear(p.profile), 
		record.GetISSN(p.profile), 
		record.GetSubject(p.profile))

	res, err := p.db.Exec(`
		INSERT INTO bibliography (title, author, isbn, publisher, pub_year, issn, subjects, raw_record, raw_record_format)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		record.GetTitle(p.profile),
		record.GetAuthor(p.profile),
		record.GetISBN(p.profile),
		record.GetPublisher(p.profile),
		record.GetPubYear(p.profile),
		record.GetISSN(p.profile),
		record.GetSubject(p.profile),
		string(marcData),
		"MARC21",
	)
	if err != nil {
		return "", err
	}

	lastID, _ := res.LastInsertId()
	return fmt.Sprintf("%d", lastID), nil
}

func (p *SQLiteProvider) UpdateRecord(db string, id string, record *z3950.MARCRecord) error {
	marcData := z3950.BuildMARC(p.profile, id, 
		record.GetTitle(p.profile), 
		record.GetAuthor(p.profile), 
		record.GetISBN(p.profile), 
		record.GetPublisher(p.profile), 
		record.GetPubYear(p.profile), 
		record.GetISSN(p.profile), 
		record.GetSubject(p.profile))

	_, err := p.db.Exec(`
		UPDATE bibliography SET 
			title = ?, author = ?, isbn = ?, publisher = ?, pub_year = ?, issn = ?, subjects = ?, raw_record = ?
		WHERE CAST(id AS TEXT) = ?`,
		record.GetTitle(p.profile),
		record.GetAuthor(p.profile),
		record.GetISBN(p.profile),
		record.GetPublisher(p.profile),
		record.GetPubYear(p.profile),
		record.GetISSN(p.profile),
		record.GetSubject(p.profile),
		string(marcData),
		id,
	)
	return err
}

// --- Item & Circulation Implementation ---

func (p *SQLiteProvider) CreateItem(bibID string, item Item) error {
	_, err := p.db.Exec("INSERT INTO items (bib_id, barcode, call_number, status, location) VALUES (?, ?, ?, ?, ?)",
		bibID, item.Barcode, item.CallNumber, "Available", item.Location)
	return err
}

func (p *SQLiteProvider) GetItems(bibID string) ([]Item, error) {
	rows, err := p.db.Query("SELECT id, bib_id, barcode, call_number, status, location FROM items WHERE bib_id = ?", bibID)
	if err != nil { return nil, err }
	defer rows.Close()

	var items []Item
	for rows.Next() {
		var i Item
		if err := rows.Scan(&i.ID, &i.BibID, &i.Barcode, &i.CallNumber, &i.Status, &i.Location); err != nil {
			continue
		}
		items = append(items, i)
	}
	return items, nil
}

func (p *SQLiteProvider) GetItemByBarcode(barcode string) (*Item, error) {
	var i Item
	err := p.db.QueryRow("SELECT id, bib_id, barcode, call_number, status, location FROM items WHERE barcode = ?", barcode).
		Scan(&i.ID, &i.BibID, &i.Barcode, &i.CallNumber, &i.Status, &i.Location)
	if err != nil { return nil, err }
	return &i, nil
}

func (p *SQLiteProvider) Checkout(itemBarcode, patronID string) (string, error) {
	item, err := p.GetItemByBarcode(itemBarcode)
	if err != nil { return "", fmt.Errorf("item not found") }
	if item.Status != "Available" { return "", fmt.Errorf("item is already checked out or not available") }

	dueDate := time.Now().Add(30 * 24 * time.Hour)
	dueDateStr := dueDate.Format(time.RFC3339)

	tx, err := p.db.Begin()
	if err != nil { return "", err }

	_, err = tx.Exec("INSERT INTO loans (item_id, patron_id, due_date, status) VALUES (?, ?, ?, ?)", 
		item.ID, patronID, dueDate, "active")
	if err != nil { tx.Rollback(); return "", err }

	_, err = tx.Exec("UPDATE items SET status = 'Checked Out' WHERE id = ?", item.ID)
	if err != nil { tx.Rollback(); return "", err }

	return dueDateStr, tx.Commit()
}

func (p *SQLiteProvider) Checkin(itemBarcode string) (float64, error) {
	item, err := p.GetItemByBarcode(itemBarcode)
	if err != nil { return 0, fmt.Errorf("item not found") }
	if item.Status == "Available" { return 0, fmt.Errorf("item is not checked out") }

	var loanID int64
	var dueDate time.Time
	err = p.db.QueryRow("SELECT id, due_date FROM loans WHERE item_id = ? AND status = 'active'", item.ID).Scan(&loanID, &dueDate)
	if err != nil { return 0, fmt.Errorf("no active loan found for item") }

	fine := 0.0
	if time.Now().After(dueDate) {
		days := int(time.Since(dueDate).Hours() / 24)
		if days > 0 { fine = float64(days) * 0.50 }
	}

	tx, err := p.db.Begin()
	if err != nil { return 0, err }

	_, err = tx.Exec("UPDATE loans SET return_date = ?, status = 'returned' WHERE id = ?", time.Now(), loanID)
	if err != nil { tx.Rollback(); return 0, err }

	_, err = tx.Exec("UPDATE items SET status = 'Available' WHERE id = ?", item.ID)
	if err != nil { tx.Rollback(); return 0, err }

	return fine, tx.Commit()
}

func (p *SQLiteProvider) GetDashboardStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// 1. Total Titles
	var titles int
	p.db.QueryRow("SELECT COUNT(*) FROM bibliography").Scan(&titles)
	stats["total_titles"] = titles

	// 2. Total Items
	var items int
	p.db.QueryRow("SELECT COUNT(*) FROM items").Scan(&items)
	stats["total_items"] = items

	// 3. Active Loans
	var activeLoans int
	p.db.QueryRow("SELECT COUNT(*) FROM loans WHERE status = 'active'").Scan(&activeLoans)
	stats["active_loans"] = activeLoans

	// 4. Overdue
	var overdue int
	p.db.QueryRow("SELECT COUNT(*) FROM loans WHERE status = 'active' AND due_date < ?", time.Now()).Scan(&overdue)
	stats["overdue_loans"] = overdue

	return stats, nil
}

func (p *SQLiteProvider) CreateUser(user *User) error {
	_, err := p.db.Exec("INSERT INTO users (username, password_hash, role) VALUES (?, ?, ?)", user.Username, user.PasswordHash, user.Role)
	return err
}

func (p *SQLiteProvider) GetUserByUsername(username string) (*User, error) {
	var user User
	err := p.db.QueryRow("SELECT id, username, password_hash, role FROM users WHERE username = ?", username).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.Role)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (p *SQLiteProvider) CreateTarget(target *Target) error {
	_, err := p.db.Exec("INSERT INTO targets (name, host, port, database_name, encoding, auth_user, auth_pass) VALUES (?, ?, ?, ?, ?, ?, ?)",
		target.Name, target.Host, target.Port, target.DatabaseName, target.Encoding, target.AuthUser, target.AuthPass)
	return err
}

func (p *SQLiteProvider) ListTargets() ([]Target, error) {
	rows, err := p.db.Query("SELECT id, name, host, port, database_name, encoding, auth_user, auth_pass FROM targets ORDER BY name ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var targets []Target
	for rows.Next() {
		var t Target
		var user, pass sql.NullString
		if err := rows.Scan(&t.ID, &t.Name, &t.Host, &t.Port, &t.DatabaseName, &t.Encoding, &user, &pass); err != nil {
			return nil, err
		}
		if user.Valid { t.AuthUser = user.String }
		if pass.Valid { t.AuthPass = pass.String }
		targets = append(targets, t)
	}
	return targets, nil
}

func (p *SQLiteProvider) DeleteTarget(id int64) error {
	_, err := p.db.Exec("DELETE FROM targets WHERE id = ?", id)
	return err
}

func (p *SQLiteProvider) GetTargetByName(name string) (*Target, error) {
	var t Target
	var user, pass sql.NullString
	err := p.db.QueryRow("SELECT id, name, host, port, database_name, encoding, auth_user, auth_pass FROM targets WHERE name = ?", name).
		Scan(&t.ID, &t.Name, &t.Host, &t.Port, &t.DatabaseName, &t.Encoding, &user, &pass)
	if err != nil {
		return nil, err
	}
	if user.Valid { t.AuthUser = user.String }
	if pass.Valid { t.AuthPass = pass.String }
	return &t, nil
}
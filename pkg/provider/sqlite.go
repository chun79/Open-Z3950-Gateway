package provider

import (
	"database/sql"
	"fmt"
	"os"
	"strings"

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

	// Ensure holdings table exists
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

	// Seed holdings if empty
	var holdingCount int
	db.QueryRow("SELECT COUNT(*) FROM holdings").Scan(&holdingCount)
	if holdingCount == 0 {
		// Just random seed for existing bibliographies (1-4)
		seedHoldings := []string{
			"INSERT INTO holdings (bib_id, call_number, status, location) VALUES (1, 'QA76.73.G63 D66 2015', 'Available', 'Main Library')",
			"INSERT INTO holdings (bib_id, call_number, status, location) VALUES (1, 'QA76.73.G63 D66 2015 c.2', 'Checked Out', 'Main Library')",
			"INSERT INTO holdings (bib_id, call_number, status, location) VALUES (2, 'QA76.73.G63 P55 2018', 'Available', 'Science Branch')",
			"INSERT INTO holdings (bib_id, call_number, status, location) VALUES (3, 'QA76.73.G63 B88 2016', 'Lost', 'Main Library')",
			"INSERT INTO holdings (bib_id, call_number, status, location) VALUES (4, 'QA76.9.A25 S74 2020', 'Available', 'Engineering Lib')",
		}
		for _, q := range seedHoldings {
			db.Exec(q)
		}
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
				hRows, err := p.db.Query("SELECT id, bib_id, call_number, status, location FROM holdings WHERE bib_id = ?", id.String)
				if err == nil {
					defer hRows.Close()
					var holdings []z3950.Holding
					for hRows.Next() {
						var h z3950.Holding
						// Note: z3950 package doesn't have Holding struct, we defined it in provider.
						// Wait, Fetch returns []*z3950.MARCRecord.
						// MARCRecord doesn't have Holdings field.
						// The interface Fetch returns []*z3950.MARCRecord.
						// We need to attach holdings to MARCRecord or change the return type.
						// Changing return type breaks everything.
						// Best approach: Add Holdings field to z3950.MARCRecord struct.
						
						var hid int64
						var bid, call, stat, loc string
						if err := hRows.Scan(&hid, &bid, &call, &stat, &loc); err == nil {
							h = z3950.Holding{
								CallNumber: call,
								Status:     stat,
								Location:   loc,
							}
							holdings = append(holdings, h)
						}
					}
					rec.Holdings = holdings
				}
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
	// 1. Build MARC binary from fields if not present, or use raw
	// For simplicity, let's assume we build it from the record fields to keep it fresh
	marcData := z3950.BuildMARC(p.profile, "", 
		record.GetTitle(p.profile), 
		record.GetAuthor(p.profile), 
		record.GetISBN(p.profile), 
		record.GetPublisher(p.profile), 
		record.GetPubYear(p.profile), 
		record.GetISSN(p.profile), 
		record.GetSubject(p.profile))

	// 2. Insert into DB
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
		"MARC21", // Or p.profile.Name
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

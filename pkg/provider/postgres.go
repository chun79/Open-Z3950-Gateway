package provider

import (
	"database/sql"
	"fmt"
	"os"
	"strings"

	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
	"github.com/yourusername/open-z3950-gateway/pkg/z3950"
)

type PostgresProvider struct {
	db       *sql.DB
	profile  *z3950.MARCProfile
	tableMap map[string]string
}

func NewPostgresProvider(dsn string) (*PostgresProvider, error) {
	if dsn == "" {
		return nil, fmt.Errorf("postgres provider requires a non-empty DSN")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open postgres db: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping postgres db: %w", err)
	}

	// Init Tables
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS ill_requests (
			id SERIAL PRIMARY KEY,
			target_db TEXT,
			record_id TEXT,
			title TEXT,
			author TEXT,
			isbn TEXT,
			status TEXT,
			requestor TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		return nil, fmt.Errorf("failed to create ill_requests table: %w", err)
	}

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			username TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			role TEXT DEFAULT 'user',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		return nil, fmt.Errorf("failed to create users table: %w", err)
	}

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS targets (
			id SERIAL PRIMARY KEY,
			name TEXT UNIQUE NOT NULL,
			host TEXT NOT NULL,
			port INTEGER NOT NULL,
			database_name TEXT NOT NULL,
			encoding TEXT DEFAULT 'MARC21',
			auth_user TEXT,
			auth_pass TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		return nil, fmt.Errorf("failed to create targets table: %w", err)
	}

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS bibliography (
			id SERIAL PRIMARY KEY,
			title TEXT,
			author TEXT,
			isbn TEXT,
			publisher TEXT,
			pub_year TEXT,
			issn TEXT,
			subjects TEXT,
			raw_record TEXT,
			raw_record_format TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		return nil, fmt.Errorf("failed to create bibliography table: %w", err)
	}

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS shared_bibliography (
			id SERIAL PRIMARY KEY,
			title TEXT,
			author TEXT,
			isbn TEXT,
			publisher TEXT,
			pub_year TEXT,
			issn TEXT,
			subjects TEXT,
			raw_record TEXT,
			raw_record_format TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		return nil, fmt.Errorf("failed to create shared_bibliography table: %w", err)
	}

	// Seed bibliography
	var bibCount int
	db.QueryRow("SELECT COUNT(*) FROM bibliography").Scan(&bibCount)
	if bibCount == 0 {
		db.Exec("INSERT INTO bibliography (title, author, isbn, publisher, pub_year, subjects) VALUES ($1, $2, $3, $4, $5, $6)",
			"Thinking in Go", "Rob Pike", "0201548550", "Addison-Wesley", "2012", "Programming")
		db.Exec("INSERT INTO bibliography (title, author, isbn, publisher, pub_year, subjects) VALUES ($1, $2, $3, $4, $5, $6)",
			"Z39.50 for Dummies", "Index Data", "1234567890", "Dummy Press", "1999", "Library Science")
	}

	// Seed admin user
	var userCount int
	db.QueryRow("SELECT COUNT(*) FROM users WHERE username = 'admin'").Scan(&userCount)
	if userCount == 0 {
		hash, _ := bcrypt.GenerateFromPassword([]byte("admin"), bcrypt.DefaultCost)
		db.Exec("INSERT INTO users (username, password_hash, role) VALUES ($1, $2, $3)", "admin", string(hash), "admin")
	}

	// Seed default targets
	var targetCount int
	db.QueryRow("SELECT COUNT(*) FROM targets").Scan(&targetCount)
	if targetCount == 0 {
		seedTargets := []Target{
			{Name: "LCDB", Host: "lx2.loc.gov", Port: 210, DatabaseName: "LCDB", Encoding: "MARC21"},
			{Name: "IndexData", Host: "z3950.indexdata.com", Port: 210, DatabaseName: "gils", Encoding: "MARC21"},
			{Name: "Oxford", Host: "z3950.ox.ac.uk", Port: 210, DatabaseName: "OLIS", Encoding: "MARC21"},
			{Name: "Harvard", Host: "hollis.harvard.edu", Port: 210, DatabaseName: "Hollie", Encoding: "MARC21"},
			{Name: "Yale", Host: "orbis.library.yale.edu", Port: 210, DatabaseName: "Orbis", Encoding: "MARC21"},
		}
		for _, t := range seedTargets {
			db.Exec("INSERT INTO targets (name, host, port, database_name, encoding) VALUES ($1, $2, $3, $4, $5)", t.Name, t.Host, t.Port, t.DatabaseName, t.Encoding)
		}
	}

	format := os.Getenv("ZSERVER_MARC_FORMAT")
	profile := &z3950.ProfileMARC21
	if format == "CNMARC" {
		profile = &z3950.ProfileCNMARC
	} else if format == "UNIMARC" {
		profile = &z3950.ProfileUNIMARC
	}

	return &PostgresProvider{
		db:      db,
		profile: profile,
		tableMap: map[string]string{
			"Default": "bibliography", "LCDB": "bibliography", "CNMARC": "bibliography",
			"Shared":  "shared_bibliography", "Union": "shared_bibliography",
			"Kids": "bibliography_kids", "Archive": "bibliography_archive",
		},
	}, nil
}

func (p *PostgresProvider) getTable(db string) string {
	if table, ok := p.tableMap[db]; ok {
		return table
	}
	return "bibliography"
}

func (p *PostgresProvider) mapAttribute(attr int) string {
	switch attr {
	case z3950.UseAttributeTitle:
		return "title"
	case z3950.UseAttributeAuthor:
		return "author"
	case z3950.UseAttributeISBN:
		return "isbn"
	case z3950.UseAttributeISSN:
		return "issn"
	case z3950.UseAttributeDatePub:
		return "pub_year"
	case z3950.UseAttributeSubject:
		return "subjects"
	case z3950.UseAttributeAny:
		// Using a special value to indicate a full-text-like search
		return "__any__"
	default:
		// Fallback to searching in title for unknown attributes
		return "title"
	}
}

// buildSQL recursively builds WHERE clause and args from QueryNode
func (p *PostgresProvider) buildSQL(node z3950.QueryNode, argCounter *int) (string, []interface{}, error) {
	if node == nil {
		// Return a clause that is always true and consumes no args
		return "1 = 1", []interface{}{}, nil
	}

	switch n := node.(type) {
	case z3950.QueryClause:
		colName := p.mapAttribute(n.Attribute)
		term := n.Term

		if colName == "isbn" {
			*argCounter++
			// Special handling for ISBN: clean and exact match
			return fmt.Sprintf("REGEXP_REPLACE(%s, '[^0-9xX]', '', 'g') = $%d", colName, *argCounter), []interface{}{CleanISBN(term)}, nil
		}

		if colName == "__any__" {
			// Handle 'Any' by searching across title and author and subjects
			*argCounter++
			arg1 := *argCounter
			*argCounter++
			arg2 := *argCounter
			*argCounter++
			arg3 := *argCounter
			searchTerm := "%" + strings.ToLower(term) + "%"
			return fmt.Sprintf("(LOWER(title) LIKE $%d OR LOWER(author) LIKE $%d OR LOWER(subjects) LIKE $%d)", arg1, arg2, arg3), []interface{}{searchTerm, searchTerm, searchTerm}, nil
		}

		// Default case: simple LIKE search
		*argCounter++
		return fmt.Sprintf("LOWER(%s) LIKE $%d", colName, *argCounter), []interface{}{"%" + strings.ToLower(term) + "%"}, nil

	case z3950.QueryComplex:
		lSql, lArgs, err := p.buildSQL(n.Left, argCounter)
		if err != nil {
			return "", nil, err
		}

		rSql, rArgs, err := p.buildSQL(n.Right, argCounter)
		if err != nil {
			return "", nil, err
		}

		op := "AND"
		if n.Operator == "OR" {
			op = "OR"
		} else if n.Operator == "AND-NOT" {
			op = "AND NOT"
		}

		return fmt.Sprintf("(%s %s %s)", lSql, op, rSql), append(lArgs, rArgs...), nil
	}
	return "", nil, fmt.Errorf("unknown query node type: %T", node)
}

func (p *PostgresProvider) Search(db string, query z3950.StructuredQuery) ([]string, error) {
	if query.Root == nil {
		return nil, nil
	}

	table := p.getTable(db)

	argCounter := 0
	whereClause, args, err := p.buildSQL(query.Root, &argCounter)
	if err != nil {
		return nil, fmt.Errorf("failed to build SQL query: %w", err)
	}

	limit := 100
	if query.Limit > 0 {
		limit = query.Limit
	}
	offset := 0
	if query.Offset > 0 {
		offset = query.Offset
	}

	// Append LIMIT and OFFSET to the query and arguments
	sqlStr := fmt.Sprintf(`SELECT CAST(id AS VARCHAR) FROM %s WHERE %s ORDER BY id LIMIT $%d OFFSET $%d`,
		table, whereClause, argCounter+1, argCounter+2)

	finalArgs := append(args, limit, offset)

	rows, err := p.db.Query(sqlStr, finalArgs...)
	if err != nil {
		return nil, fmt.Errorf("dynamic postgres query failed: %w. SQL: %s. Args: %v", err, sqlStr, finalArgs)
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

func (p *PostgresProvider) Fetch(db string, ids []string) ([]*z3950.MARCRecord, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	table := p.getTable(db)
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}
	query := fmt.Sprintf(`SELECT id, title, author, isbn, publisher, pub_year, issn, subjects, raw_record, raw_record_format FROM %s WHERE CAST(id AS VARCHAR) IN (%s)`, table, strings.Join(placeholders, ","))
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
			if title.Valid {
				t = title.String
			}
			if author.Valid {
				a = author.String
			}
			if isbn.Valid {
				i = isbn.String
			}
			if publisher.Valid {
				pub = publisher.String
			}
			if pubYear.Valid {
				y = pubYear.String
			}
			if issn.Valid {
				iss = issn.String
			}
			if subjects.Valid {
				sub = subjects.String
			}
			if id.Valid {
				rid = id.String
			}
			bytes := z3950.BuildMARC(p.profile, rid, t, a, i, pub, y, iss, sub)
			rec, _ = z3950.ParseMARC(bytes)
		}
		if rec != nil {
			records = append(records, rec)
		}
	}
	return records, nil
}

func (p *PostgresProvider) Scan(db, field, startTerm string, opts z3950.ScanOptions) ([]ScanResult, error) {
	table := p.getTable(db)
	sqlStr := fmt.Sprintf(`SELECT title, 1 FROM %s WHERE title >= $1 ORDER BY title ASC LIMIT 10`, table)
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

func (p *PostgresProvider) CreateILLRequest(req ILLRequest) error {
	sqlStr := `INSERT INTO ill_requests (target_db, record_id, title, author, isbn, status, requestor, comments) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err := p.db.Exec(sqlStr, req.TargetDB, req.RecordID, req.Title, req.Author, req.ISBN, req.Status, req.Requestor, req.Comments)
	return err
}

func (p *PostgresProvider) GetILLRequest(id int64) (*ILLRequest, error) {
	var r ILLRequest
	var comments sql.NullString
	err := p.db.QueryRow("SELECT id, target_db, record_id, title, author, isbn, status, requestor, comments FROM ill_requests WHERE id = $1", id).
		Scan(&r.ID, &r.TargetDB, &r.RecordID, &r.Title, &r.Author, &r.ISBN, &r.Status, &r.Requestor, &comments)
	if err != nil {
		return nil, err
	}
	if comments.Valid {
		r.Comments = comments.String
	}
	return &r, nil
}

func (p *PostgresProvider) ListILLRequests() ([]ILLRequest, error) {
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

func (p *PostgresProvider) UpdateILLRequestStatus(id int64, status string) error {
	_, err := p.db.Exec("UPDATE ill_requests SET status = $1 WHERE id = $2", status, id)
	return err
}

func (p *PostgresProvider) CreateRecord(db string, record *z3950.MARCRecord) (string, error) {
	table := p.getTable(db)
	marcData := z3950.BuildMARC(p.profile, "", record.GetTitle(p.profile), record.GetAuthor(p.profile), record.GetISBN(p.profile), record.GetPublisher(p.profile), record.GetPubYear(p.profile), record.GetISSN(p.profile), record.GetSubject(p.profile))

	var lastID int64
	err := p.db.QueryRow(fmt.Sprintf(`
		INSERT INTO %s (title, author, isbn, publisher, pub_year, issn, subjects, raw_record, raw_record_format)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id`, table),
		record.GetTitle(p.profile), record.GetAuthor(p.profile), record.GetISBN(p.profile),
		record.GetPublisher(p.profile), record.GetPubYear(p.profile), record.GetISSN(p.profile),
		record.GetSubject(p.profile), string(marcData), "MARC21",
	).Scan(&lastID)

	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d", lastID), nil
}

func (p *PostgresProvider) UpdateRecord(db string, id string, record *z3950.MARCRecord) error {
	table := p.getTable(db)
	marcData := z3950.BuildMARC(p.profile, id, record.GetTitle(p.profile), record.GetAuthor(p.profile), record.GetISBN(p.profile), record.GetPublisher(p.profile), record.GetPubYear(p.profile), record.GetISSN(p.profile), record.GetSubject(p.profile))

	_, err := p.db.Exec(fmt.Sprintf(`
		UPDATE %s SET title = $1, author = $2, isbn = $3, publisher = $4, pub_year = $5, issn = $6, subjects = $7, raw_record = $8
		WHERE CAST(id AS VARCHAR) = $9`, table),
		record.GetTitle(p.profile), record.GetAuthor(p.profile), record.GetISBN(p.profile),
		record.GetPublisher(p.profile), record.GetPubYear(p.profile), record.GetISSN(p.profile),
		record.GetSubject(p.profile), string(marcData), id,
	)
	return err
}

func (p *PostgresProvider) CreateItem(bibID string, item Item) error { return fmt.Errorf("not implemented") }
func (p *PostgresProvider) GetItems(bibID string) ([]Item, error) { return nil, fmt.Errorf("not implemented") }
func (p *PostgresProvider) GetItemByBarcode(barcode string) (*Item, error) { return nil, fmt.Errorf("not implemented") }
func (p *PostgresProvider) Checkout(itemBarcode, patronID string) (string, error) { return "", fmt.Errorf("not implemented") }
func (p *PostgresProvider) Checkin(itemBarcode string) (float64, error) { return 0, fmt.Errorf("not implemented") }

func (p *PostgresProvider) CreateUser(user *User) error {
	_, err := p.db.Exec("INSERT INTO users (username, password_hash, role) VALUES ($1, $2, $3)", user.Username, user.PasswordHash, user.Role)
	return err
}

func (p *PostgresProvider) GetUserByUsername(username string) (*User, error) {
	var user User
	err := p.db.QueryRow("SELECT id, username, password_hash, role FROM users WHERE username = $1", username).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.Role)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (p *PostgresProvider) CreateTarget(target *Target) error {
	_, err := p.db.Exec("INSERT INTO targets (name, host, port, database_name, encoding, auth_user, auth_pass) VALUES ($1, $2, $3, $4, $5, $6, $7)",
		target.Name, target.Host, target.Port, target.DatabaseName, target.Encoding, target.AuthUser, target.AuthPass)
	return err
}

func (p *PostgresProvider) ListTargets() ([]Target, error) {
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

func (p *PostgresProvider) DeleteTarget(id int64) error {
	_, err := p.db.Exec("DELETE FROM targets WHERE id = $1", id)
	return err
}

func (p *PostgresProvider) GetTargetByName(name string) (*Target, error) {
	var t Target
	var user, pass sql.NullString
	err := p.db.QueryRow("SELECT id, name, host, port, database_name, encoding, auth_user, auth_pass FROM targets WHERE name = $1", name).
		Scan(&t.ID, &t.Name, &t.Host, &t.Port, &t.DatabaseName, &t.Encoding, &user, &pass)
	if err != nil {
		return nil, err
	}
	if user.Valid { t.AuthUser = user.String }
	if pass.Valid { t.AuthPass = pass.String }
	return &t, nil
}

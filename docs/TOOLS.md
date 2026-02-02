# Administrative & Debugging Tools

This directory contains various utility scripts located in `cmd/tools/` to help administrators and developers manage the database, debug connections, and inspect data.

## Usage

You can run any of these tools using `go run`. Ensure you have the necessary environment variables set (e.g., `DB_DSN` for database tools).

```bash
# Example: Check database connection and sample data
export DB_DSN="postgres://user:password@localhost:5432/library?sslmode=disable"
go run cmd/tools/check_db.go
```

## Available Tools

### Database Inspection

| Tool | Description |
| :--- | :--- |
| `check_db.go` | Connects to the configured Postgres database (`DB_DSN`), performs a broad search (`%`), and displays the first 3 records with their rich metadata (Title, Publisher). |
| `check_shared.go` | Inspects the `shared_bibliography` table schema and row count in Postgres. Useful for verifying database migrations. |
| `inspect_data.go` | General data inspection tool (likely for checking raw MARC/records). |
| `inspect_one.go` | Fetches and displays details for a single record. |
| `list_tables.go` | Lists all tables in the database to verify schema initialization. |
| `full_census.go` | Performs a full count of records in the database. |

### Connection & Z39.50 Debugging

| Tool | Description |
| :--- | :--- |
| `verify_client.go` | Acts as a standalone Z39.50 client to test connectivity against an external server (defaults to Index Data's public server). Checks Connect, Init, Search, and Present flows. |
| `debug_conn.go` | Low-level connection debugger. |
| `debug_search.go` | performing debug searches against a target. |
| `probe_conn.go` | Probes a Z39.50 connection for liveness/latency. |

### Utilities

| Tool | Description |
| :--- | :--- |
| `find_isbn.go` | Search for a specific ISBN. |

## Environment Variables

*   `DB_DSN`: Postgres connection string (Required for DB tools).

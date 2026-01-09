# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build Commands

```bash
# Build the binary
go build -o database-mcp main.go

# Run tests
go test -v ./...

# Format code
go fmt ./...

# Vet code
go vet ./...
```

## Configuration

Environment variables:
- `DB_DRIVER`: Database driver (`sqlserver`, `postgres`, `mysql`, `godror`, `sqlite3`)
- `DB_CONNECTION_STRING`: Connection string (required)

## Architecture

This is an MCP (Model Context Protocol) server that provides database access to Claude. It uses the `github.com/mark3labs/mcp-go` framework.

### Core Components

**Entry Point** (`main.go`): Initializes database connection, creates QueryBuilder, starts MCP stdio server.

**DatabaseMCP** (`mcp/struct.go`): Central struct holding MCP server, database connection pool, and QueryBuilder.

**QueryBuilder** (`mcp/query_builder.go`): Abstracts database-specific SQL syntax (parameter placeholders, pagination, string concatenation) across SQL Server, PostgreSQL, MySQL, Oracle, and SQLite.

**Database Connection** (`mcp/database.go`): Manages connection pool (25 max open, 5 max idle, 5min lifetime).

### Tool Registration Flow

`mcp/mcp_tools.go` registers 16 database tools:
- Query: `execute_query`
- Tables: `list_tables`, `describe_table`, `list_table_rows`, `get_table_schema_full`
- Procedures: `list_procedures`, `get_procedure_code`, `execute_procedure`
- Functions: `list_functions`, `get_function_code`
- Views: `list_views`, `get_view_definition`
- Triggers: `list_triggers`, `get_trigger_code`
- Utility: `search_objects`, `get_database_info`

Each tool type has its own file (`mcp/sql_*.go`).

### Security

`mcp/sql_query_validation.go` prevents SQL injection:
- Only SELECT/WITH queries allowed via `execute_query`
- Max query length: 10KB
- Limits on subqueries (10), UNIONs (5), nesting depth (20)
- Identifier validation for schema names

### Database Driver Differences

The QueryBuilder handles per-database variations:
- **Parameter placeholders**: `@p1` (SQL Server), `$1` (Postgres), `?` (MySQL/SQLite), `:1` (Oracle)
- **Pagination**: `OFFSET/FETCH` (SQL Server, Oracle) vs `LIMIT/OFFSET` (others)
- **Case-insensitive search**: `ILIKE` (Postgres) vs `LIKE` (others)

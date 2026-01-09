# Database MCP - Project Overview

## Purpose
MCP (Model Context Protocol) server that provides database access to Claude. It allows Claude to query and inspect databases through a standardized protocol.

## Supported Databases
- SQL Server (driver: `sqlserver`)
- PostgreSQL (driver: `postgres`)
- MySQL (driver: `mysql`)
- Oracle (driver: `godror`)
- SQLite (driver: `sqlite3`)

## Tech Stack
- **Language**: Go 1.24.3
- **MCP Framework**: github.com/mark3labs/mcp-go v0.43.2
- **Database Drivers**:
  - SQL Server: github.com/denisenkom/go-mssqldb
  - PostgreSQL: github.com/lib/pq
  - MySQL: github.com/go-sql-driver/mysql
  - Oracle: github.com/godror/godror
  - SQLite: github.com/mattn/go-sqlite3

## Configuration
Environment variables:
- `DB_DRIVER`: Database driver name (default: `sqlserver`)
- `DB_CONNECTION_STRING`: Connection string (required)

## Architecture

### Core Components
1. **Entry Point** (`main.go`): Initializes database connection, creates QueryBuilder, starts MCP stdio server
2. **DatabaseMCP** (`mcp/struct.go`): Central struct holding MCP server, database connection pool, and QueryBuilder
3. **QueryBuilder** (`mcp/query_builder.go`): Abstracts database-specific SQL syntax across all supported databases
4. **Database Connection** (`mcp/database.go`): Manages connection pool (25 max open, 5 max idle, 5min lifetime)

### Available Tools (16 total)
- Query: `execute_query`
- Tables: `list_tables`, `describe_table`, `list_table_rows`, `get_table_schema_full`
- Procedures: `list_procedures`, `get_procedure_code`, `execute_procedure`
- Functions: `list_functions`, `get_function_code`
- Views: `list_views`, `get_view_definition`
- Triggers: `list_triggers`, `get_trigger_code`
- Utility: `search_objects`, `get_database_info`

### Code Organization
```
main.go              - Entry point
mcp/
  struct.go          - Core struct definitions
  mcp.go             - MCP server setup
  mcp_tools.go       - Tool registration
  database.go        - Database connection management
  datasource.go      - Data source abstraction
  query_builder.go   - SQL dialect abstraction
  util.go            - Utility functions
  sql_query.go       - Query execution
  sql_query_validation.go - SQL injection prevention
  sql_table.go       - Table operations
  sql_procedure.go   - Procedure operations
  sql_function.go    - Function operations
  sql_view.go        - View operations
  sql_trigger.go     - Trigger operations
  sql_database.go    - Database info operations
```

### Security Features
- Only SELECT/WITH queries allowed via `execute_query`
- Max query length: 10KB
- Limits on subqueries (10), UNIONs (5), nesting depth (20)
- Identifier validation for schema names

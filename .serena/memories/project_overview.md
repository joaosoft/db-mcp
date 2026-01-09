# Database MCP (db-mcp) - Project Overview

## Purpose
MCP (Model Context Protocol) server that provides database access to Claude. It allows Claude to query and inspect databases through a standardized protocol.

## Supported Databases
- SQL Server (driver: `sqlserver`)
- PostgreSQL (driver: `postgres`)
- MySQL (driver: `mysql`)
- Oracle (driver: `oracle` or `godror`)
- SQLite (driver: `sqlite` or `sqlite3`)

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
The server supports two configuration methods:

### 1. Environment Variables (Static)
- `DB_DRIVER`: Database driver name (default: `sqlserver`)
- `DB_CONNECTION_STRING`: Connection string (optional - can be configured dynamically)

### 2. Dynamic Configuration (via MCP Tools)
Use `configure_datasource` tool to connect to databases at runtime.

## Architecture

### Core Components
1. **Entry Point** (`main.go`): Initializes MCP server, starts stdio server
2. **DatabaseMCP** (`mcp/struct.go`): Central struct holding MCP server, database connection pool, and QueryBuilder
3. **QueryBuilder** (`mcp/query_builder.go` + `query_builder_helpers.go`): Abstracts database-specific SQL syntax across all supported databases
4. **Connection Management** (`mcp/connection.go`): Creates database connections with connection pool settings
5. **Constants** (`mcp/constant.go`): All constants, errors, driver types, and default values

### Available Tools (21 total)
- **DataSource Management** (5): `configure_datasource`, `get_current_datasource`, `test_connection`, `disconnect_datasource`, `list_database_drivers`
- **Query** (1): `execute_query`
- **Tables** (4): `list_tables`, `describe_table`, `list_table_rows`, `get_table_schema_full`
- **Procedures** (3): `list_procedures`, `get_procedure_code`, `execute_procedure`
- **Functions** (2): `list_functions`, `get_function_code`
- **Views** (2): `list_views`, `get_view_definition`
- **Triggers** (2): `list_triggers`, `get_trigger_code`
- **Utility** (2): `search_objects`, `get_database_info`

### Code Organization
```
main.go                      - Entry point
mcp/
  struct.go                  - Core struct definitions and precompiled regexes
  constant.go                - Constants, errors, driver types
  mcp.go                     - MCP server setup
  mcp_tools.go               - Tool registration
  connection.go              - Database connection management
  util.go                    - Utility functions (pagination, validation, helpers)
  query_builder.go           - SQL dialect abstraction (queries)
  query_builder_helpers.go   - SQL dialect abstraction (helpers)
  query_validation.go        - SQL injection prevention
  tool_datasource.go         - DataSource management tools
  tool_query.go              - Query execution tool
  tool_table.go              - Table operations
  tool_procedure.go          - Procedure operations
  tool_function.go           - Function operations
  tool_view.go               - View operations
  tool_trigger.go            - Trigger operations
  tool_database.go           - Database info and search operations
```

### Security Features
- Only SELECT/WITH queries allowed via `execute_query`
- Max query length: 10KB
- Limits on subqueries (10), UNIONs (5), nesting depth (20)
- Identifier validation for schema/table names
- All tools verify connection before executing

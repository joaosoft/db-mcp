# db-mcp
[![Build Status](https://travis-ci.org/joaosoft/db-mcp.svg?branch=master)](https://travis-ci.org/joaosoft/db-mcp) | [![codecov](https://codecov.io/gh/joaosoft/db-mcp/branch/master/graph/badge.svg)](https://codecov.io/gh/joaosoft/db-mcp) | [![Go Report Card](https://goreportcard.com/badge/github.com/joaosoft/db-mcp)](https://goreportcard.com/report/github.com/joaosoft/db-mcp) | [![GoDoc](https://godoc.org/github.com/joaosoft/db-mcp?status.svg)](https://godoc.org/github.com/joaosoft/db-mcp)

MCP server for database access supporting multiple database engines (SQL Server, PostgreSQL, MySQL, Oracle, SQLite)

## Supported Databases

- **SQL Server** (driver: `sqlserver`)
- **PostgreSQL** (driver: `postgres`)
- **MySQL** (driver: `mysql`)
- **Oracle** (driver: `oracle`)
- **SQLite** (driver: `sqlite`)

## Configuration

The server supports two configuration methods:

### 1. Environment Variables (Static)

- `DB_DRIVER`: Database driver name (default: `sqlserver`)
- `DB_CONNECTION_STRING`: Database connection string (optional)

### 2. Dynamic Configuration (via MCP Tools)

Use the `configure_datasource` tool to connect to databases at runtime. This allows switching databases without restarting the server.

### Connection String Examples

**SQL Server:**
```bash
export DB_DRIVER=sqlserver
export DB_CONNECTION_STRING="sqlserver://username:password@localhost:1433?database=mydb"
```

**PostgreSQL:**
```bash
export DB_DRIVER=postgres
export DB_CONNECTION_STRING="postgres://username:password@localhost:5432/mydb?sslmode=disable"
```

**MySQL:**
```bash
export DB_DRIVER=mysql
export DB_CONNECTION_STRING="username:password@tcp(localhost:3306)/mydb"
```

**Oracle:**
```bash
export DB_DRIVER=oracle
export DB_CONNECTION_STRING="oracle://username:password@localhost:1521/servicename"
```

**SQLite:**
```bash
export DB_DRIVER=sqlite
export DB_CONNECTION_STRING="./mydb.sqlite"
```

## Available Tools

### DataSource Management
| Tool | Description |
|------|-------------|
| `configure_datasource` | Configure and connect to a database dynamically |
| `get_current_datasource` | Get information about the currently active connection |
| `test_connection` | Test a database connection without switching to it |
| `disconnect_datasource` | Disconnect from the current database |
| `list_database_drivers` | List all supported database drivers and connection string formats |

### Query Execution
| Tool | Description |
|------|-------------|
| `execute_query` | Execute a SELECT query (read-only) |

### Tables
| Tool | Description |
|------|-------------|
| `list_tables` | List database tables with pagination |
| `describe_table` | Get table structure (columns, types, constraints) |
| `list_table_rows` | List table rows with pagination and filters |
| `get_table_schema_full` | Get complete table schema including indexes and foreign keys |

### Stored Procedures
| Tool | Description |
|------|-------------|
| `list_procedures` | List stored procedures with pagination |
| `get_procedure_code` | Get the source code of a stored procedure |
| `execute_procedure` | Execute a stored procedure with parameters |

### Functions
| Tool | Description |
|------|-------------|
| `list_functions` | List database functions (scalar, table-valued) |
| `get_function_code` | Get the source code of a function |

### Views
| Tool | Description |
|------|-------------|
| `list_views` | List database views with pagination |
| `get_view_definition` | Get the SQL definition of a view |

### Triggers
| Tool | Description |
|------|-------------|
| `list_triggers` | List database triggers with pagination |
| `get_trigger_code` | Get the source code of a trigger |

### Utility
| Tool | Description |
|------|-------------|
| `search_objects` | Search for objects by name or in source code |
| `get_database_info` | Get general information about the database |

## Build

```bash
go build -o db-mcp main.go
```

## Usage Example

```
# First, list available drivers
> list_database_drivers

# Test a connection before switching
> test_connection(driver="postgres", connection_string="postgres://user:pass@localhost:5432/mydb")

# Connect to the database
> configure_datasource(driver="postgres", connection_string="postgres://user:pass@localhost:5432/mydb", name="my_postgres")

# Check current connection
> get_current_datasource

# Now use any database tool
> list_tables
> execute_query(query="SELECT * FROM users LIMIT 10")

# Disconnect when done
> disconnect_datasource
```

## Known Issues

## Follow me at
Facebook: https://www.facebook.com/joaosoft

LinkedIn: https://www.linkedin.com/in/jo%C3%A3o-ribeiro-b2775438/

##### If you have something to add, please let me know joaosoft@gmail.com

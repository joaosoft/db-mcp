# database-mcp
[![Build Status](https://travis-ci.org/joaosoft/database-mcp.svg?branch=master)](https://travis-ci.org/joaosoft/database-mcp) | [![codecov](https://codecov.io/gh/joaosoft/database-mcp/branch/master/graph/badge.svg)](https://codecov.io/gh/joaosoft/database-mcp) | [![Go Report Card](https://goreportcard.com/badge/github.com/joaosoft/database-mcp)](https://goreportcard.com/report/github.com/joaosoft/database-mcp) | [![GoDoc](https://godoc.org/github.com/joaosoft/database-mcp?status.svg)](https://godoc.org/github.com/joaosoft/database-mcp)

MCP server for database access supporting multiple database engines (SQL Server, PostgreSQL, MySQL, Oracle, SQLite)

## Supported Databases

- **SQL Server** (driver: `sqlserver`)
- **PostgreSQL** (driver: `postgres`)
- **MySQL** (driver: `mysql`)
- **Oracle** (driver: `godror`)
- **SQLite** (driver: `sqlite3`)

## Configuration

The server uses environment variables for database configuration:

- `DB_DRIVER`: Database driver name (default: `sqlserver`)
  - `sqlserver` - Microsoft SQL Server
  - `postgres` - PostgreSQL
  - `mysql` - MySQL/MariaDB
  - `godror` - Oracle Database
  - `sqlite3` - SQLite

- `DB_CONNECTION_STRING`: Database connection string (required)

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
export DB_DRIVER=godror
export DB_CONNECTION_STRING="user/password@localhost:1521/servicename"
# Or with connection descriptor:
export DB_CONNECTION_STRING="user/password@(DESCRIPTION=(ADDRESS=(PROTOCOL=TCP)(HOST=localhost)(PORT=1521))(CONNECT_DATA=(SERVICE_NAME=ORCL)))"
```

**SQLite:**
```bash
export DB_DRIVER=sqlite3
export DB_CONNECTION_STRING="./mydb.sqlite"
```

## Build
```bash
go build -o database-mcp main.go
```

## Known issues

## Follow me at
Facebook: https://www.facebook.com/joaosoft

LinkedIn: https://www.linkedin.com/in/jo%C3%A3o-ribeiro-b2775438/

##### If you have something to add, please let me know joaosoft@gmail.com

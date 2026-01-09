package mcp

import "time"

// Database connection pool configuration constants
const (
	DBMaxOpenConns    = 25
	DBMaxIdleConns    = 5
	DBConnMaxLifetime = 5 * time.Minute
	DBPingTimeout     = 5 * time.Second
)

// Query validation constants
const (
	MaxQueryLength       = 10000 // 10KB - reduced from 50KB for DoS prevention
	MaxSubqueryCount     = 10
	MaxUnionCount        = 5
	MaxParenthesesDepth  = 20
	MaxHexEncodingCount  = 3
	MaxCharFunctionCount = 10
)

// Pagination constants
const (
	DefaultPage     = 1
	DefaultPageSize = 100
	MaxPageSize     = 500
	MaxRowsPageSize = 1000
)

// Query timeout constants
const (
	DefaultQueryTimeout = 30 * time.Second
	ShortQueryTimeout   = 10 * time.Second
)

// Drivers
const (
	DriverSQLServer   DriverType = "sqlserver"
	DriverPostgresSQL DriverType = "postgres"
	DriverMySQL       DriverType = "mysql"
	DriverOracle      DriverType = "godror"
	DriverSQLite      DriverType = "sqlite3"
)

// Default schema per driver
const (
	DefaultSchemaSQLServer = "dbo"
	DefaultSchemaPostgres  = "public"
	DefaultSchemaMySQL     = ""
	DefaultSchemaOracle    = ""
	DefaultSchemaSQLite    = "main"
)

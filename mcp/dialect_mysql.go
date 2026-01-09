package mcp

import (
	"fmt"
	"strings"
)

// MySQLDialect implements Dialect for MySQL/MariaDB
type MySQLDialect struct {
	BaseDialect
}

// NewMySQLDialect creates a new MySQL dialect
func NewMySQLDialect() *MySQLDialect {
	return &MySQLDialect{
		BaseDialect: BaseDialect{driver: DriverMySQL},
	}
}

// Placeholder returns ?
func (d *MySQLDialect) Placeholder(index int) string {
	return "?"
}

// QuoteIdentifier returns `name`
func (d *MySQLDialect) QuoteIdentifier(name string) string {
	return fmt.Sprintf("`%s`", name)
}

// PaginationClause returns LIMIT/OFFSET syntax
// MySQL does not require ORDER BY for LIMIT/OFFSET to work
func (d *MySQLDialect) PaginationClause(limit, offset int, orderBy string) string {
	pagination := fmt.Sprintf("LIMIT %d OFFSET %d", limit, offset)
	if orderBy != "" {
		return fmt.Sprintf("ORDER BY %s %s", orderBy, pagination)
	}
	return pagination
}

// ConcatOperator returns CONCAT function
func (d *MySQLDialect) ConcatOperator(parts ...string) string {
	return "CONCAT(" + strings.Join(parts, ", ") + ")"
}

// CurrentDatabase returns DATABASE()
func (d *MySQLDialect) CurrentDatabase() string {
	return "DATABASE()"
}

// SystemSchemas returns MySQL system schemas
func (d *MySQLDialect) SystemSchemas() []string {
	return []string{"mysql", "information_schema", "performance_schema", "sys"}
}

// TableMetadata returns MySQL table metadata queries
func (d *MySQLDialect) TableMetadata() TableMetadataSQL {
	return TableMetadataSQL{
		ListTables: `
			SELECT
				TABLE_SCHEMA,
				TABLE_NAME,
				TABLE_TYPE
			FROM INFORMATION_SCHEMA.TABLES
			WHERE TABLE_TYPE = 'BASE TABLE'
				AND TABLE_SCHEMA NOT IN ('mysql', 'information_schema', 'performance_schema', 'sys')`,
		SchemaFilter: " AND TABLE_SCHEMA = ?",
		NameFilter:   " AND TABLE_NAME LIKE ?",
		OrderBy:      " ORDER BY TABLE_SCHEMA, TABLE_NAME",

		DescribeTable: `
			SELECT
				COLUMN_NAME,
				DATA_TYPE,
				IS_NULLABLE,
				COLUMN_DEFAULT,
				CHARACTER_MAXIMUM_LENGTH
			FROM INFORMATION_SCHEMA.COLUMNS
			WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?
			ORDER BY ORDINAL_POSITION`,

		TableExists: `SELECT COUNT(*) FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?`,

		GetColumns: `
			SELECT
				COLUMN_NAME,
				DATA_TYPE,
				CHARACTER_MAXIMUM_LENGTH,
				IS_NULLABLE,
				COLUMN_DEFAULT
			FROM INFORMATION_SCHEMA.COLUMNS
			WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?
			ORDER BY ORDINAL_POSITION`,

		GetFullSchema: `
			SELECT
				c.COLUMN_NAME,
				c.DATA_TYPE,
				c.CHARACTER_MAXIMUM_LENGTH,
				c.NUMERIC_PRECISION,
				c.NUMERIC_SCALE,
				c.IS_NULLABLE,
				c.COLUMN_DEFAULT,
				CASE WHEN pk.COLUMN_NAME IS NOT NULL THEN 'YES' ELSE 'NO' END AS IS_PRIMARY_KEY
			FROM INFORMATION_SCHEMA.COLUMNS c
			LEFT JOIN (
				SELECT ku.TABLE_SCHEMA, ku.TABLE_NAME, ku.COLUMN_NAME
				FROM INFORMATION_SCHEMA.TABLE_CONSTRAINTS tc
				JOIN INFORMATION_SCHEMA.KEY_COLUMN_USAGE ku
					ON tc.CONSTRAINT_TYPE = 'PRIMARY KEY'
					AND tc.CONSTRAINT_NAME = ku.CONSTRAINT_NAME
					AND tc.TABLE_SCHEMA = ku.TABLE_SCHEMA
					AND tc.TABLE_NAME = ku.TABLE_NAME
			) pk ON c.TABLE_SCHEMA = pk.TABLE_SCHEMA
				AND c.TABLE_NAME = pk.TABLE_NAME
				AND c.COLUMN_NAME = pk.COLUMN_NAME
			WHERE c.TABLE_SCHEMA = ? AND c.TABLE_NAME = ?
			ORDER BY c.ORDINAL_POSITION`,

		GetPrimaryKey: `
			SELECT ku.COLUMN_NAME
			FROM INFORMATION_SCHEMA.TABLE_CONSTRAINTS tc
			JOIN INFORMATION_SCHEMA.KEY_COLUMN_USAGE ku
				ON tc.CONSTRAINT_NAME = ku.CONSTRAINT_NAME
				AND tc.TABLE_SCHEMA = ku.TABLE_SCHEMA
				AND tc.TABLE_NAME = ku.TABLE_NAME
			WHERE tc.CONSTRAINT_TYPE = 'PRIMARY KEY'
				AND tc.TABLE_SCHEMA = ?
				AND tc.TABLE_NAME = ?
			ORDER BY ku.ORDINAL_POSITION`,

		GetIndexes: `
			SELECT
				INDEX_NAME AS index_name,
				INDEX_TYPE AS index_type,
				CASE WHEN NON_UNIQUE = 0 THEN 1 ELSE 0 END AS is_unique,
				COLUMN_NAME AS column_name
			FROM INFORMATION_SCHEMA.STATISTICS
			WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?
			ORDER BY INDEX_NAME, SEQ_IN_INDEX`,

		GetForeignKeys: `
			SELECT
				kcu.CONSTRAINT_NAME AS constraint_name,
				kcu.COLUMN_NAME AS column_name,
				kcu.REFERENCED_TABLE_SCHEMA AS referenced_schema,
				kcu.REFERENCED_TABLE_NAME AS referenced_table,
				kcu.REFERENCED_COLUMN_NAME AS referenced_column
			FROM INFORMATION_SCHEMA.KEY_COLUMN_USAGE kcu
			WHERE kcu.TABLE_SCHEMA = ?
				AND kcu.TABLE_NAME = ?
				AND kcu.REFERENCED_TABLE_NAME IS NOT NULL
			ORDER BY kcu.CONSTRAINT_NAME`,
	}
}

// ProcedureMetadata returns MySQL procedure metadata queries
func (d *MySQLDialect) ProcedureMetadata() ProcedureMetadataSQL {
	return ProcedureMetadataSQL{
		ListProcedures: `
			SELECT
				ROUTINE_SCHEMA as routine_schema,
				ROUTINE_NAME as routine_name,
				CREATED as created,
				LAST_ALTERED as last_altered
			FROM INFORMATION_SCHEMA.ROUTINES
			WHERE ROUTINE_TYPE = 'PROCEDURE'
				AND ROUTINE_SCHEMA NOT IN ('mysql', 'information_schema', 'performance_schema', 'sys')`,
		SchemaFilter: " AND ROUTINE_SCHEMA = ?",
		NameFilter:   " AND ROUTINE_NAME LIKE ?",
		OrderBy:      " ORDER BY ROUTINE_SCHEMA, ROUTINE_NAME",

		GetCode: `
			SELECT ROUTINE_DEFINITION
			FROM INFORMATION_SCHEMA.ROUTINES
			WHERE ROUTINE_SCHEMA = ? AND ROUTINE_NAME = ? AND ROUTINE_TYPE = 'PROCEDURE'`,
	}
}

// FunctionMetadata returns MySQL function metadata queries
func (d *MySQLDialect) FunctionMetadata() FunctionMetadataSQL {
	return FunctionMetadataSQL{
		ListFunctions: `
			SELECT
				ROUTINE_SCHEMA as routine_schema,
				ROUTINE_NAME as routine_name,
				'FUNCTION' as function_type,
				CREATED as created,
				LAST_ALTERED as last_altered
			FROM INFORMATION_SCHEMA.ROUTINES
			WHERE ROUTINE_TYPE = 'FUNCTION'
				AND ROUTINE_SCHEMA NOT IN ('mysql', 'information_schema', 'performance_schema', 'sys')`,
		TypeFilterScalar: "",
		TypeFilterTable:  "",
		TypeFilterAll:    "",
		SchemaFilter:     " AND ROUTINE_SCHEMA = ?",
		NameFilter:       " AND ROUTINE_NAME LIKE ?",
		OrderBy:          " ORDER BY ROUTINE_SCHEMA, ROUTINE_NAME",

		GetCode: `
			SELECT ROUTINE_DEFINITION
			FROM INFORMATION_SCHEMA.ROUTINES
			WHERE ROUTINE_SCHEMA = ? AND ROUTINE_NAME = ? AND ROUTINE_TYPE = 'FUNCTION'`,
	}
}

// ViewMetadata returns MySQL view metadata queries
func (d *MySQLDialect) ViewMetadata() ViewMetadataSQL {
	return ViewMetadataSQL{
		ListViews: `
			SELECT
				TABLE_SCHEMA as view_schema,
				TABLE_NAME as view_name,
				NULL as created,
				NULL as last_altered
			FROM INFORMATION_SCHEMA.VIEWS
			WHERE TABLE_SCHEMA NOT IN ('mysql', 'information_schema', 'performance_schema', 'sys')`,
		SchemaFilter: " AND TABLE_SCHEMA = ?",
		NameFilter:   " AND TABLE_NAME LIKE ?",
		OrderBy:      " ORDER BY TABLE_SCHEMA, TABLE_NAME",

		GetDefinition: `
			SELECT VIEW_DEFINITION
			FROM INFORMATION_SCHEMA.VIEWS
			WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?`,
	}
}

// TriggerMetadata returns MySQL trigger metadata queries
func (d *MySQLDialect) TriggerMetadata() TriggerMetadataSQL {
	return TriggerMetadataSQL{
		ListTriggers: `
			SELECT
				TRIGGER_SCHEMA as schema_name,
				TRIGGER_NAME as trigger_name,
				EVENT_OBJECT_TABLE as table_name,
				0 as is_disabled,
				CREATED as create_date,
				NULL as modify_date
			FROM INFORMATION_SCHEMA.TRIGGERS
			WHERE TRIGGER_SCHEMA NOT IN ('mysql', 'information_schema', 'performance_schema', 'sys')`,
		SchemaFilter:   " AND TRIGGER_SCHEMA = ?",
		TableFilter:    " AND EVENT_OBJECT_TABLE = ?",
		NameFilter:     " AND TRIGGER_NAME LIKE ?",
		DisabledFilter: "", // MySQL doesn't have disabled triggers
		OrderBy:        " ORDER BY TRIGGER_SCHEMA, EVENT_OBJECT_TABLE, TRIGGER_NAME",

		GetCode: `
			SELECT ACTION_STATEMENT
			FROM INFORMATION_SCHEMA.TRIGGERS
			WHERE TRIGGER_SCHEMA = ? AND TRIGGER_NAME = ?`,
	}
}

// DatabaseInfo returns MySQL database info queries
func (d *MySQLDialect) DatabaseInfo() DatabaseInfoSQL {
	return DatabaseInfoSQL{
		Version: "SELECT VERSION()",

		Details: `
			SELECT
				DATABASE() AS database_name,
				DEFAULT_COLLATION_NAME AS collation_name,
				'' AS recovery_model_desc,
				0 AS compatibility_level,
				NULL AS create_date
			FROM INFORMATION_SCHEMA.SCHEMATA
			WHERE SCHEMA_NAME = DATABASE()`,

		ObjectCounts: `
			SELECT
				SUM(CASE WHEN TABLE_TYPE = 'BASE TABLE' THEN 1 ELSE 0 END) AS tables,
				SUM(CASE WHEN TABLE_TYPE = 'VIEW' THEN 1 ELSE 0 END) AS views,
				(SELECT COUNT(*) FROM INFORMATION_SCHEMA.ROUTINES WHERE ROUTINE_TYPE = 'PROCEDURE' AND ROUTINE_SCHEMA = DATABASE()) AS procedures,
				(SELECT COUNT(*) FROM INFORMATION_SCHEMA.ROUTINES WHERE ROUTINE_TYPE = 'FUNCTION' AND ROUTINE_SCHEMA = DATABASE()) AS functions,
				(SELECT COUNT(*) FROM INFORMATION_SCHEMA.TRIGGERS WHERE TRIGGER_SCHEMA = DATABASE()) AS triggers
			FROM INFORMATION_SCHEMA.TABLES
			WHERE TABLE_SCHEMA = DATABASE()`,

		ListSchemas: `
			SELECT SCHEMA_NAME
			FROM INFORMATION_SCHEMA.SCHEMATA
			WHERE SCHEMA_NAME NOT IN ('mysql', 'information_schema', 'performance_schema', 'sys')
			ORDER BY SCHEMA_NAME`,

		SearchObjects: `
			SELECT
				TABLE_SCHEMA AS schema_name,
				TABLE_NAME AS object_name,
				TABLE_TYPE AS object_type,
				CREATE_TIME AS create_date,
				UPDATE_TIME AS modify_date,
				0 AS has_code
			FROM INFORMATION_SCHEMA.TABLES
			WHERE TABLE_SCHEMA NOT IN ('mysql', 'information_schema', 'performance_schema', 'sys')
			  AND TABLE_NAME LIKE CONCAT('%%', ?, '%%')
			  %s
			ORDER BY TABLE_SCHEMA, TABLE_NAME`,
	}
}

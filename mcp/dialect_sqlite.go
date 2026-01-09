package mcp

import (
	"fmt"
	"strings"
)

// SQLiteDialect implements Dialect for SQLite
type SQLiteDialect struct {
	BaseDialect
}

// NewSQLiteDialect creates a new SQLite dialect
func NewSQLiteDialect() *SQLiteDialect {
	return &SQLiteDialect{
		BaseDialect: BaseDialect{driver: DriverSQLite},
	}
}

// Placeholder returns ?
func (d *SQLiteDialect) Placeholder(index int) string {
	return "?"
}

// QuoteIdentifier returns "name"
func (d *SQLiteDialect) QuoteIdentifier(name string) string {
	return fmt.Sprintf(`"%s"`, name)
}

// PaginationClause returns LIMIT/OFFSET syntax
func (d *SQLiteDialect) PaginationClause(limit, offset int, orderBy string) string {
	pagination := fmt.Sprintf("LIMIT %d OFFSET %d", limit, offset)
	if orderBy != "" {
		return fmt.Sprintf("ORDER BY %s %s", orderBy, pagination)
	}
	return pagination
}

// ConcatOperator returns || concatenation
func (d *SQLiteDialect) ConcatOperator(parts ...string) string {
	return strings.Join(parts, " || ")
}

// CurrentDatabase returns 'main'
func (d *SQLiteDialect) CurrentDatabase() string {
	return "'main'"
}

// SystemSchemas returns empty (SQLite has no schemas)
func (d *SQLiteDialect) SystemSchemas() []string {
	return []string{}
}

// SupportsFeature checks SQLite feature support
func (d *SQLiteDialect) SupportsFeature(feature DialectFeature) bool {
	switch feature {
	case FeatureStoredProcedures:
		return false
	case FeatureFunctions:
		return false
	case FeatureSchemas:
		return false
	default:
		return true
	}
}

// TableMetadata returns SQLite table metadata queries
func (d *SQLiteDialect) TableMetadata() TableMetadataSQL {
	return TableMetadataSQL{
		ListTables: `
			SELECT
				'main' as table_schema,
				name as table_name,
				'BASE TABLE' as table_type
			FROM sqlite_master
			WHERE type = 'table'
				AND name NOT LIKE 'sqlite_%'`,
		SchemaFilter: "", // SQLite doesn't have schemas
		NameFilter:   " AND name LIKE ?",
		OrderBy:      " ORDER BY name",

		DescribeTable: "PRAGMA table_info(%s)",

		TableExists: `SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?`,

		GetColumns: "PRAGMA table_info(%s)",

		GetFullSchema: "PRAGMA table_info(%s)",

		GetPrimaryKey: "PRAGMA table_info(%s)",

		GetIndexes: "PRAGMA index_list(%s)",

		GetForeignKeys: "PRAGMA foreign_key_list(%s)",
	}
}

// ProcedureMetadata returns empty (SQLite doesn't support stored procedures)
func (d *SQLiteDialect) ProcedureMetadata() ProcedureMetadataSQL {
	return ProcedureMetadataSQL{}
}

// FunctionMetadata returns empty (SQLite doesn't support user functions in system tables)
func (d *SQLiteDialect) FunctionMetadata() FunctionMetadataSQL {
	return FunctionMetadataSQL{}
}

// ViewMetadata returns SQLite view metadata queries
func (d *SQLiteDialect) ViewMetadata() ViewMetadataSQL {
	return ViewMetadataSQL{
		ListViews: `
			SELECT
				'main' as view_schema,
				name as view_name,
				NULL as created,
				NULL as last_altered
			FROM sqlite_master
			WHERE type = 'view'`,
		SchemaFilter: "", // SQLite doesn't have schemas
		NameFilter:   " AND name LIKE ?",
		OrderBy:      " ORDER BY name",

		GetDefinition: `
			SELECT sql
			FROM sqlite_master
			WHERE type = 'view' AND name = ?`,
	}
}

// TriggerMetadata returns SQLite trigger metadata queries
func (d *SQLiteDialect) TriggerMetadata() TriggerMetadataSQL {
	return TriggerMetadataSQL{
		ListTriggers: `
			SELECT
				'main' as schema_name,
				name as trigger_name,
				tbl_name as table_name,
				0 as is_disabled,
				NULL as create_date,
				NULL as modify_date
			FROM sqlite_master
			WHERE type = 'trigger'`,
		SchemaFilter:   "", // SQLite doesn't have schemas
		TableFilter:    " AND tbl_name = ?",
		NameFilter:     " AND name LIKE ?",
		DisabledFilter: "", // SQLite doesn't have disabled triggers
		OrderBy:        " ORDER BY tbl_name, name",

		GetCode: `
			SELECT sql
			FROM sqlite_master
			WHERE type = 'trigger' AND name = ?`,
	}
}

// DatabaseInfo returns SQLite database info queries
func (d *SQLiteDialect) DatabaseInfo() DatabaseInfoSQL {
	return DatabaseInfoSQL{
		Version: "SELECT sqlite_version()",

		Details: "", // SQLite doesn't have detailed info

		ObjectCounts: `
			SELECT
				SUM(CASE WHEN type = 'table' THEN 1 ELSE 0 END) AS tables,
				SUM(CASE WHEN type = 'view' THEN 1 ELSE 0 END) AS views,
				0 AS procedures,
				0 AS functions,
				SUM(CASE WHEN type = 'trigger' THEN 1 ELSE 0 END) AS triggers
			FROM sqlite_master
			WHERE name NOT LIKE 'sqlite_%'`,

		ListSchemas: "", // SQLite doesn't have schemas

		SearchObjects: `
			SELECT
				'' AS schema_name,
				name AS object_name,
				type AS object_type,
				NULL AS create_date,
				NULL AS modify_date,
				CASE WHEN sql IS NOT NULL THEN 1 ELSE 0 END AS has_code
			FROM sqlite_master
			WHERE name NOT LIKE 'sqlite_%%'
			  AND (name LIKE '%%' || ? || '%%' %s)
			  %s
			ORDER BY name`,
	}
}

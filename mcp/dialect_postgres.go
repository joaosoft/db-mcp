package mcp

import (
	"fmt"
	"strings"
)

// PostgresDialect implements Dialect for PostgreSQL
type PostgresDialect struct {
	BaseDialect
}

// NewPostgresDialect creates a new PostgreSQL dialect
func NewPostgresDialect() *PostgresDialect {
	return &PostgresDialect{
		BaseDialect: BaseDialect{driver: DriverPostgresSQL},
	}
}

// Placeholder returns $1, $2, etc.
func (d *PostgresDialect) Placeholder(index int) string {
	return fmt.Sprintf("$%d", index)
}

// QuoteIdentifier returns "name"
func (d *PostgresDialect) QuoteIdentifier(name string) string {
	return fmt.Sprintf(`"%s"`, name)
}

// PaginationClause returns LIMIT/OFFSET syntax
func (d *PostgresDialect) PaginationClause(limit, offset int, orderBy string) string {
	pagination := fmt.Sprintf("LIMIT %d OFFSET %d", limit, offset)
	if orderBy != "" {
		return fmt.Sprintf("ORDER BY %s %s", orderBy, pagination)
	}
	return pagination
}

// LikeOperator returns ILIKE for case-insensitive searches
func (d *PostgresDialect) LikeOperator(caseSensitive bool) string {
	if !caseSensitive {
		return "ILIKE"
	}
	return "LIKE"
}

// ConcatOperator returns || concatenation
func (d *PostgresDialect) ConcatOperator(parts ...string) string {
	return strings.Join(parts, " || ")
}

// CurrentDatabase returns current_database()
func (d *PostgresDialect) CurrentDatabase() string {
	return "current_database()"
}

// SystemSchemas returns PostgreSQL system schemas
func (d *PostgresDialect) SystemSchemas() []string {
	return []string{"pg_catalog", "information_schema", "pg_toast"}
}

// SupportsFeature checks PostgreSQL feature support
func (d *PostgresDialect) SupportsFeature(feature DialectFeature) bool {
	switch feature {
	case FeatureILike:
		return true
	default:
		return true
	}
}

// TableMetadata returns PostgreSQL table metadata queries
func (d *PostgresDialect) TableMetadata() TableMetadataSQL {
	return TableMetadataSQL{
		ListTables: `
			SELECT
				table_schema,
				table_name,
				table_type
			FROM information_schema.tables
			WHERE table_type = 'BASE TABLE'
				AND table_schema NOT IN ('pg_catalog', 'information_schema')`,
		SchemaFilter: " AND table_schema = %s",
		NameFilter:   " AND table_name ILIKE %s",
		OrderBy:      " ORDER BY table_schema, table_name",

		DescribeTable: `
			SELECT
				column_name,
				data_type,
				is_nullable,
				column_default,
				character_maximum_length
			FROM information_schema.columns
			WHERE table_schema = $1 AND table_name = $2
			ORDER BY ordinal_position`,

		TableExists: `SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = $1 AND table_name = $2`,

		GetColumns: `
			SELECT
				column_name,
				data_type,
				character_maximum_length,
				is_nullable,
				column_default
			FROM information_schema.columns
			WHERE table_schema = $1 AND table_name = $2
			ORDER BY ordinal_position`,

		GetFullSchema: `
			SELECT
				c.column_name,
				c.data_type,
				c.character_maximum_length,
				c.numeric_precision,
				c.numeric_scale,
				c.is_nullable,
				c.column_default,
				CASE WHEN pk.column_name IS NOT NULL THEN 'YES' ELSE 'NO' END AS is_primary_key
			FROM information_schema.columns c
			LEFT JOIN (
				SELECT ku.table_schema, ku.table_name, ku.column_name
				FROM information_schema.table_constraints tc
				JOIN information_schema.key_column_usage ku
					ON tc.constraint_type = 'PRIMARY KEY'
					AND tc.constraint_name = ku.constraint_name
					AND tc.table_schema = ku.table_schema
					AND tc.table_name = ku.table_name
			) pk ON c.table_schema = pk.table_schema
				AND c.table_name = pk.table_name
				AND c.column_name = pk.column_name
			WHERE c.table_schema = $1 AND c.table_name = $2
			ORDER BY c.ordinal_position`,

		GetPrimaryKey: `
			SELECT ku.column_name
			FROM information_schema.table_constraints tc
			JOIN information_schema.key_column_usage ku
				ON tc.constraint_name = ku.constraint_name
				AND tc.table_schema = ku.table_schema
				AND tc.table_name = ku.table_name
			WHERE tc.constraint_type = 'PRIMARY KEY'
				AND tc.table_schema = $1
				AND tc.table_name = $2
			ORDER BY ku.ordinal_position`,

		GetIndexes: `
			SELECT
				i.indexname AS index_name,
				a.amname AS index_type,
				ix.indisunique AS is_unique,
				a2.attname AS column_name
			FROM pg_indexes i
			JOIN pg_class c ON i.indexname = c.relname
			JOIN pg_index ix ON c.oid = ix.indexrelid
			JOIN pg_class t ON ix.indrelid = t.oid
			JOIN pg_namespace n ON t.relnamespace = n.oid
			JOIN pg_am a ON c.relam = a.oid
			JOIN pg_attribute a2 ON a2.attrelid = t.oid AND a2.attnum = ANY(ix.indkey)
			WHERE n.nspname = $1 AND t.relname = $2
			ORDER BY i.indexname`,

		GetForeignKeys: `
			SELECT
				tc.constraint_name,
				kcu.column_name,
				ccu.table_schema AS referenced_schema,
				ccu.table_name AS referenced_table,
				ccu.column_name AS referenced_column
			FROM information_schema.table_constraints tc
			JOIN information_schema.key_column_usage kcu
				ON tc.constraint_name = kcu.constraint_name
				AND tc.table_schema = kcu.table_schema
			JOIN information_schema.constraint_column_usage ccu
				ON ccu.constraint_name = tc.constraint_name
				AND ccu.table_schema = tc.table_schema
			WHERE tc.constraint_type = 'FOREIGN KEY'
				AND tc.table_schema = $1
				AND tc.table_name = $2
			ORDER BY tc.constraint_name`,
	}
}

// ProcedureMetadata returns PostgreSQL procedure metadata queries
func (d *PostgresDialect) ProcedureMetadata() ProcedureMetadataSQL {
	return ProcedureMetadataSQL{
		ListProcedures: `
			SELECT
				routine_schema,
				routine_name,
				created::timestamp AS created,
				created::timestamp AS last_altered
			FROM information_schema.routines
			WHERE routine_type = 'PROCEDURE'
				AND routine_schema NOT IN ('pg_catalog', 'information_schema')`,
		SchemaFilter: " AND routine_schema = %s",
		NameFilter:   " AND routine_name ILIKE %s",
		OrderBy:      " ORDER BY routine_schema, routine_name",

		GetCode: `
			SELECT pg_get_functiondef(p.oid)
			FROM pg_proc p
			JOIN pg_namespace n ON p.pronamespace = n.oid
			WHERE n.nspname = $1 AND p.proname = $2 AND prokind = 'p'`,
	}
}

// FunctionMetadata returns PostgreSQL function metadata queries
func (d *PostgresDialect) FunctionMetadata() FunctionMetadataSQL {
	return FunctionMetadataSQL{
		ListFunctions: `
			SELECT
				n.nspname AS routine_schema,
				p.proname AS routine_name,
				CASE WHEN p.proretset THEN 'TABLE' ELSE 'SCALAR' END AS function_type,
				NULL::timestamp AS created,
				NULL::timestamp AS last_altered
			FROM pg_proc p
			JOIN pg_namespace n ON p.pronamespace = n.oid
			WHERE n.nspname NOT IN ('pg_catalog', 'information_schema')
				AND prokind = 'f'`,
		TypeFilterScalar: "", // PostgreSQL doesn't have direct filter, use proretset
		TypeFilterTable:  "",
		TypeFilterAll:    "",
		SchemaFilter:     " AND n.nspname = %s",
		NameFilter:       " AND p.proname ILIKE %s",
		OrderBy:          " ORDER BY n.nspname, p.proname",

		GetCode: `
			SELECT pg_get_functiondef(p.oid)
			FROM pg_proc p
			JOIN pg_namespace n ON p.pronamespace = n.oid
			WHERE n.nspname = $1 AND p.proname = $2 AND prokind = 'f'`,
	}
}

// ViewMetadata returns PostgreSQL view metadata queries
func (d *PostgresDialect) ViewMetadata() ViewMetadataSQL {
	return ViewMetadataSQL{
		ListViews: `
			SELECT
				table_schema as view_schema,
				table_name as view_name,
				NULL::timestamp as created,
				NULL::timestamp as last_altered
			FROM information_schema.views
			WHERE table_schema NOT IN ('pg_catalog', 'information_schema')`,
		SchemaFilter: " AND table_schema = %s",
		NameFilter:   " AND table_name ILIKE %s",
		OrderBy:      " ORDER BY table_schema, table_name",

		GetDefinition: `
			SELECT view_definition
			FROM information_schema.views
			WHERE table_schema = $1 AND table_name = $2`,
	}
}

// TriggerMetadata returns PostgreSQL trigger metadata queries
func (d *PostgresDialect) TriggerMetadata() TriggerMetadataSQL {
	return TriggerMetadataSQL{
		ListTriggers: `
			SELECT
				n.nspname AS schema_name,
				t.tgname AS trigger_name,
				c.relname AS table_name,
				NOT t.tgenabled::boolean AS is_disabled,
				NULL::timestamp AS create_date,
				NULL::timestamp AS modify_date
			FROM pg_trigger t
			JOIN pg_class c ON t.tgrelid = c.oid
			JOIN pg_namespace n ON c.relnamespace = n.oid
			WHERE NOT t.tgisinternal
				AND n.nspname NOT IN ('pg_catalog', 'information_schema')`,
		SchemaFilter:   " AND n.nspname = %s",
		TableFilter:    " AND c.relname = %s",
		NameFilter:     " AND t.tgname ILIKE %s",
		DisabledFilter: "", // PostgreSQL doesn't have simple disabled filter
		OrderBy:        " ORDER BY n.nspname, c.relname, t.tgname",

		GetCode: `
			SELECT pg_get_triggerdef(t.oid)
			FROM pg_trigger t
			JOIN pg_class c ON t.tgrelid = c.oid
			JOIN pg_namespace n ON c.relnamespace = n.oid
			WHERE n.nspname = $1 AND t.tgname = $2`,
	}
}

// DatabaseInfo returns PostgreSQL database info queries
func (d *PostgresDialect) DatabaseInfo() DatabaseInfoSQL {
	return DatabaseInfoSQL{
		Version: "SELECT version()",

		Details: `
			SELECT
				current_database() AS database_name,
				pg_encoding_to_char(encoding) AS encoding,
				datcollate AS collation,
				'' AS recovery_model_desc,
				0 AS compatibility_level,
				NULL AS create_date
			FROM pg_database
			WHERE datname = current_database()`,

		ObjectCounts: `
			SELECT
				COUNT(CASE WHEN table_type = 'BASE TABLE' THEN 1 END) AS tables,
				COUNT(CASE WHEN table_type = 'VIEW' THEN 1 END) AS views,
				(SELECT COUNT(*) FROM pg_proc p JOIN pg_namespace n ON p.pronamespace = n.oid WHERE n.nspname NOT IN ('pg_catalog', 'information_schema') AND prokind = 'p') AS procedures,
				(SELECT COUNT(*) FROM pg_proc p JOIN pg_namespace n ON p.pronamespace = n.oid WHERE n.nspname NOT IN ('pg_catalog', 'information_schema') AND prokind = 'f') AS functions,
				(SELECT COUNT(*) FROM pg_trigger t JOIN pg_class c ON t.tgrelid = c.oid JOIN pg_namespace n ON c.relnamespace = n.oid WHERE n.nspname NOT IN ('pg_catalog', 'information_schema') AND NOT tgisinternal) AS triggers
			FROM information_schema.tables
			WHERE table_schema NOT IN ('pg_catalog', 'information_schema')`,

		ListSchemas: `
			SELECT schema_name
			FROM information_schema.schemata
			WHERE schema_name NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
			ORDER BY schema_name`,

		SearchObjects: `
			SELECT
				table_schema AS schema_name,
				table_name AS object_name,
				table_type AS object_type,
				NULL AS create_date,
				NULL AS modify_date,
				CASE WHEN view_definition IS NOT NULL THEN 1 ELSE 0 END AS has_code
			FROM information_schema.tables
			WHERE table_schema NOT IN ('pg_catalog', 'information_schema')
			  AND (table_name LIKE '%%' || $1 || '%%' %s)
			  %s
			ORDER BY table_schema, table_name`,
	}
}

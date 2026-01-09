package mcp

import (
	"fmt"
	"strings"
)

// SQLServerDialect implements Dialect for Microsoft SQL Server
type SQLServerDialect struct {
	BaseDialect
}

// NewSQLServerDialect creates a new SQL Server dialect
func NewSQLServerDialect() *SQLServerDialect {
	return &SQLServerDialect{
		BaseDialect: BaseDialect{driver: DriverSQLServer},
	}
}

// Placeholder returns @p1, @p2, etc.
func (d *SQLServerDialect) Placeholder(index int) string {
	return fmt.Sprintf("@p%d", index)
}

// QuoteIdentifier returns [name]
func (d *SQLServerDialect) QuoteIdentifier(name string) string {
	return fmt.Sprintf("[%s]", name)
}

// PaginationClause returns OFFSET/FETCH syntax
// SQL Server requires ORDER BY before OFFSET/FETCH, uses (SELECT NULL) if not provided
func (d *SQLServerDialect) PaginationClause(limit, offset int, orderBy string) string {
	if orderBy == "" {
		orderBy = "(SELECT NULL)"
	}
	return fmt.Sprintf("ORDER BY %s OFFSET %d ROWS FETCH NEXT %d ROWS ONLY", orderBy, offset, limit)
}

// ConcatOperator returns + concatenation
func (d *SQLServerDialect) ConcatOperator(parts ...string) string {
	return strings.Join(parts, " + ")
}

// CurrentDatabase returns DB_NAME()
func (d *SQLServerDialect) CurrentDatabase() string {
	return "DB_NAME()"
}

// SystemSchemas returns SQL Server system schemas
func (d *SQLServerDialect) SystemSchemas() []string {
	return []string{"sys", "INFORMATION_SCHEMA"}
}

// TableMetadata returns SQL Server table metadata queries
func (d *SQLServerDialect) TableMetadata() TableMetadataSQL {
	return TableMetadataSQL{
		ListTables: `
			SELECT
				TABLE_SCHEMA,
				TABLE_NAME,
				TABLE_TYPE
			FROM INFORMATION_SCHEMA.TABLES
			WHERE TABLE_TYPE = 'BASE TABLE'`,
		SchemaFilter: " AND TABLE_SCHEMA = %s",
		NameFilter:   " AND TABLE_NAME LIKE %s",
		OrderBy:      " ORDER BY TABLE_SCHEMA, TABLE_NAME",

		DescribeTable: `
			SELECT
				COLUMN_NAME,
				DATA_TYPE,
				IS_NULLABLE,
				COLUMN_DEFAULT,
				CHARACTER_MAXIMUM_LENGTH
			FROM INFORMATION_SCHEMA.COLUMNS
			WHERE TABLE_SCHEMA = @p1 AND TABLE_NAME = @p2
			ORDER BY ORDINAL_POSITION`,

		TableExists: `SELECT COUNT(*) FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_SCHEMA = @p1 AND TABLE_NAME = @p2`,

		GetColumns: `
			SELECT
				COLUMN_NAME,
				DATA_TYPE,
				CHARACTER_MAXIMUM_LENGTH,
				IS_NULLABLE,
				COLUMN_DEFAULT
			FROM INFORMATION_SCHEMA.COLUMNS
			WHERE TABLE_SCHEMA = @p1 AND TABLE_NAME = @p2
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
			WHERE c.TABLE_SCHEMA = @p1 AND c.TABLE_NAME = @p2
			ORDER BY c.ORDINAL_POSITION`,

		GetPrimaryKey: `
			SELECT ku.COLUMN_NAME
			FROM INFORMATION_SCHEMA.TABLE_CONSTRAINTS tc
			JOIN INFORMATION_SCHEMA.KEY_COLUMN_USAGE ku
				ON tc.CONSTRAINT_NAME = ku.CONSTRAINT_NAME
				AND tc.TABLE_SCHEMA = ku.TABLE_SCHEMA
				AND tc.TABLE_NAME = ku.TABLE_NAME
			WHERE tc.CONSTRAINT_TYPE = 'PRIMARY KEY'
				AND tc.TABLE_SCHEMA = @p1
				AND tc.TABLE_NAME = @p2
			ORDER BY ku.ORDINAL_POSITION`,

		GetIndexes: `
			SELECT
				i.name AS index_name,
				i.type_desc AS index_type,
				i.is_unique,
				COL_NAME(ic.object_id, ic.column_id) AS column_name
			FROM sys.indexes i
			INNER JOIN sys.index_columns ic ON i.object_id = ic.object_id AND i.index_id = ic.index_id
			INNER JOIN sys.tables t ON i.object_id = t.object_id
			INNER JOIN sys.schemas s ON t.schema_id = s.schema_id
			WHERE s.name = @p1 AND t.name = @p2
			ORDER BY i.name, ic.key_ordinal`,

		GetForeignKeys: `
			SELECT
				fk.name AS constraint_name,
				COL_NAME(fkc.parent_object_id, fkc.parent_column_id) AS column_name,
				SCHEMA_NAME(ref_t.schema_id) AS referenced_schema,
				ref_t.name AS referenced_table,
				COL_NAME(fkc.referenced_object_id, fkc.referenced_column_id) AS referenced_column
			FROM sys.foreign_keys fk
			INNER JOIN sys.foreign_key_columns fkc ON fk.object_id = fkc.constraint_object_id
			INNER JOIN sys.tables t ON fk.parent_object_id = t.object_id
			INNER JOIN sys.schemas s ON t.schema_id = s.schema_id
			INNER JOIN sys.tables ref_t ON fkc.referenced_object_id = ref_t.object_id
			WHERE s.name = @p1 AND t.name = @p2
			ORDER BY fk.name`,
	}
}

// ProcedureMetadata returns SQL Server procedure metadata queries
func (d *SQLServerDialect) ProcedureMetadata() ProcedureMetadataSQL {
	return ProcedureMetadataSQL{
		ListProcedures: `
			SELECT
				s.name AS routine_schema,
				o.name AS routine_name,
				o.create_date AS created,
				o.modify_date AS last_altered
			FROM sys.objects o
			INNER JOIN sys.schemas s ON o.schema_id = s.schema_id
			WHERE o.type = 'P' AND o.is_ms_shipped = 0`,
		SchemaFilter: " AND s.name = %s",
		NameFilter:   " AND o.name LIKE %s",
		OrderBy:      " ORDER BY s.name, o.name",

		GetCode: `
			SELECT m.definition
			FROM sys.sql_modules m
			INNER JOIN sys.objects o ON m.object_id = o.object_id
			INNER JOIN sys.schemas s ON o.schema_id = s.schema_id
			WHERE o.type = 'P' AND s.name = @p1 AND o.name = @p2`,
	}
}

// FunctionMetadata returns SQL Server function metadata queries
func (d *SQLServerDialect) FunctionMetadata() FunctionMetadataSQL {
	return FunctionMetadataSQL{
		ListFunctions: `
			SELECT
				s.name AS routine_schema,
				o.name AS routine_name,
				o.type_desc AS function_type,
				o.create_date AS created,
				o.modify_date AS last_altered
			FROM sys.objects o
			INNER JOIN sys.schemas s ON o.schema_id = s.schema_id
			WHERE o.is_ms_shipped = 0`,
		TypeFilterScalar: " AND o.type = 'FN'",
		TypeFilterTable:  " AND o.type IN ('IF', 'TF')",
		TypeFilterAll:    " AND o.type IN ('FN', 'IF', 'TF')",
		SchemaFilter:     " AND s.name = %s",
		NameFilter:       " AND o.name LIKE %s",
		OrderBy:          " ORDER BY s.name, o.name",

		GetCode: `
			SELECT m.definition
			FROM sys.sql_modules m
			INNER JOIN sys.objects o ON m.object_id = o.object_id
			INNER JOIN sys.schemas s ON o.schema_id = s.schema_id
			WHERE o.type IN ('FN', 'IF', 'TF') AND s.name = @p1 AND o.name = @p2`,
	}
}

// ViewMetadata returns SQL Server view metadata queries
func (d *SQLServerDialect) ViewMetadata() ViewMetadataSQL {
	return ViewMetadataSQL{
		ListViews: `
			SELECT
				s.name AS view_schema,
				v.name AS view_name,
				v.create_date AS created,
				v.modify_date AS last_altered
			FROM sys.views v
			INNER JOIN sys.schemas s ON v.schema_id = s.schema_id
			WHERE v.is_ms_shipped = 0`,
		SchemaFilter: " AND s.name = %s",
		NameFilter:   " AND v.name LIKE %s",
		OrderBy:      " ORDER BY s.name, v.name",

		GetDefinition: `
			SELECT m.definition
			FROM sys.sql_modules m
			INNER JOIN sys.views v ON m.object_id = v.object_id
			INNER JOIN sys.schemas s ON v.schema_id = s.schema_id
			WHERE s.name = @p1 AND v.name = @p2`,
	}
}

// TriggerMetadata returns SQL Server trigger metadata queries
func (d *SQLServerDialect) TriggerMetadata() TriggerMetadataSQL {
	return TriggerMetadataSQL{
		ListTriggers: `
			SELECT
				s.name AS schema_name,
				t.name AS trigger_name,
				OBJECT_NAME(tr.parent_id) AS table_name,
				tr.is_disabled,
				tr.create_date,
				tr.modify_date
			FROM sys.triggers tr
			INNER JOIN sys.objects t ON tr.object_id = t.object_id
			INNER JOIN sys.schemas s ON t.schema_id = s.schema_id
			WHERE 1=1`,
		SchemaFilter:   " AND s.name = %s",
		TableFilter:    " AND OBJECT_NAME(tr.parent_id) = %s",
		NameFilter:     " AND t.name LIKE %s",
		DisabledFilter: " AND tr.is_disabled = 0",
		OrderBy:        " ORDER BY s.name, OBJECT_NAME(tr.parent_id), t.name",

		GetCode: `
			SELECT m.definition
			FROM sys.sql_modules m
			INNER JOIN sys.triggers tr ON m.object_id = tr.object_id
			INNER JOIN sys.objects t ON tr.object_id = t.object_id
			INNER JOIN sys.schemas s ON t.schema_id = s.schema_id
			WHERE s.name = @p1 AND t.name = @p2`,
	}
}

// DatabaseInfo returns SQL Server database info queries
func (d *SQLServerDialect) DatabaseInfo() DatabaseInfoSQL {
	return DatabaseInfoSQL{
		Version: "SELECT @@VERSION",

		Details: `
			SELECT
				DB_NAME() AS database_name,
				collation_name,
				recovery_model_desc,
				compatibility_level,
				create_date
			FROM sys.databases
			WHERE name = DB_NAME()`,

		ObjectCounts: `
			SELECT
				COUNT(CASE WHEN type = 'U' THEN 1 END) AS tables,
				COUNT(CASE WHEN type = 'V' THEN 1 END) AS views,
				COUNT(CASE WHEN type = 'P' THEN 1 END) AS procedures,
				COUNT(CASE WHEN type IN ('FN', 'IF', 'TF') THEN 1 END) AS functions,
				COUNT(CASE WHEN type = 'TR' THEN 1 END) AS triggers
			FROM sys.objects
			WHERE is_ms_shipped = 0`,

		ListSchemas: `
			SELECT name
			FROM sys.schemas
			WHERE schema_id < 16384
			ORDER BY name`,

		SearchObjects: `
			SELECT DISTINCT
				s.name AS schema_name,
				o.name AS object_name,
				o.type_desc AS object_type,
				o.create_date,
				o.modify_date,
				CASE WHEN m.definition IS NOT NULL THEN 1 ELSE 0 END AS has_code
			FROM sys.objects o
			INNER JOIN sys.schemas s ON o.schema_id = s.schema_id
			LEFT JOIN sys.sql_modules m ON o.object_id = m.object_id
			WHERE o.type IN (%s)
			  AND (o.name LIKE '%%' + @p1 + '%%' %s)
			ORDER BY s.name, o.name`,
	}
}

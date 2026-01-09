package mcp

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func (s *DbMCPServer) toolListTables() (mcp.Tool, server.ToolHandlerFunc) {
	return mcp.Tool{
		Name:        "list_tables",
		Description: "List database tables with pagination",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"schema": map[string]interface{}{
					"type":        "string",
					"description": "Schema name (optional)",
				},
				"name_filter": map[string]interface{}{
					"type":        "string",
					"description": "Filter by table name (optional)",
				},
				"page": map[string]interface{}{
					"type":        "number",
					"description": "Page number (default: 1)",
				},
				"page_size": map[string]interface{}{
					"type":        "number",
					"description": "Items per page (default: 100, maximum: 500)",
				},
			},
		},
	}, s.handleListTables
}

func (s *DbMCPServer) handleListTables(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := s.requireConnection(); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	args, ok := getArgs(request.Params.Arguments)
	if !ok {
		return mcp.NewToolResultError(ErrInvalidArguments.Error()), nil
	}

	schema, err := getValidSchema(args, "")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	nameFilter, _ := getStringArg(args, "name_filter")
	pagination := GetPaginationParams(args, DefaultPageSize, MaxPageSize)

	query, queryArgs := s.queryBuilder.ListTablesQuery(schema, nameFilter, pagination.PageSize, pagination.Offset)

	ctx, cancel := context.WithTimeout(ctx, ShortQueryTimeout)
	defer cancel()

	rows, err := s.db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Errorf("%w: %v", ErrListingTables, err).Error()), nil
	}
	defer rows.Close()

	var tables []map[string]interface{}
	for rows.Next() {
		var tableSchema, tableName, tableType string
		if err = rows.Scan(&tableSchema, &tableName, &tableType); err != nil {
			continue
		}
		tables = append(tables, map[string]interface{}{
			"schema": tableSchema,
			"name":   tableName,
			"type":   tableType,
		})
	}

	response := map[string]interface{}{
		"tables": tables,
		"pagination": map[string]interface{}{
			"page":      pagination.Page,
			"page_size": pagination.PageSize,
			"count":     len(tables),
		},
		"filter": map[string]interface{}{
			"schema":      schema,
			"name_filter": nameFilter,
		},
	}

	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(ErrSerializingJSON.Error()), nil
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

func (s *DbMCPServer) toolDescribeTable() (mcp.Tool, server.ToolHandlerFunc) {
	return mcp.Tool{
		Name:        "describe_table",
		Description: "Returns the structure of a table (columns, types, constraints)",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"table_name": map[string]interface{}{
					"type":        "string",
					"description": "Table name",
				},
				"schema": map[string]interface{}{
					"type":        "string",
					"description": "Schema name (optional)",
				},
			},
			Required: []string{"table_name"},
		},
	}, s.handleDescribeTable
}

func (s *DbMCPServer) handleDescribeTable(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := s.requireConnection(); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	args, ok := getArgs(request.Params.Arguments)
	if !ok {
		return mcp.NewToolResultError(ErrInvalidArguments.Error()), nil
	}

	tableName, ok := getStringArg(args, "table_name")
	if !ok || !isValidIdentifier(tableName) {
		return mcp.NewToolResultError(ErrInvalidTableName.Error()), nil
	}

	defaultSchema := getDefaultSchema(s.queryBuilder.GetDriver())
	schema, err := getValidSchema(args, defaultSchema)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	query, queryArgs := s.queryBuilder.DescribeTableQuery(schema, tableName)

	ctx, cancel := context.WithTimeout(ctx, ShortQueryTimeout)
	defer cancel()

	rows, err := s.db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Errorf("%w: %v", ErrDescribingTable, err).Error()), nil
	}
	defer rows.Close()

	var columns []map[string]interface{}

	// Handle SQLite PRAGMA differently
	if s.queryBuilder.IsSQLite() {
		columns = s.parseSQLitePragmaTableInfo(rows)
	} else {
		columns = s.parseStandardDescribeTable(rows)
	}

	if len(columns) == 0 {
		return mcp.NewToolResultError(ErrTableNotFound.Error()), nil
	}

	response := map[string]interface{}{
		"schema":  schema,
		"table":   tableName,
		"columns": columns,
	}

	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(ErrSerializingJSON.Error()), nil
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

func (s *DbMCPServer) parseStandardDescribeTable(rows *sql.Rows) []map[string]interface{} {
	var columns []map[string]interface{}
	for rows.Next() {
		var colName, dataType, isNullable string
		var colDefault sql.NullString
		var maxLength sql.NullInt64

		if err := rows.Scan(&colName, &dataType, &isNullable, &colDefault, &maxLength); err != nil {
			continue
		}

		col := map[string]interface{}{
			"name":     colName,
			"type":     dataType,
			"nullable": strings.EqualFold(isNullable, "YES") || strings.EqualFold(isNullable, "Y"),
		}
		if maxLength.Valid {
			col["max_length"] = maxLength.Int64
		}
		if colDefault.Valid {
			col["default"] = colDefault.String
		}
		columns = append(columns, col)
	}
	return columns
}

func (s *DbMCPServer) parseSQLitePragmaTableInfo(rows *sql.Rows) []map[string]interface{} {
	var columns []map[string]interface{}
	for rows.Next() {
		var cid int
		var name, dataType string
		var notNull, pk int
		var dfltValue sql.NullString

		if err := rows.Scan(&cid, &name, &dataType, &notNull, &dfltValue, &pk); err != nil {
			continue
		}

		col := map[string]interface{}{
			"name":        name,
			"type":        dataType,
			"nullable":    notNull == 0,
			"primary_key": pk == 1,
		}
		if dfltValue.Valid {
			col["default"] = dfltValue.String
		}
		columns = append(columns, col)
	}
	return columns
}

func (s *DbMCPServer) toolListTableRows() (mcp.Tool, server.ToolHandlerFunc) {
	return mcp.Tool{
		Name:        "list_table_rows",
		Description: "List the rows of a database table with pagination and advanced filters",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"table_name": map[string]interface{}{
					"type":        "string",
					"description": "Table name",
				},
				"schema": map[string]interface{}{
					"type":        "string",
					"description": "Schema name (optional)",
				},
				"filters": map[string]interface{}{
					"type":        "array",
					"description": "Filters (e.g.: [{\"column\": \"name\", \"operator\": \"contains\", \"value\": \"john\"}])",
					"items": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"column": map[string]interface{}{
								"type":        "string",
								"description": "Column name",
							},
							"operator": map[string]interface{}{
								"type":        "string",
								"description": "Operator: eq, neq, gt, gte, lt, lte, contains, starts_with, ends_with, is_null, is_not_null",
								"enum":        []string{"eq", "neq", "gt", "gte", "lt", "lte", "contains", "starts_with", "ends_with", "is_null", "is_not_null"},
							},
							"value": map[string]interface{}{
								"description": "Value to compare (not required for is_null/is_not_null)",
							},
						},
						"required": []string{"column", "operator"},
					},
				},
				"page": map[string]interface{}{
					"type":        "number",
					"description": "Page number (default: 1)",
				},
				"page_size": map[string]interface{}{
					"type":        "number",
					"description": "Items per page (default: 50, maximum: 1000)",
				},
				"order_by": map[string]interface{}{
					"type":        "string",
					"description": "Column for sorting (optional)",
				},
				"order_direction": map[string]interface{}{
					"type":        "string",
					"description": "Sorting direction: ASC or DESC (default: ASC)",
				},
			},
			Required: []string{"table_name"},
		},
	}, s.handleListTableRows
}

func (s *DbMCPServer) handleListTableRows(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := s.requireConnection(); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	args, ok := getArgs(request.Params.Arguments)
	if !ok {
		return mcp.NewToolResultError(ErrInvalidArguments.Error()), nil
	}

	tableName, ok := getStringArg(args, "table_name")
	if !ok || !isValidIdentifier(tableName) {
		return mcp.NewToolResultError(ErrInvalidTableName.Error()), nil
	}

	defaultSchema := getDefaultSchema(s.queryBuilder.GetDriver())
	schema, err := getValidSchema(args, defaultSchema)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	ctx, cancel := context.WithTimeout(ctx, DefaultQueryTimeout)
	defer cancel()

	// Check if table exists
	if exists, err := s.tableExists(ctx, schema, tableName); err != nil {
		return mcp.NewToolResultError(fmt.Errorf("%w: %v", ErrCheckingTable, err).Error()), nil
	} else if !exists {
		return mcp.NewToolResultError(fmt.Errorf("%w: %s.%s", ErrTableNotFound, schema, tableName).Error()), nil
	}

	// Get columns
	columns, err := s.getTableColumns(ctx, schema, tableName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Errorf("%w: %v", ErrRetrievingColumns, err).Error()), nil
	}
	if len(columns) == 0 {
		return mcp.NewToolResultError(ErrNoColumnsFound.Error()), nil
	}

	// Pagination
	pagination := GetPaginationParams(args, 50, MaxRowsPageSize)

	// Sorting
	orderBy, _ := getStringArg(args, "order_by")
	orderDirection := "ASC"
	if dir, ok := getStringArg(args, "order_direction"); ok {
		dir = strings.ToUpper(dir)
		if dir == "DESC" || dir == "ASC" {
			orderDirection = dir
		}
	}

	// Validate orderBy column
	if orderBy != "" {
		if !s.columnExists(columns, orderBy) {
			return mcp.NewToolResultError(fmt.Errorf("%w: %s", ErrColumnNotExists, orderBy).Error()), nil
		}
	} else if len(columns) > 0 {
		orderBy = columns[0]
	}

	// Build WHERE clause from filters
	whereClauses, queryParams, err := s.buildWhereClause(args, columns)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	whereClause := ""
	if len(whereClauses) > 0 {
		whereClause = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	// Count total rows
	totalCount, err := s.countRows(ctx, schema, tableName, whereClause, queryParams)
	if err != nil {
		return mcp.NewToolResultError(fmt.Errorf("%w: %v", ErrCountingRows, err).Error()), nil
	}

	// Fetch rows
	rows, err := s.fetchRows(ctx, schema, tableName, columns, whereClause, orderBy, orderDirection, pagination, queryParams)
	if err != nil {
		return mcp.NewToolResultError(fmt.Errorf("%w: %v", ErrFetchingRows, err).Error()), nil
	}

	totalPages := (totalCount + pagination.PageSize - 1) / pagination.PageSize
	if totalCount == 0 {
		totalPages = 0
	}

	response := map[string]interface{}{
		"rows":    rows,
		"columns": columns,
		"pagination": map[string]interface{}{
			"page":         pagination.Page,
			"page_size":    pagination.PageSize,
			"total_count":  totalCount,
			"total_pages":  totalPages,
			"has_next":     pagination.Page < totalPages,
			"has_previous": pagination.Page > 1,
		},
		"table": map[string]interface{}{
			"schema":          schema,
			"name":            tableName,
			"order_by":        orderBy,
			"order_direction": orderDirection,
		},
	}

	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(ErrSerializingJSON.Error()), nil
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

func (s *DbMCPServer) tableExists(ctx context.Context, schema, tableName string) (bool, error) {
	query, args := s.queryBuilder.TableExistsQuery(schema, tableName)
	var count int
	err := s.db.QueryRowContext(ctx, query, args...).Scan(&count)
	return count > 0, err
}

func (s *DbMCPServer) getTableColumns(ctx context.Context, schema, tableName string) ([]string, error) {
	query, args := s.queryBuilder.GetTableColumnsQuery(schema, tableName)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []string
	if s.queryBuilder.IsSQLite() {
		for rows.Next() {
			var cid int
			var name, dataType string
			var notNull, pk int
			var dfltValue sql.NullString
			if err := rows.Scan(&cid, &name, &dataType, &notNull, &dfltValue, &pk); err != nil {
				continue
			}
			columns = append(columns, name)
		}
	} else {
		for rows.Next() {
			var columnName, dataType string
			var maxLength sql.NullInt64
			var isNullable string
			var colDefault sql.NullString
			if err := rows.Scan(&columnName, &dataType, &maxLength, &isNullable, &colDefault); err != nil {
				continue
			}
			columns = append(columns, columnName)
		}
	}
	return columns, nil
}

func (s *DbMCPServer) columnExists(columns []string, name string) bool {
	for _, col := range columns {
		if strings.EqualFold(col, name) {
			return true
		}
	}
	return false
}

func (s *DbMCPServer) buildWhereClause(args map[string]interface{}, columns []string) ([]string, []interface{}, error) {
	var whereClauses []string
	var queryParams []interface{}
	paramIndex := 1

	filters, ok := args["filters"].([]interface{})
	if !ok {
		return whereClauses, queryParams, nil
	}

	for _, filterInterface := range filters {
		filter, ok := filterInterface.(map[string]interface{})
		if !ok {
			continue
		}

		column, _ := filter["column"].(string)
		operator, _ := filter["operator"].(string)

		if column == "" || operator == "" {
			continue
		}

		// Validate column exists
		if !s.columnExists(columns, column) {
			return nil, nil, fmt.Errorf("%w: %s", ErrColumnNotExists, column)
		}

		// Quote column name based on driver
		quotedColumn := s.queryBuilder.QuoteIdentifier(column)
		placeholder := s.queryBuilder.Placeholder(paramIndex)
		likeOp := s.queryBuilder.LikeOperator(false)

		switch operator {
		case "eq":
			whereClauses = append(whereClauses, fmt.Sprintf("%s = %s", quotedColumn, placeholder))
			queryParams = append(queryParams, filter["value"])
			paramIndex++
		case "neq":
			whereClauses = append(whereClauses, fmt.Sprintf("%s != %s", quotedColumn, placeholder))
			queryParams = append(queryParams, filter["value"])
			paramIndex++
		case "gt":
			whereClauses = append(whereClauses, fmt.Sprintf("%s > %s", quotedColumn, placeholder))
			queryParams = append(queryParams, filter["value"])
			paramIndex++
		case "gte":
			whereClauses = append(whereClauses, fmt.Sprintf("%s >= %s", quotedColumn, placeholder))
			queryParams = append(queryParams, filter["value"])
			paramIndex++
		case "lt":
			whereClauses = append(whereClauses, fmt.Sprintf("%s < %s", quotedColumn, placeholder))
			queryParams = append(queryParams, filter["value"])
			paramIndex++
		case "lte":
			whereClauses = append(whereClauses, fmt.Sprintf("%s <= %s", quotedColumn, placeholder))
			queryParams = append(queryParams, filter["value"])
			paramIndex++
		case "contains":
			value, ok := filter["value"].(string)
			if !ok {
				return nil, nil, ErrContainsRequiresString
			}
			whereClauses = append(whereClauses, fmt.Sprintf("%s %s %s", quotedColumn, likeOp, placeholder))
			queryParams = append(queryParams, "%"+value+"%")
			paramIndex++
		case "starts_with":
			value, ok := filter["value"].(string)
			if !ok {
				return nil, nil, ErrStartsWithRequiresString
			}
			whereClauses = append(whereClauses, fmt.Sprintf("%s %s %s", quotedColumn, likeOp, placeholder))
			queryParams = append(queryParams, value+"%")
			paramIndex++
		case "ends_with":
			value, ok := filter["value"].(string)
			if !ok {
				return nil, nil, ErrEndsWithRequiresString
			}
			whereClauses = append(whereClauses, fmt.Sprintf("%s %s %s", quotedColumn, likeOp, placeholder))
			queryParams = append(queryParams, "%"+value)
			paramIndex++
		case "is_null":
			whereClauses = append(whereClauses, fmt.Sprintf("%s IS NULL", quotedColumn))
		case "is_not_null":
			whereClauses = append(whereClauses, fmt.Sprintf("%s IS NOT NULL", quotedColumn))
		default:
			return nil, nil, fmt.Errorf("%w: %s", ErrInvalidOperator, operator)
		}
	}

	return whereClauses, queryParams, nil
}

func (s *DbMCPServer) countRows(ctx context.Context, schema, tableName, whereClause string, params []interface{}) (int, error) {
	query := s.queryBuilder.BuildCountQuery(schema, tableName, whereClause)

	var count int
	err := s.db.QueryRowContext(ctx, query, params...).Scan(&count)
	return count, err
}

func (s *DbMCPServer) fetchRows(ctx context.Context, schema, tableName string, columns []string, whereClause, orderBy, orderDirection string, pagination PaginationParams, params []interface{}) ([]map[string]interface{}, error) {
	query := s.queryBuilder.BuildSelectQuery(SelectQueryParams{
		Schema:         schema,
		Table:          tableName,
		Columns:        columns,
		WhereClause:    whereClause,
		OrderBy:        orderBy,
		OrderDirection: orderDirection,
		Limit:          pagination.PageSize,
		Offset:         pagination.Offset,
	})

	dbRows, err := s.db.QueryContext(ctx, query, params...)
	if err != nil {
		return nil, err
	}
	defer dbRows.Close()

	var rows []map[string]interface{}
	for dbRows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err = dbRows.Scan(valuePtrs...); err != nil {
			continue
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			row[col] = formatValue(values[i])
		}
		rows = append(rows, row)
	}

	return rows, nil
}

func (s *DbMCPServer) toolGetTableSchemaFull() (mcp.Tool, server.ToolHandlerFunc) {
	return mcp.Tool{
		Name:        "get_table_schema_full",
		Description: "Returns complete information about a table's structure, including columns, indexes, foreign keys, and constraints",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"table_name": map[string]interface{}{
					"type":        "string",
					"description": "Table name",
				},
				"schema": map[string]interface{}{
					"type":        "string",
					"description": "Schema name (optional)",
				},
			},
			Required: []string{"table_name"},
		},
	}, s.handleGetTableSchemaFull
}

func (s *DbMCPServer) handleGetTableSchemaFull(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := s.requireConnection(); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	args, ok := getArgs(request.Params.Arguments)
	if !ok {
		return mcp.NewToolResultError(ErrInvalidArguments.Error()), nil
	}

	tableName, ok := getStringArg(args, "table_name")
	if !ok || !isValidIdentifier(tableName) {
		return mcp.NewToolResultError(ErrInvalidTableName.Error()), nil
	}

	defaultSchema := getDefaultSchema(s.queryBuilder.GetDriver())
	schema, err := getValidSchema(args, defaultSchema)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	ctx, cancel := context.WithTimeout(ctx, DefaultQueryTimeout)
	defer cancel()

	// Get columns
	columnsQuery, columnsArgs := s.queryBuilder.GetTableSchemaFullQuery(schema, tableName)
	columns, err := s.fetchSchemaColumns(ctx, columnsQuery, columnsArgs)
	if err != nil {
		return mcp.NewToolResultError(fmt.Errorf("%w: %v", ErrRetrievingColumns, err).Error()), nil
	}

	if len(columns) == 0 {
		return mcp.NewToolResultError(fmt.Errorf("%w: %s.%s", ErrTableNotFound, schema, tableName).Error()), nil
	}

	// Get indexes
	indexesQuery, indexesArgs := s.queryBuilder.GetIndexesQuery(schema, tableName)
	indexes, _ := s.fetchIndexes(ctx, indexesQuery, indexesArgs)

	// Get foreign keys
	fkQuery, fkArgs := s.queryBuilder.GetForeignKeysQuery(schema, tableName)
	foreignKeys, _ := s.fetchForeignKeys(ctx, fkQuery, fkArgs)

	// Get primary key
	pkQuery, pkArgs := s.queryBuilder.GetPrimaryKeyQuery(schema, tableName)
	primaryKey, _ := s.fetchPrimaryKey(ctx, pkQuery, pkArgs)

	response := map[string]interface{}{
		"schema":       schema,
		"table":        tableName,
		"columns":      columns,
		"primary_key":  primaryKey,
		"indexes":      indexes,
		"foreign_keys": foreignKeys,
	}

	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(ErrSerializingJSON.Error()), nil
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

func (s *DbMCPServer) fetchSchemaColumns(ctx context.Context, query string, args []interface{}) ([]map[string]interface{}, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []map[string]interface{}

	if s.queryBuilder.IsSQLite() {
		for rows.Next() {
			var cid int
			var name, dataType string
			var notNull, pk int
			var dfltValue sql.NullString

			if err := rows.Scan(&cid, &name, &dataType, &notNull, &dfltValue, &pk); err != nil {
				continue
			}

			col := map[string]interface{}{
				"name":           name,
				"type":           dataType,
				"nullable":       notNull == 0,
				"is_primary_key": pk == 1,
			}
			if dfltValue.Valid {
				col["default_value"] = dfltValue.String
			}
			columns = append(columns, col)
		}
	} else {
		for rows.Next() {
			var columnName, dataType string
			var maxLength, precision, scale sql.NullInt64
			var isNullable, isPrimaryKey string
			var defaultValue sql.NullString

			if err := rows.Scan(&columnName, &dataType, &maxLength, &precision, &scale, &isNullable, &defaultValue, &isPrimaryKey); err != nil {
				continue
			}

			col := map[string]interface{}{
				"name":           columnName,
				"type":           dataType,
				"nullable":       strings.EqualFold(isNullable, "YES") || strings.EqualFold(isNullable, "Y"),
				"is_primary_key": strings.EqualFold(isPrimaryKey, "YES"),
			}
			if maxLength.Valid {
				col["max_length"] = maxLength.Int64
			}
			if precision.Valid {
				col["precision"] = precision.Int64
			}
			if scale.Valid {
				col["scale"] = scale.Int64
			}
			if defaultValue.Valid {
				col["default_value"] = defaultValue.String
			}
			columns = append(columns, col)
		}
	}

	return columns, nil
}

func (s *DbMCPServer) fetchIndexes(ctx context.Context, query string, args []interface{}) ([]map[string]interface{}, error) {
	if s.queryBuilder.IsSQLite() {
		return s.fetchSQLiteIndexes(ctx, query)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var indexes []map[string]interface{}
	for rows.Next() {
		var indexName, indexType, columnName string
		var isUnique bool

		if err := rows.Scan(&indexName, &indexType, &isUnique, &columnName); err != nil {
			continue
		}

		indexes = append(indexes, map[string]interface{}{
			"name":      indexName,
			"type":      indexType,
			"is_unique": isUnique,
			"column":    columnName,
		})
	}
	return indexes, nil
}

func (s *DbMCPServer) fetchSQLiteIndexes(ctx context.Context, tableName string) ([]map[string]interface{}, error) {
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf("PRAGMA index_list(%s)", tableName))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var indexes []map[string]interface{}
	for rows.Next() {
		var seq int
		var name, unique, origin, partial string

		if err := rows.Scan(&seq, &name, &unique, &origin, &partial); err != nil {
			continue
		}

		indexes = append(indexes, map[string]interface{}{
			"name":      name,
			"is_unique": unique == "1",
			"origin":    origin,
		})
	}
	return indexes, nil
}

func (s *DbMCPServer) fetchForeignKeys(ctx context.Context, query string, args []interface{}) ([]map[string]interface{}, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var foreignKeys []map[string]interface{}

	if s.queryBuilder.IsSQLite() {
		for rows.Next() {
			var id, seq int
			var table, from, to, onUpdate, onDelete, match string

			if err := rows.Scan(&id, &seq, &table, &from, &to, &onUpdate, &onDelete, &match); err != nil {
				continue
			}

			foreignKeys = append(foreignKeys, map[string]interface{}{
				"column":            from,
				"referenced_table":  table,
				"referenced_column": to,
			})
		}
	} else {
		for rows.Next() {
			var constraintName, columnName, refSchema, refTable, refColumn string

			if err := rows.Scan(&constraintName, &columnName, &refSchema, &refTable, &refColumn); err != nil {
				continue
			}

			foreignKeys = append(foreignKeys, map[string]interface{}{
				"name":              constraintName,
				"column":            columnName,
				"referenced_schema": refSchema,
				"referenced_table":  refTable,
				"referenced_column": refColumn,
			})
		}
	}

	return foreignKeys, nil
}

func (s *DbMCPServer) fetchPrimaryKey(ctx context.Context, query string, args []interface{}) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pkColumns []string

	if s.queryBuilder.IsSQLite() {
		for rows.Next() {
			var cid int
			var name, dataType string
			var notNull, pk int
			var dfltValue sql.NullString

			if err := rows.Scan(&cid, &name, &dataType, &notNull, &dfltValue, &pk); err != nil {
				continue
			}
			if pk > 0 {
				pkColumns = append(pkColumns, name)
			}
		}
	} else {
		for rows.Next() {
			var columnName string
			if err := rows.Scan(&columnName); err != nil {
				continue
			}
			pkColumns = append(pkColumns, columnName)
		}
	}

	return pkColumns, nil
}

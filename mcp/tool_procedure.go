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

func (s *DbMCPServer) toolListProcedures() (mcp.Tool, server.ToolHandlerFunc) {
	return mcp.Tool{
		Name:        "list_procedures",
		Description: "List database stored procedures with pagination",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"schema": map[string]interface{}{
					"type":        "string",
					"description": "Schema name (optional)",
				},
				"name_filter": map[string]interface{}{
					"type":        "string",
					"description": "Filter by procedure name (optional)",
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
	}, s.handleListProcedures
}

func (s *DbMCPServer) handleListProcedures(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := s.requireConnection(); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if !s.queryBuilder.SupportsStoredProcedures() {
		return mcp.NewToolResultError(ErrStoredProceduresNotSupported.Error()), nil
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

	query, queryArgs := s.queryBuilder.ListProceduresQuery(schema, nameFilter, pagination.PageSize, pagination.Offset)
	if query == "" {
		return mcp.NewToolResultError(ErrStoredProceduresNotSupported.Error()), nil
	}

	ctx, cancel := context.WithTimeout(ctx, ShortQueryTimeout)
	defer cancel()

	rows, err := s.db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Errorf("%w: %v", ErrListingProcedures, err).Error()), nil
	}
	defer rows.Close()

	var procedures []map[string]interface{}
	for rows.Next() {
		var routineSchema, routineName string
		var created, lastAltered sql.NullTime

		if err = rows.Scan(&routineSchema, &routineName, &created, &lastAltered); err != nil {
			continue
		}

		proc := map[string]interface{}{
			"schema": routineSchema,
			"name":   routineName,
		}
		if created.Valid {
			proc["created"] = created.Time.Format("2006-01-02 15:04:05")
		}
		if lastAltered.Valid {
			proc["last_altered"] = lastAltered.Time.Format("2006-01-02 15:04:05")
		}
		procedures = append(procedures, proc)
	}

	response := map[string]interface{}{
		"procedures": procedures,
		"pagination": map[string]interface{}{
			"page":      pagination.Page,
			"page_size": pagination.PageSize,
			"count":     len(procedures),
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

func (s *DbMCPServer) toolGetProcedureCode() (mcp.Tool, server.ToolHandlerFunc) {
	return mcp.Tool{
		Name:        "get_procedure_code",
		Description: "Returns the full source code of a stored procedure",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"procedure_name": map[string]interface{}{
					"type":        "string",
					"description": "Stored procedure name",
				},
				"schema": map[string]interface{}{
					"type":        "string",
					"description": "Schema name (optional)",
				},
			},
			Required: []string{"procedure_name"},
		},
	}, s.handleGetProcedureCode
}

func (s *DbMCPServer) handleGetProcedureCode(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := s.requireConnection(); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if !s.queryBuilder.SupportsStoredProcedures() {
		return mcp.NewToolResultError(ErrStoredProceduresNotSupported.Error()), nil
	}

	args, ok := getArgs(request.Params.Arguments)
	if !ok {
		return mcp.NewToolResultError(ErrInvalidArguments.Error()), nil
	}

	procedureName, ok := getStringArg(args, "procedure_name")
	if !ok || !isValidIdentifier(procedureName) {
		return mcp.NewToolResultError(ErrInvalidProcedureName.Error()), nil
	}

	defaultSchema := getDefaultSchema(s.queryBuilder.GetDriver())
	schema, err := getValidSchema(args, defaultSchema)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	query, queryArgs := s.queryBuilder.GetProcedureCodeQuery(schema, procedureName)

	ctx, cancel := context.WithTimeout(ctx, ShortQueryTimeout)
	defer cancel()

	// For Oracle, we need to collect all lines
	if s.queryBuilder.IsOracle() {
		return s.getOracleSourceCode(ctx, query, queryArgs, "procedure")
	}

	var definition sql.NullString
	err = s.db.QueryRowContext(ctx, query, queryArgs...).Scan(&definition)
	if err == sql.ErrNoRows {
		return mcp.NewToolResultError(ErrProcedureNotFound.Error()), nil
	}
	if err != nil {
		return mcp.NewToolResultError(fmt.Errorf("%w: %v", ErrFetchingCode, err).Error()), nil
	}

	if !definition.Valid || definition.String == "" {
		return mcp.NewToolResultError(ErrCodeNotAvailable.Error()), nil
	}

	response := map[string]interface{}{
		"schema":     schema,
		"name":       procedureName,
		"definition": definition.String,
	}

	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(ErrSerializingJSON.Error()), nil
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

func (s *DbMCPServer) toolExecuteProcedure() (mcp.Tool, server.ToolHandlerFunc) {
	return mcp.Tool{
		Name:        "execute_procedure",
		Description: "Execute a stored procedure with parameters",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"procedure_name": map[string]interface{}{
					"type":        "string",
					"description": "Stored procedure name",
				},
				"parameters": map[string]interface{}{
					"type":        "object",
					"description": "Procedure parameters as a JSON object",
				},
				"schema": map[string]interface{}{
					"type":        "string",
					"description": "Schema name (optional)",
				},
			},
			Required: []string{"procedure_name"},
		},
	}, s.handleExecuteProcedure
}

func (s *DbMCPServer) handleExecuteProcedure(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := s.requireConnection(); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if !s.queryBuilder.SupportsStoredProcedures() {
		return mcp.NewToolResultError(ErrStoredProceduresNotSupported.Error()), nil
	}

	args, ok := getArgs(request.Params.Arguments)
	if !ok {
		return mcp.NewToolResultError(ErrInvalidArguments.Error()), nil
	}

	procedureName, ok := getStringArg(args, "procedure_name")
	if !ok || !isValidIdentifier(procedureName) {
		return mcp.NewToolResultError(ErrInvalidProcedureName.Error()), nil
	}

	defaultSchema := getDefaultSchema(s.queryBuilder.GetDriver())
	schema, err := getValidSchema(args, defaultSchema)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	userParams := make(map[string]interface{})
	if p, ok := args["parameters"].(map[string]interface{}); ok {
		userParams = p
	}

	ctx, cancel := context.WithTimeout(ctx, DefaultQueryTimeout)
	defer cancel()

	// Build and execute the procedure call based on driver
	var execSQL string
	var paramValues []interface{}

	qualifiedName := s.queryBuilder.QualifyTable(schema, procedureName)

	switch s.queryBuilder.GetDriver() {
	case DriverSQLServer:
		execSQL, paramValues = s.buildSQLServerProcedureCall(qualifiedName, userParams)
	case DriverPostgresSQL:
		execSQL, paramValues = s.buildPostgresProcedureCall(qualifiedName, userParams)
	case DriverMySQL:
		execSQL, paramValues = s.buildMySQLProcedureCall(qualifiedName, userParams)
	case DriverOracle:
		execSQL, paramValues = s.buildOracleProcedureCall(qualifiedName, userParams)
	default:
		return mcp.NewToolResultError(ErrFeatureNotSupported.Error()), nil
	}

	resultRows, err := s.db.QueryContext(ctx, execSQL, paramValues...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Errorf("%w: %v", ErrExecutingProcedure, err).Error()), nil
	}
	defer resultRows.Close()

	columns, err := resultRows.Columns()
	if err != nil {
		return mcp.NewToolResultText("Procedure executed successfully (no results)"), nil
	}

	var results []map[string]interface{}
	for resultRows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err = resultRows.Scan(valuePtrs...); err != nil {
			continue
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			row[col] = formatValue(values[i])
		}
		results = append(results, row)
	}

	response := map[string]interface{}{
		"status":    "success",
		"procedure": procedureName,
		"schema":    schema,
		"results":   results,
		"row_count": len(results),
	}

	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(ErrSerializingJSON.Error()), nil
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

func (s *DbMCPServer) buildSQLServerProcedureCall(qualifiedName string, params map[string]interface{}) (string, []interface{}) {
	var paramPlaceholders []string
	var paramValues []interface{}
	i := 1

	for name, value := range params {
		paramPlaceholders = append(paramPlaceholders, fmt.Sprintf("@%s = @p%d", name, i))
		paramValues = append(paramValues, value)
		i++
	}

	execSQL := fmt.Sprintf("EXEC %s", qualifiedName)
	if len(paramPlaceholders) > 0 {
		execSQL += " " + strings.Join(paramPlaceholders, ", ")
	}

	return execSQL, paramValues
}

func (s *DbMCPServer) buildPostgresProcedureCall(qualifiedName string, params map[string]interface{}) (string, []interface{}) {
	var placeholders []string
	var paramValues []interface{}
	i := 1

	for _, value := range params {
		placeholders = append(placeholders, fmt.Sprintf("$%d", i))
		paramValues = append(paramValues, value)
		i++
	}

	execSQL := fmt.Sprintf("CALL %s(%s)", qualifiedName, strings.Join(placeholders, ", "))
	return execSQL, paramValues
}

func (s *DbMCPServer) buildMySQLProcedureCall(qualifiedName string, params map[string]interface{}) (string, []interface{}) {
	var placeholders []string
	var paramValues []interface{}

	for _, value := range params {
		placeholders = append(placeholders, "?")
		paramValues = append(paramValues, value)
	}

	execSQL := fmt.Sprintf("CALL %s(%s)", qualifiedName, strings.Join(placeholders, ", "))
	return execSQL, paramValues
}

func (s *DbMCPServer) buildOracleProcedureCall(qualifiedName string, params map[string]interface{}) (string, []interface{}) {
	var placeholders []string
	var paramValues []interface{}
	i := 1

	for _, value := range params {
		placeholders = append(placeholders, fmt.Sprintf(":%d", i))
		paramValues = append(paramValues, value)
		i++
	}

	execSQL := fmt.Sprintf("BEGIN %s(%s); END;", qualifiedName, strings.Join(placeholders, ", "))
	return execSQL, paramValues
}

// getOracleSourceCode retrieves source code that spans multiple lines (Oracle specific)
func (s *DbMCPServer) getOracleSourceCode(ctx context.Context, query string, args []interface{}, objectType string) (*mcp.CallToolResult, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Errorf("%w: %v", ErrFetchingCode, err).Error()), nil
	}
	defer rows.Close()

	var lines []string
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			continue
		}
		lines = append(lines, line)
	}

	if len(lines) == 0 {
		return mcp.NewToolResultError(ErrCodeNotAvailable.Error()), nil
	}

	response := map[string]interface{}{
		"definition": strings.Join(lines, ""),
	}

	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(ErrSerializingJSON.Error()), nil
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

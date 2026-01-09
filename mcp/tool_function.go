package mcp

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func (s *DbMCPServer) toolListFunctions() (mcp.Tool, server.ToolHandlerFunc) {
	return mcp.Tool{
		Name:        "list_functions",
		Description: "List database functions (scalar, table-valued) with pagination",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"schema": map[string]interface{}{
					"type":        "string",
					"description": "Schema name (optional)",
				},
				"type": map[string]interface{}{
					"type":        "string",
					"description": "Function type: 'scalar', 'table' or 'all' (default: all)",
				},
				"name_filter": map[string]interface{}{
					"type":        "string",
					"description": "Filter by function name (optional)",
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
	}, s.handleListFunctions
}

func (s *DbMCPServer) handleListFunctions(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := s.requireConnection(); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if !s.queryBuilder.SupportsFunctions() {
		return mcp.NewToolResultError(ErrFunctionsNotSupported.Error()), nil
	}

	args, ok := getArgs(request.Params.Arguments)
	if !ok {
		return mcp.NewToolResultError(ErrInvalidArguments.Error()), nil
	}

	schema, err := getValidSchema(args, "")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	funcType, _ := getStringArg(args, "type")
	if funcType == "" {
		funcType = "all"
	}

	switch funcType {
	case "scalar", "table", "all":
		// valid
	default:
		return mcp.NewToolResultError(ErrInvalidFunctionType.Error()), nil
	}

	nameFilter, _ := getStringArg(args, "name_filter")
	pagination := GetPaginationParams(args, DefaultPageSize, MaxPageSize)

	query, queryArgs := s.queryBuilder.ListFunctionsQuery(schema, nameFilter, funcType, pagination.PageSize, pagination.Offset)
	if query == "" {
		return mcp.NewToolResultError(ErrFunctionsNotSupported.Error()), nil
	}

	ctx, cancel := context.WithTimeout(ctx, ShortQueryTimeout)
	defer cancel()

	rows, err := s.db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Errorf("%w: %v", ErrListingFunctions, err).Error()), nil
	}
	defer rows.Close()

	var functions []map[string]interface{}
	for rows.Next() {
		var routineSchema, routineName, functionType string
		var created, lastAltered sql.NullTime

		if err = rows.Scan(&routineSchema, &routineName, &functionType, &created, &lastAltered); err != nil {
			continue
		}

		fn := map[string]interface{}{
			"schema": routineSchema,
			"name":   routineName,
			"type":   functionType,
		}
		if created.Valid {
			fn["created"] = created.Time.Format("2006-01-02 15:04:05")
		}
		if lastAltered.Valid {
			fn["last_altered"] = lastAltered.Time.Format("2006-01-02 15:04:05")
		}
		functions = append(functions, fn)
	}

	response := map[string]interface{}{
		"functions": functions,
		"pagination": map[string]interface{}{
			"page":      pagination.Page,
			"page_size": pagination.PageSize,
			"count":     len(functions),
		},
		"filter": map[string]interface{}{
			"schema":      schema,
			"type":        funcType,
			"name_filter": nameFilter,
		},
	}

	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(ErrSerializingJSON.Error()), nil
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

func (s *DbMCPServer) toolGetFunctionCode() (mcp.Tool, server.ToolHandlerFunc) {
	return mcp.Tool{
		Name:        "get_function_code",
		Description: "Returns the full source code of a function",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"function_name": map[string]interface{}{
					"type":        "string",
					"description": "Function name",
				},
				"schema": map[string]interface{}{
					"type":        "string",
					"description": "Schema name (optional)",
				},
			},
			Required: []string{"function_name"},
		},
	}, s.handleGetFunctionCode
}

func (s *DbMCPServer) handleGetFunctionCode(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := s.requireConnection(); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if !s.queryBuilder.SupportsFunctions() {
		return mcp.NewToolResultError(ErrFunctionsNotSupported.Error()), nil
	}

	args, ok := getArgs(request.Params.Arguments)
	if !ok {
		return mcp.NewToolResultError(ErrInvalidArguments.Error()), nil
	}

	functionName, ok := getStringArg(args, "function_name")
	if !ok || !isValidIdentifier(functionName) {
		return mcp.NewToolResultError(ErrInvalidFunctionName.Error()), nil
	}

	defaultSchema := getDefaultSchema(s.queryBuilder.GetDriver())
	schema, err := getValidSchema(args, defaultSchema)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	query, queryArgs := s.queryBuilder.GetFunctionCodeQuery(schema, functionName)

	ctx, cancel := context.WithTimeout(ctx, ShortQueryTimeout)
	defer cancel()

	// For Oracle, we need to collect all lines
	if s.queryBuilder.IsOracle() {
		return s.getOracleSourceCode(ctx, query, queryArgs, "function")
	}

	var definition sql.NullString
	err = s.db.QueryRowContext(ctx, query, queryArgs...).Scan(&definition)
	if err == sql.ErrNoRows {
		return mcp.NewToolResultError(ErrFunctionNotFound.Error()), nil
	}
	if err != nil {
		return mcp.NewToolResultError(fmt.Errorf("%w: %v", ErrFetchingCode, err).Error()), nil
	}

	if !definition.Valid || definition.String == "" {
		return mcp.NewToolResultError(ErrCodeNotAvailable.Error()), nil
	}

	response := map[string]interface{}{
		"schema":     schema,
		"name":       functionName,
		"definition": definition.String,
	}

	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(ErrSerializingJSON.Error()), nil
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

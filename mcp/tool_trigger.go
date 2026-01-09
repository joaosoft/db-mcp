package mcp

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func (s *DbMCPServer) toolListTriggers() (mcp.Tool, server.ToolHandlerFunc) {
	return mcp.Tool{
		Name:        "list_triggers",
		Description: "List database triggers with pagination",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"table_name": map[string]interface{}{
					"type":        "string",
					"description": "Table name (optional, if not specified, lists all)",
				},
				"schema": map[string]interface{}{
					"type":        "string",
					"description": "Schema name (optional)",
				},
				"name_filter": map[string]interface{}{
					"type":        "string",
					"description": "Filter by trigger name (optional)",
				},
				"page": map[string]interface{}{
					"type":        "number",
					"description": "Page number (default: 1)",
				},
				"page_size": map[string]interface{}{
					"type":        "number",
					"description": "Items per page (default: 100, maximum: 500)",
				},
				"include_disabled": map[string]interface{}{
					"type":        "boolean",
					"description": "Include disabled triggers (default: true)",
				},
			},
		},
	}, s.handleListTriggers
}

func (s *DbMCPServer) handleListTriggers(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

	tableName, _ := getStringArg(args, "table_name")
	nameFilter, _ := getStringArg(args, "name_filter")
	includeDisabled := getBoolArg(args, "include_disabled", true)
	pagination := GetPaginationParams(args, DefaultPageSize, MaxPageSize)

	query, queryArgs := s.queryBuilder.ListTriggersQuery(schema, tableName, nameFilter, includeDisabled, pagination.PageSize, pagination.Offset)

	ctx, cancel := context.WithTimeout(ctx, ShortQueryTimeout)
	defer cancel()

	rows, err := s.db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Errorf("%w: %v", ErrListingTriggers, err).Error()), nil
	}
	defer rows.Close()

	var triggers []map[string]interface{}
	for rows.Next() {
		var schemaName, triggerName, table string
		var isDisabled bool
		var createDate, modifyDate sql.NullTime

		if err = rows.Scan(&schemaName, &triggerName, &table, &isDisabled, &createDate, &modifyDate); err != nil {
			continue
		}

		trigger := map[string]interface{}{
			"schema":      schemaName,
			"name":        triggerName,
			"table":       table,
			"is_disabled": isDisabled,
		}
		if createDate.Valid {
			trigger["created"] = createDate.Time.Format("2006-01-02 15:04:05")
		}
		if modifyDate.Valid {
			trigger["last_altered"] = modifyDate.Time.Format("2006-01-02 15:04:05")
		}
		triggers = append(triggers, trigger)
	}

	response := map[string]interface{}{
		"triggers": triggers,
		"pagination": map[string]interface{}{
			"page":      pagination.Page,
			"page_size": pagination.PageSize,
			"count":     len(triggers),
		},
		"filter": map[string]interface{}{
			"schema":           schema,
			"table":            tableName,
			"name_filter":      nameFilter,
			"include_disabled": includeDisabled,
		},
	}

	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(ErrSerializingJSON.Error()), nil
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

func (s *DbMCPServer) toolGetTriggerCode() (mcp.Tool, server.ToolHandlerFunc) {
	return mcp.Tool{
		Name:        "get_trigger_code",
		Description: "Returns the complete source code of a trigger",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"trigger_name": map[string]interface{}{
					"type":        "string",
					"description": "Trigger name",
				},
				"schema": map[string]interface{}{
					"type":        "string",
					"description": "Schema name (optional)",
				},
			},
			Required: []string{"trigger_name"},
		},
	}, s.handleGetTriggerCode
}

func (s *DbMCPServer) handleGetTriggerCode(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := s.requireConnection(); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	args, ok := getArgs(request.Params.Arguments)
	if !ok {
		return mcp.NewToolResultError(ErrInvalidArguments.Error()), nil
	}

	triggerName, ok := getStringArg(args, "trigger_name")
	if !ok || !isValidIdentifier(triggerName) {
		return mcp.NewToolResultError(ErrInvalidTriggerName.Error()), nil
	}

	defaultSchema := getDefaultSchema(s.queryBuilder.GetDriver())
	schema, err := getValidSchema(args, defaultSchema)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	query, queryArgs := s.queryBuilder.GetTriggerCodeQuery(schema, triggerName)

	ctx, cancel := context.WithTimeout(ctx, ShortQueryTimeout)
	defer cancel()

	var definition sql.NullString
	err = s.db.QueryRowContext(ctx, query, queryArgs...).Scan(&definition)

	if err == sql.ErrNoRows {
		return mcp.NewToolResultError(ErrTriggerNotFound.Error()), nil
	}
	if err != nil {
		return mcp.NewToolResultError(fmt.Errorf("%w: %v", ErrRetrievingTrigger, err).Error()), nil
	}

	if !definition.Valid || definition.String == "" {
		return mcp.NewToolResultError(ErrCodeNotAvailable.Error()), nil
	}

	response := map[string]interface{}{
		"schema":     schema,
		"name":       triggerName,
		"definition": definition.String,
	}

	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(ErrSerializingJSON.Error()), nil
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

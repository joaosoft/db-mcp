package mcp

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func (s *DatabaseMCP) toolListTriggers() (mcp.Tool, server.ToolHandlerFunc) {
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
					"description": "Scheme name (optional)",
				},
				"name_filter": map[string]interface{}{
					"type":        "string",
					"description": "Filter for trigger name (uses ILIKE, optional)",
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

func (s *DatabaseMCP) handleListTriggers(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return mcp.NewToolResultError("Invalid arguments"), nil
	}

	schema, _ := getValidSchema(args, "")
	tableName, _ := args["table_name"].(string)
	nameFilter, _ := args["name_filter"].(string)

	// Pagination parameters
	page := 1
	pageSize := 100

	if pageVal, ok := args["page"].(float64); ok {
		page = int(pageVal)
		if page < 1 {
			page = 1
		}
	}

	if pageSizeVal, ok := args["page_size"].(float64); ok {
		pageSize = int(pageSizeVal)
		if pageSize < 1 {
			pageSize = 100
		}
		if pageSize > 500 {
			pageSize = 500
		}
	}

	// Include disabled triggers (default: true)
	includeDisabled := true
	if includeDisabledVal, ok := args["include_disabled"].(bool); ok {
		includeDisabled = includeDisabledVal
	}

	// Calculate offset
	offset := (page - 1) * pageSize

	// Use query builder to generate database-specific query
	query, queryArgs := s.queryBuilder.ListTriggersQuery(schema, tableName, nameFilter, includeDisabled, pageSize, offset)

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Execute main query
	rows, err := s.db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error listing triggers: %v", err)), nil
	}
	defer rows.Close()

	var triggers []map[string]interface{}
	for rows.Next() {
		var schemaName, triggerName, table string
		var isDisabled, isUpdate, isDelete, isInsert, isInsteadOf bool
		var createDate, modifyDate time.Time

		if err = rows.Scan(&schemaName, &triggerName, &table, &isDisabled, &createDate, &modifyDate,
			&isUpdate, &isDelete, &isInsert, &isInsteadOf); err != nil {
			continue
		}

		// Determine the events that activate the trigger
		var events []string
		if isInsert {
			events = append(events, "INSERT")
		}
		if isUpdate {
			events = append(events, "UPDATE")
		}
		if isDelete {
			events = append(events, "DELETE")
		}

		// Determine trigger type
		triggerType := "AFTER"
		if isInsteadOf {
			triggerType = "INSTEAD OF"
		}

		trigger := map[string]interface{}{
			"schema":       schemaName,
			"name":         triggerName,
			"table":        table,
			"type":         triggerType,
			"events":       events,
			"is_disabled":  isDisabled,
			"created":      createDate.Format("2006-01-02 15:04:05"),
			"last_altered": modifyDate.Format("2006-01-02 15:04:05"),
		}
		triggers = append(triggers, trigger)
	}

	// Response with pagination metadata
	response := map[string]interface{}{
		"triggers": triggers,
		"pagination": map[string]interface{}{
			"page":         page,
			"page_size":    pageSize,
			"has_previous": page > 1,
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
		return mcp.NewToolResultError(fmt.Sprintf("Error serializing JSON: %v", err)), nil
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

func (s *DatabaseMCP) toolGetTriggerCode() (mcp.Tool, server.ToolHandlerFunc) {
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
					"description": "Schema name (optional, default: dbo)",
				},
			},
			Required: []string{"trigger_name"},
		},
	}, s.handleGetTriggerCode
}

func (s *DatabaseMCP) handleGetTriggerCode(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return mcp.NewToolResultError("Invalid arguments"), nil
	}

	triggerName, ok := args["trigger_name"].(string)
	if !ok || triggerName == "" {
		return mcp.NewToolResultError("Trigger name is required"), nil
	}

	schema, err := getValidSchema(args, "dbo")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Use query builder to generate database-specific query
	query, queryArgs := s.queryBuilder.GetTriggerCodeQuery(schema, triggerName)

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var schemaName, name, tableName, definition string
	err = s.db.QueryRowContext(ctx, query, queryArgs...).Scan(&schemaName, &name, &tableName, &definition)

	if err == sql.ErrNoRows {
		return mcp.NewToolResultError(fmt.Sprintf("Trigger '%s.%s' not found", schema, triggerName)), nil
	}
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error retrieving trigger code.: %v", err)), nil
	}

	response := map[string]interface{}{
		"schema":     schemaName,
		"name":       name,
		"table":      tableName,
		"definition": definition,
	}

	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error serializing JSON: %v", err)), nil
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

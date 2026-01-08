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

func (s *DatabaseMCP) toolListViews() (mcp.Tool, server.ToolHandlerFunc) {
	return mcp.Tool{
		Name:        "list_views",
		Description: "List of database views with pagination",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"schema": map[string]interface{}{
					"type":        "string",
					"description": "Scheme name (optional)",
				},
				"name_filter": map[string]interface{}{
					"type":        "string",
					"description": "Filter for view name (uses ILIKE, optional)",
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
	}, s.handleListViews
}

func (s *DatabaseMCP) handleListViews(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return mcp.NewToolResultError("Invalid arguments"), nil
	}

	schema, err := getValidSchema(args, "")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	nameFilter, _ := args["name_filter"].(string)

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

	offset := (page - 1) * pageSize

	// Use query builder to generate database-specific query
	query, queryArgs := s.queryBuilder.ListViewsQuery(schema, nameFilter, pageSize, offset)

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	rows, err := s.db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error listing views: %v", err)), nil
	}
	defer rows.Close()

	var views []map[string]interface{}
	for rows.Next() {
		var viewSchema, viewName string
		var created, lastAltered time.Time
		var isSystem, hasDependencies bool

		if err = rows.Scan(&viewSchema, &viewName, &created, &lastAltered, &isSystem, &hasDependencies); err != nil {
			continue
		}

		view := map[string]interface{}{
			"schema":           viewSchema,
			"name":             viewName,
			"created":          created.Format("2006-01-02 15:04:05"),
			"last_altered":     lastAltered.Format("2006-01-02 15:04:05"),
			"is_system":        isSystem,
			"has_dependencies": hasDependencies,
		}
		views = append(views, view)
	}

	response := map[string]interface{}{
		"views": views,
		"pagination": map[string]interface{}{
			"page":         page,
			"page_size":    pageSize,
			"has_previous": page > 1,
		},
		"filter": map[string]interface{}{
			"schema":      schema,
			"name_filter": nameFilter,
		},
	}

	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error serializing JSON: %v", err)), nil
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

func (s *DatabaseMCP) toolGetViewDefinition() (mcp.Tool, server.ToolHandlerFunc) {
	return mcp.Tool{
		Name:        "get_view_definition",
		Description: "Returns the SQL definition/code of a view.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"view_name": map[string]interface{}{
					"type":        "string",
					"description": "View name",
				},
				"schema": map[string]interface{}{
					"type":        "string",
					"description": "Schema name (optional, default: dbo)",
				},
			},
			Required: []string{"view_name"},
		},
	}, s.handleGetViewDefinition
}

func (s *DatabaseMCP) handleGetViewDefinition(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return mcp.NewToolResultError("Invalid arguments"), nil
	}

	viewName, ok := args["view_name"].(string)
	if !ok || viewName == "" {
		return mcp.NewToolResultError("View name is required"), nil
	}

	schema, err := getValidSchema(args, "dbo")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Use query builder to generate database-specific query
	query, queryArgs := s.queryBuilder.GetViewDefinitionQuery(schema, viewName)

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var schemaName, name, definition string
	var createDate, modifyDate time.Time

	err = s.db.QueryRowContext(ctx, query, queryArgs...).Scan(
		&schemaName, &name, &createDate, &modifyDate, &definition,
	)

	if err == sql.ErrNoRows {
		return mcp.NewToolResultError(fmt.Sprintf("View '%s.%s' not found", schema, viewName)), nil
	}
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error retrieving view definition: %v", err)), nil
	}

	response := map[string]interface{}{
		"schema":       schemaName,
		"name":         name,
		"created":      createDate.Format("2006-01-02 15:04:05"),
		"last_altered": modifyDate.Format("2006-01-02 15:04:05"),
		"definition":   definition,
	}

	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error serializing JSON: %v", err)), nil
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

package mcp

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func (s *DbMCPServer) toolListViews() (mcp.Tool, server.ToolHandlerFunc) {
	return mcp.Tool{
		Name:        "list_views",
		Description: "List database views with pagination",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"schema": map[string]interface{}{
					"type":        "string",
					"description": "Schema name (optional)",
				},
				"name_filter": map[string]interface{}{
					"type":        "string",
					"description": "Filter by view name (optional)",
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

func (s *DbMCPServer) handleListViews(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

	query, queryArgs := s.queryBuilder.ListViewsQuery(schema, nameFilter, pagination.PageSize, pagination.Offset)

	ctx, cancel := context.WithTimeout(ctx, ShortQueryTimeout)
	defer cancel()

	rows, err := s.db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Errorf("%w: %v", ErrListingViews, err).Error()), nil
	}
	defer rows.Close()

	var views []map[string]interface{}
	for rows.Next() {
		var viewSchema, viewName string
		var created, lastAltered sql.NullTime

		if err = rows.Scan(&viewSchema, &viewName, &created, &lastAltered); err != nil {
			continue
		}

		view := map[string]interface{}{
			"schema": viewSchema,
			"name":   viewName,
		}
		if created.Valid {
			view["created"] = created.Time.Format("2006-01-02 15:04:05")
		}
		if lastAltered.Valid {
			view["last_altered"] = lastAltered.Time.Format("2006-01-02 15:04:05")
		}
		views = append(views, view)
	}

	response := map[string]interface{}{
		"views": views,
		"pagination": map[string]interface{}{
			"page":      pagination.Page,
			"page_size": pagination.PageSize,
			"count":     len(views),
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

func (s *DbMCPServer) toolGetViewDefinition() (mcp.Tool, server.ToolHandlerFunc) {
	return mcp.Tool{
		Name:        "get_view_definition",
		Description: "Returns the SQL definition of a view",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"view_name": map[string]interface{}{
					"type":        "string",
					"description": "View name",
				},
				"schema": map[string]interface{}{
					"type":        "string",
					"description": "Schema name (optional)",
				},
			},
			Required: []string{"view_name"},
		},
	}, s.handleGetViewDefinition
}

func (s *DbMCPServer) handleGetViewDefinition(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := s.requireConnection(); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	args, ok := getArgs(request.Params.Arguments)
	if !ok {
		return mcp.NewToolResultError(ErrInvalidArguments.Error()), nil
	}

	viewName, ok := getStringArg(args, "view_name")
	if !ok || !isValidIdentifier(viewName) {
		return mcp.NewToolResultError(ErrInvalidViewName.Error()), nil
	}

	defaultSchema := getDefaultSchema(s.queryBuilder.GetDriver())
	schema, err := getValidSchema(args, defaultSchema)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	query, queryArgs := s.queryBuilder.GetViewDefinitionQuery(schema, viewName)

	ctx, cancel := context.WithTimeout(ctx, ShortQueryTimeout)
	defer cancel()

	var definition sql.NullString
	err = s.db.QueryRowContext(ctx, query, queryArgs...).Scan(&definition)

	if err == sql.ErrNoRows {
		return mcp.NewToolResultError(ErrViewNotFound.Error()), nil
	}
	if err != nil {
		return mcp.NewToolResultError(fmt.Errorf("%w: %v", ErrRetrievingView, err).Error()), nil
	}

	if !definition.Valid || definition.String == "" {
		return mcp.NewToolResultError(ErrDefinitionNotAvailable.Error()), nil
	}

	response := map[string]interface{}{
		"schema":     schema,
		"name":       viewName,
		"definition": definition.String,
	}

	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(ErrSerializingJSON.Error()), nil
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

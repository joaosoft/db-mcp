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

func (s *DbMCPServer) toolSearchObjects() (mcp.Tool, server.ToolHandlerFunc) {
	return mcp.Tool{
		Name:        "search_objects",
		Description: "Search for objects (tables, views, procedures, functions) by name or in the source code",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"search_term": map[string]interface{}{
					"type":        "string",
					"description": "Search term",
				},
				"search_in_code": map[string]interface{}{
					"type":        "boolean",
					"description": "If true, also search in the source code of procedures/functions/views (default: false)",
				},
				"object_types": map[string]interface{}{
					"type":        "array",
					"description": "Object types: 'table', 'view', 'procedure', 'function' (default: all)",
					"items": map[string]interface{}{
						"type": "string",
					},
				},
			},
			Required: []string{"search_term"},
		},
	}, s.handleSearchObjects
}

func (s *DbMCPServer) handleSearchObjects(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := s.requireConnection(); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	args, ok := getArgs(request.Params.Arguments)
	if !ok {
		return mcp.NewToolResultError(ErrInvalidArguments.Error()), nil
	}

	searchTerm, ok := getStringArg(args, "search_term")
	if !ok || searchTerm == "" {
		return mcp.NewToolResultError(ErrSearchTermRequired.Error()), nil
	}

	searchInCode := getBoolArg(args, "search_in_code", false)

	var objectTypes []string
	if objectTypesArg, ok := args["object_types"].([]interface{}); ok && len(objectTypesArg) > 0 {
		for _, ot := range objectTypesArg {
			if otStr, ok := ot.(string); ok {
				objectTypes = append(objectTypes, otStr)
			}
		}
	}

	query, queryArgs := s.queryBuilder.SearchObjectsQuery(searchTerm, searchInCode, objectTypes)

	ctx, cancel := context.WithTimeout(ctx, DefaultQueryTimeout)
	defer cancel()

	rows, err := s.db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Errorf("%w: %v", ErrSearchingObjects, err).Error()), nil
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var schemaName, objectName, objectType string
		var createDate, modifyDate sql.NullTime
		var hasCode bool

		if err = rows.Scan(&schemaName, &objectName, &objectType, &createDate, &modifyDate, &hasCode); err != nil {
			continue
		}

		result := map[string]interface{}{
			"schema":   schemaName,
			"name":     objectName,
			"type":     objectType,
			"has_code": hasCode,
		}
		if createDate.Valid {
			result["created"] = createDate.Time.Format("2006-01-02 15:04:05")
		}
		if modifyDate.Valid {
			result["last_altered"] = modifyDate.Time.Format("2006-01-02 15:04:05")
		}
		results = append(results, result)
	}

	response := map[string]interface{}{
		"results": results,
		"search": map[string]interface{}{
			"term":    searchTerm,
			"in_code": searchInCode,
			"count":   len(results),
		},
	}

	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(ErrSerializingJSON.Error()), nil
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

func (s *DbMCPServer) toolGetDatabaseInfo() (mcp.Tool, server.ToolHandlerFunc) {
	return mcp.Tool{
		Name:        "get_database_info",
		Description: "Returns general information about the database",
		InputSchema: mcp.ToolInputSchema{
			Type:       "object",
			Properties: map[string]interface{}{},
		},
	}, s.handleGetDatabaseInfo
}

func (s *DbMCPServer) handleGetDatabaseInfo(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := s.requireConnection(); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	ctx, cancel := context.WithTimeout(ctx, ShortQueryTimeout)
	defer cancel()

	response := map[string]interface{}{
		"driver": string(s.queryBuilder.GetDriver()),
	}

	// Database version
	var version string
	versionQuery := s.queryBuilder.GetDatabaseInfoQuery()
	err := s.db.QueryRowContext(ctx, versionQuery).Scan(&version)
	if err != nil {
		version = "Unknown"
	}
	response["version"] = version

	// Database details (if supported)
	if detailsQuery, supported := s.queryBuilder.GetDatabaseDetailsQuery(); supported {
		var dbName, collation, recoveryModel string
		var compatibilityLevel int
		var createDate interface{}

		err := s.db.QueryRowContext(ctx, detailsQuery).Scan(&dbName, &collation, &recoveryModel, &compatibilityLevel, &createDate)
		if err == nil {
			response["database_name"] = dbName
			response["collation"] = collation
			if recoveryModel != "" {
				response["recovery_model"] = recoveryModel
			}
			if compatibilityLevel > 0 {
				response["compatibility_level"] = compatibilityLevel
			}
			if createDate != nil {
				if t, ok := createDate.(time.Time); ok {
					response["created"] = t.Format("2006-01-02 15:04:05")
				}
			}
		}
	}

	// Count objects
	if countQuery, supported := s.queryBuilder.GetObjectCountsQuery(); supported {
		var tables, views, procedures, functions, triggers sql.NullInt64
		err = s.db.QueryRowContext(ctx, countQuery).Scan(&tables, &views, &procedures, &functions, &triggers)
		if err == nil {
			response["object_counts"] = map[string]interface{}{
				"tables":     nullInt64ToInt(tables),
				"views":      nullInt64ToInt(views),
				"procedures": nullInt64ToInt(procedures),
				"functions":  nullInt64ToInt(functions),
				"triggers":   nullInt64ToInt(triggers),
			}
		}
	}

	// List schemas
	if schemasQuery, supported := s.queryBuilder.GetSchemasListQuery(); supported {
		rows, err := s.db.QueryContext(ctx, schemasQuery)
		if err == nil {
			defer rows.Close()
			var schemas []string
			for rows.Next() {
				var schemaName string
				if err = rows.Scan(&schemaName); err == nil {
					schemas = append(schemas, schemaName)
				}
			}
			response["schemas"] = schemas
		}
	}

	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(ErrSerializingJSON.Error()), nil
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

func nullInt64ToInt(n sql.NullInt64) int {
	if n.Valid {
		return int(n.Int64)
	}
	return 0
}

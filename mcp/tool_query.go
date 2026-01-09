package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"
	"unicode/utf8"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func (s *DbMCPServer) toolExecuteQuery() (mcp.Tool, server.ToolHandlerFunc) {
	return mcp.Tool{
		Name:        "execute_query",
		Description: "Executes a SELECT query and returns the results. Only read-only queries are allowed.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "SQL query to be executed (SELECT only)",
				},
				"max_rows": map[string]interface{}{
					"type":        "number",
					"description": "Maximum number of rows to be returned (default: 100, max: 10000)",
				},
			},
			Required: []string{"query"},
		},
	}, s.handleExecuteQuery
}

func (s *DbMCPServer) handleExecuteQuery(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := s.requireConnection(); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	args, ok := getArgs(request.Params.Arguments)
	if !ok {
		return mcp.NewToolResultError(ErrInvalidArguments.Error()), nil
	}

	query, ok := getStringArg(args, "query")
	if !ok || query == "" {
		return mcp.NewToolResultError(ErrQueryRequired.Error()), nil
	}

	// Complete validation
	validator := NewSQLValidator(query)
	if err := validator.Validate(); err != nil {
		log.Printf("Query blocked: %s\nReason: %v\n", query, err)
		return mcp.NewToolResultError(fmt.Errorf("%w: %v", ErrQueryNotAllowed, err).Error()), nil
	}

	maxRows := getIntArg(args, "max_rows", 100)
	if maxRows <= 0 {
		maxRows = 100
	}
	if maxRows > 10000 {
		maxRows = 10000
	}

	ctx, cancel := context.WithTimeout(ctx, DefaultQueryTimeout)
	defer cancel()

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		log.Printf("Error in query: %v\nQuery: %s\n", err, query)
		return mcp.NewToolResultError(ErrQuerySyntax.Error()), nil
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return mcp.NewToolResultError(ErrRetrievingColumns.Error()), nil
	}

	var results []map[string]interface{}
	count := 0

	for rows.Next() && count < maxRows {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err = rows.Scan(valuePtrs...); err != nil {
			return mcp.NewToolResultError(ErrReadingRow.Error()), nil
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			row[col] = formatValue(values[i])
		}
		results = append(results, row)
		count++
	}

	if err = rows.Err(); err != nil {
		log.Printf("Error during iteration: %v\n", err)
		return mcp.NewToolResultError(ErrReadingResults.Error()), nil
	}

	response := map[string]interface{}{
		"rows":      results,
		"row_count": len(results),
		"columns":   columns,
		"truncated": count >= maxRows,
		"max_rows":  maxRows,
	}

	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(ErrSerializingJSON.Error()), nil
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

// formatValue converts database values to JSON-safe formats
func formatValue(val interface{}) interface{} {
	switch v := val.(type) {
	case []byte:
		if len(v) > 1000 {
			return fmt.Sprintf("<binary data: %d bytes>", len(v))
		}
		if utf8.Valid(v) {
			return string(v)
		}
		return fmt.Sprintf("<binary data: %d bytes>", len(v))
	case time.Time:
		return v.Format("2006-01-02 15:04:05")
	case nil:
		return nil
	default:
		return v
	}
}

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

func (s *DatabaseMCP) toolExecuteQuery() (mcp.Tool, server.ToolHandlerFunc) {
	return mcp.Tool{
		Name:        "execute_query",
		Description: "Executes a SELECT query in SQL Server and returns the results.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "SQL query to be executed (SELECT only)",
				},
				"max_rows": map[string]interface{}{
					"type":        "number",
					"description": "Maximum number of rows to be returned (default: 100)",
				},
			},
			Required: []string{"query"},
		},
	}, s.handleExecuteQuery
}

func (s *DatabaseMCP) handleExecuteQuery(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return mcp.NewToolResultError("Invalid arguments"), nil
	}

	query, ok := getStringArg(args, "query")
	if !ok {
		return mcp.NewToolResultError("Query not provided."), nil
	}

	// Complete validation
	validator := NewSQLValidator(query)
	if err := validator.Validate(); err != nil {
		// Log of the suspicious attempt
		log.Printf("Query blocked: %s\nReason: %v\n", query, err)
		return mcp.NewToolResultError(fmt.Sprintf("Query not allowed: %v", err)), nil
	}

	maxRows := getIntArg(args, "max_rows", 100)
	if maxRows <= 0 || maxRows > 10000 {
		maxRows = 100
	}

	// Shorter timeout to prevent slow queries
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Run the query in read-only mode if possible
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		// Do not disclose error details to the user (security)
		log.Printf("Error in query: %v\nQuery: %s\n", err, query)
		return mcp.NewToolResultError("Error executing query. Check the syntax."), nil
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return mcp.NewToolResultError("Error retrieving columns"), nil
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
			return mcp.NewToolResultError("Error reading line"), nil
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]

			// Safe treatment of types
			switch v := val.(type) {
			case []byte:
				// Limit the size of returned binary data.
				if len(v) > 1000 {
					row[col] = fmt.Sprintf("<binary data: %d bytes>", len(v))
				} else if utf8.Valid(v) {
					row[col] = string(v)
				} else {
					row[col] = fmt.Sprintf("<binary data: %d bytes>", len(v))
				}
			case time.Time:
				row[col] = v.Format("2006-01-02 15:04:05")
			case nil:
				row[col] = nil
			default:
				row[col] = v
			}
		}
		results = append(results, row)
		count++
	}

	if err = rows.Err(); err != nil {
		log.Printf("Error during iteration: %v\n", err)
		return mcp.NewToolResultError("Error during results reading"), nil
	}

	jsonData, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return mcp.NewToolResultError("Error serializing results"), nil
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

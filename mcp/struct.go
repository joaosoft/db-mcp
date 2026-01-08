package mcp

import (
	"database/sql"

	"github.com/mark3labs/mcp-go/server"
)

type DatabaseMCP struct {
	server       *server.MCPServer
	db           *sql.DB
	queryBuilder *QueryBuilder
}

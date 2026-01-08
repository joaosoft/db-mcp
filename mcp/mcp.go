package mcp

import (
	"github.com/mark3labs/mcp-go/server"
)

func NewMcpServer() (*DatabaseMCP, error) {
	db, driver, err := newDbConnection()
	if err != nil {
		return nil, err
	}

	dbMCPServer := &DatabaseMCP{
		server: server.NewMCPServer(
			"Database MCP",
			"1.0.0",
			server.WithToolCapabilities(true),
		),
		db:           db,
		queryBuilder: NewQueryBuilder(driver),
	}

	// Register tools
	dbMCPServer.registerTools()

	return dbMCPServer, nil
}

func (s *DatabaseMCP) Start() error {
	return server.ServeStdio(s.server)
}

func (s *DatabaseMCP) Close() error {
	return s.db.Close()
}

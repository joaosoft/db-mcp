package mcp

import (
	"github.com/mark3labs/mcp-go/server"
)

// NewMcpServer creates a new MCP server instance.
// If DB_CONNECTION_STRING is not set, the server starts without a database connection.
// Use the configure_datasource tool to connect to a database dynamically.
func NewMcpServer() (*DbMCPServer, error) {
	db, driver, err := newDbConnection()
	if err != nil {
		return nil, err
	}

	var queryBuilder *QueryBuilder
	if driver != "" {
		queryBuilder = NewQueryBuilder(driver)
	}

	dbMCPServer := &DbMCPServer{
		server: server.NewMCPServer(
			"Database MCP",
			"1.0.0",
			server.WithToolCapabilities(true),
		),
		db:           db,
		queryBuilder: queryBuilder,
	}

	// Register tools
	dbMCPServer.registerTools()

	return dbMCPServer, nil
}

// Start starts the MCP server in stdio mode
func (s *DbMCPServer) Start() error {
	return server.ServeStdio(s.server)
}

// Close closes the database connection if it exists
func (s *DbMCPServer) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// IsConnected returns true if a database connection is established
func (s *DbMCPServer) IsConnected() bool {
	return s.db != nil
}

package main

import (
	"db-mcp/mcp"
	"log"

	_ "github.com/denisenkom/go-mssqldb"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/godror/godror"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// Define MCP Server
	mcpServer, err := mcp.NewMcpServer()
	if err != nil {
		log.Fatalf("Error setting up MCP server: %v", err)
		return
	}

	// Start server in stdio
	defer mcpServer.Close()
	if err = mcpServer.Start(); err != nil {
		log.Fatalf("Error starting server: %v", err)
		return
	}
}

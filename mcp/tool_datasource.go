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

// Tool: Configure DataSource
func (s *DbMCPServer) toolConfigureDataSource() (mcp.Tool, server.ToolHandlerFunc) {
	return mcp.Tool{
		Name:        "configure_datasource",
		Description: "Configure and connect to a database. Supports multiple database drivers: sqlserver, postgres, mysql, sqlite, oracle. The connection will be used for all subsequent database operations.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"driver": map[string]interface{}{
					"type":        "string",
					"description": "Database driver: 'sqlserver', 'postgres', 'mysql', 'sqlite', 'oracle'",
					"enum":        []string{"sqlserver", "postgres", "mysql", "sqlite", "oracle"},
				},
				"connection_string": map[string]interface{}{
					"type":        "string",
					"description": "Database connection string. Examples:\n- SQL Server: sqlserver://user:password@host:1433?database=dbname\n- PostgreSQL: postgres://user:password@host:5432/dbname?sslmode=disable\n- MySQL: user:password@tcp(host:3306)/dbname\n- SQLite: /path/to/database.db\n- Oracle: oracle://user:password@host:1521/service",
				},
				"name": map[string]interface{}{
					"type":        "string",
					"description": "Optional friendly name for this connection (for identification)",
				},
			},
			Required: []string{"driver", "connection_string"},
		},
	}, s.handleConfigureDataSource
}

func (s *DbMCPServer) handleConfigureDataSource(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := getArgs(request.Params.Arguments)
	if !ok {
		return mcp.NewToolResultError(ErrInvalidArguments.Error()), nil
	}

	driver, ok := getStringArg(args, "driver")
	if !ok || driver == "" {
		return mcp.NewToolResultError(ErrDriverRequired.Error()), nil
	}

	connString, ok := getStringArg(args, "connection_string")
	if !ok || connString == "" {
		return mcp.NewToolResultError(ErrConnectionStringRequired.Error()), nil
	}

	name := "default"
	if n, ok := getStringArg(args, "name"); ok && n != "" {
		name = n
	}

	// Normalize driver names
	normalizedDriver := normalizeDriver(driver)
	if normalizedDriver == "" {
		return mcp.NewToolResultError(fmt.Errorf("%w: '%s'. Supported drivers: sqlserver, postgres, mysql, sqlite, oracle", ErrInvalidDriver, driver).Error()), nil
	}

	// Try to connect
	newDB, err := sql.Open(normalizedDriver, connString)
	if err != nil {
		return mcp.NewToolResultError(fmt.Errorf("%w: %v", ErrConnectionFailed, err).Error()), nil
	}

	// Configure connection pool
	newDB.SetMaxOpenConns(DBMaxOpenConns)
	newDB.SetMaxIdleConns(DBMaxIdleConns)
	newDB.SetConnMaxLifetime(DBConnMaxLifetime)

	// Test connection
	pingCtx, cancel := context.WithTimeout(ctx, DBPingTimeout)
	defer cancel()

	if err = newDB.PingContext(pingCtx); err != nil {
		newDB.Close()
		return mcp.NewToolResultError(fmt.Errorf("%w: %v", ErrConnectionTestFailed, err).Error()), nil
	}

	// Close old connection
	if s.db != nil {
		s.db.Close()
	}

	// Update server with new connection
	s.db = newDB
	s.queryBuilder = NewQueryBuilder(normalizedDriver)

	// Generate connection ID
	connID := fmt.Sprintf("%s_%d", name, time.Now().UnixNano())

	// Store connection info
	connManager.mu.Lock()
	for _, conn := range connManager.connections {
		conn.IsActive = false
	}
	connManager.connections[connID] = &ConnectionInfo{
		ID:               connID,
		Driver:           driver,
		ConnectionString: connString,
		Name:             name,
		ConnectedAt:      time.Now(),
		IsActive:         true,
	}
	connManager.activeConnID = connID
	connManager.mu.Unlock()

	// Get database info for response
	var dbInfo string
	infoQuery := s.queryBuilder.GetDatabaseInfoQuery()
	if err := s.db.QueryRowContext(ctx, infoQuery).Scan(&dbInfo); err != nil {
		dbInfo = "Connected successfully"
	}

	response := map[string]interface{}{
		"status":        "connected",
		"driver":        driver,
		"name":          name,
		"connection_id": connID,
		"database_info": dbInfo,
		"message":       fmt.Sprintf("Successfully connected to %s database", driver),
	}

	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(ErrSerializingJSON.Error()), nil
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

// Tool: Get Current DataSource
func (s *DbMCPServer) toolGetCurrentDataSource() (mcp.Tool, server.ToolHandlerFunc) {
	return mcp.Tool{
		Name:        "get_current_datasource",
		Description: "Get information about the currently active database connection",
		InputSchema: mcp.ToolInputSchema{
			Type:       "object",
			Properties: map[string]interface{}{},
		},
	}, s.handleGetCurrentDataSource
}

func (s *DbMCPServer) handleGetCurrentDataSource(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	connManager.mu.RLock()
	defer connManager.mu.RUnlock()

	if connManager.activeConnID == "" {
		// Check if there's a connection from environment variables
		if s.db != nil {
			pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
			defer cancel()

			status := "connected"
			if err := s.db.PingContext(pingCtx); err != nil {
				status = "disconnected"
			}

			response := map[string]interface{}{
				"status":  status,
				"source":  "environment",
				"message": "Connection was configured via environment variables (DB_DRIVER, DB_CONNECTION_STRING)",
			}

			jsonData, _ := json.MarshalIndent(response, "", "  ")
			return mcp.NewToolResultText(string(jsonData)), nil
		}

		return mcp.NewToolResultError(ErrNoConnection.Error()), nil
	}

	connInfo, exists := connManager.connections[connManager.activeConnID]
	if !exists {
		return mcp.NewToolResultError(ErrNoConnection.Error()), nil
	}

	// Check if connection is alive
	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	status := "connected"
	if err := s.db.PingContext(pingCtx); err != nil {
		status = "disconnected"
	}

	response := map[string]interface{}{
		"status":        status,
		"connection_id": connInfo.ID,
		"driver":        connInfo.Driver,
		"name":          connInfo.Name,
		"connected_at":  connInfo.ConnectedAt.Format("2006-01-02 15:04:05"),
		"source":        "configure_datasource",
	}

	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(ErrSerializingJSON.Error()), nil
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

// Tool: Test Connection
func (s *DbMCPServer) toolTestConnection() (mcp.Tool, server.ToolHandlerFunc) {
	return mcp.Tool{
		Name:        "test_connection",
		Description: "Test a database connection without switching to it. Useful to validate connection strings before configuring.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"driver": map[string]interface{}{
					"type":        "string",
					"description": "Database driver: 'sqlserver', 'postgres', 'mysql', 'sqlite', 'oracle'",
					"enum":        []string{"sqlserver", "postgres", "mysql", "sqlite", "oracle"},
				},
				"connection_string": map[string]interface{}{
					"type":        "string",
					"description": "Database connection string to test",
				},
			},
			Required: []string{"driver", "connection_string"},
		},
	}, s.handleTestConnection
}

func (s *DbMCPServer) handleTestConnection(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := getArgs(request.Params.Arguments)
	if !ok {
		return mcp.NewToolResultError(ErrInvalidArguments.Error()), nil
	}

	driver, ok := getStringArg(args, "driver")
	if !ok || driver == "" {
		return mcp.NewToolResultError(ErrDriverRequired.Error()), nil
	}

	connString, ok := getStringArg(args, "connection_string")
	if !ok || connString == "" {
		return mcp.NewToolResultError(ErrConnectionStringRequired.Error()), nil
	}

	normalizedDriver := normalizeDriver(driver)
	if normalizedDriver == "" {
		return mcp.NewToolResultError(fmt.Errorf("%w: '%s'", ErrInvalidDriver, driver).Error()), nil
	}

	// Try to connect
	testDB, err := sql.Open(normalizedDriver, connString)
	if err != nil {
		response := map[string]interface{}{
			"status":  "failed",
			"driver":  driver,
			"error":   fmt.Errorf("%w: %v", ErrConnectionFailed, err).Error(),
			"message": "Connection string may be invalid",
		}
		jsonData, _ := json.MarshalIndent(response, "", "  ")
		return mcp.NewToolResultText(string(jsonData)), nil
	}
	defer testDB.Close()

	// Test connection
	pingCtx, cancel := context.WithTimeout(ctx, DBPingTimeout)
	defer cancel()

	if err = testDB.PingContext(pingCtx); err != nil {
		response := map[string]interface{}{
			"status":  "failed",
			"driver":  driver,
			"error":   fmt.Errorf("%w: %v", ErrConnectionTestFailed, err).Error(),
			"message": "Could not reach the database server",
		}
		jsonData, _ := json.MarshalIndent(response, "", "  ")
		return mcp.NewToolResultText(string(jsonData)), nil
	}

	// Get database version
	qb := NewQueryBuilder(normalizedDriver)
	var version string
	if err := testDB.QueryRowContext(ctx, qb.GetDatabaseInfoQuery()).Scan(&version); err != nil {
		version = "Unknown"
	}

	response := map[string]interface{}{
		"status":          "success",
		"driver":          driver,
		"database_info":   version,
		"message":         "Connection test successful! You can now use configure_datasource to switch to this database.",
		"example_command": fmt.Sprintf(`configure_datasource(driver="%s", connection_string="<your_connection_string>", name="my_database")`, driver),
	}

	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(ErrSerializingJSON.Error()), nil
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

// Tool: Disconnect
func (s *DbMCPServer) toolDisconnect() (mcp.Tool, server.ToolHandlerFunc) {
	return mcp.Tool{
		Name:        "disconnect_datasource",
		Description: "Disconnect from the current database",
		InputSchema: mcp.ToolInputSchema{
			Type:       "object",
			Properties: map[string]interface{}{},
		},
	}, s.handleDisconnect
}

func (s *DbMCPServer) handleDisconnect(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if s.db == nil {
		return mcp.NewToolResultError(ErrNoConnection.Error()), nil
	}

	// Close the connection
	err := s.db.Close()
	s.db = nil
	s.queryBuilder = nil

	connManager.mu.Lock()
	if connManager.activeConnID != "" {
		if conn, exists := connManager.connections[connManager.activeConnID]; exists {
			conn.IsActive = false
		}
	}
	connManager.activeConnID = ""
	connManager.mu.Unlock()

	if err != nil {
		response := map[string]interface{}{
			"status":  "disconnected",
			"warning": fmt.Sprintf("Disconnected with warning: %v", err),
		}
		jsonData, _ := json.MarshalIndent(response, "", "  ")
		return mcp.NewToolResultText(string(jsonData)), nil
	}

	response := map[string]interface{}{
		"status":  "disconnected",
		"message": "Successfully disconnected from database",
	}

	jsonData, _ := json.MarshalIndent(response, "", "  ")
	return mcp.NewToolResultText(string(jsonData)), nil
}

// Tool: List Supported Drivers
func (s *DbMCPServer) toolListDrivers() (mcp.Tool, server.ToolHandlerFunc) {
	return mcp.Tool{
		Name:        "list_database_drivers",
		Description: "List all supported database drivers and their connection string formats",
		InputSchema: mcp.ToolInputSchema{
			Type:       "object",
			Properties: map[string]interface{}{},
		},
	}, s.handleListDrivers
}

func (s *DbMCPServer) handleListDrivers(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	drivers := []map[string]interface{}{
		{
			"driver":                   "sqlserver",
			"name":                     "Microsoft SQL Server",
			"connection_string_format": "sqlserver://username:password@host:1433?database=dbname",
			"example":                  "sqlserver://sa:MyPassword123@localhost:1433?database=AdventureWorks",
			"notes":                    "Port 1433 is default. Use 'encrypt=disable' for local development.",
		},
		{
			"driver":                   "postgres",
			"name":                     "PostgreSQL",
			"connection_string_format": "postgres://username:password@host:5432/dbname?sslmode=disable",
			"example":                  "postgres://postgres:password@localhost:5432/mydb?sslmode=disable",
			"notes":                    "Port 5432 is default. sslmode can be: disable, require, verify-ca, verify-full.",
		},
		{
			"driver":                   "mysql",
			"name":                     "MySQL / MariaDB",
			"connection_string_format": "username:password@tcp(host:3306)/dbname",
			"example":                  "root:password@tcp(localhost:3306)/mydb",
			"notes":                    "Port 3306 is default. Add ?parseTime=true for time.Time support.",
		},
		{
			"driver":                   "sqlite",
			"name":                     "SQLite",
			"connection_string_format": "/path/to/database.db",
			"example":                  "/tmp/mydata.db or :memory: for in-memory database",
			"notes":                    "File-based database. Use ':memory:' for in-memory database.",
		},
		{
			"driver":                   "oracle",
			"name":                     "Oracle Database",
			"connection_string_format": "oracle://username:password@host:1521/service_name",
			"example":                  "oracle://system:oracle@localhost:1521/XEPDB1",
			"notes":                    "Requires Oracle Instant Client. Port 1521 is default.",
		},
	}

	response := map[string]interface{}{
		"supported_drivers": drivers,
		"usage": map[string]string{
			"test_connection":      "Use test_connection to validate a connection string before switching",
			"configure_datasource": "Use configure_datasource to connect and switch to a new database",
			"get_current":          "Use get_current_datasource to see the active connection",
		},
	}

	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(ErrSerializingJSON.Error()), nil
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

// normalizeDriver converts user-friendly driver names to internal driver names
func normalizeDriver(driver string) string {
	switch driver {
	case "sqlserver":
		return "sqlserver"
	case "postgres":
		return "postgres"
	case "mysql":
		return "mysql"
	case "sqlite", "sqlite3":
		return "sqlite3"
	case "oracle", "godror":
		return "godror"
	default:
		return ""
	}
}

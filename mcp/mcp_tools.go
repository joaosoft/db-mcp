package mcp

func (s *DatabaseMCP) registerTools() {
	// ===== DataSource Management =====
	// Configure DataSource (connect to a database)
	s.server.AddTool(s.toolConfigureDataSource())

	// Get Current DataSource
	s.server.AddTool(s.toolGetCurrentDataSource())

	// Test Connection
	s.server.AddTool(s.toolTestConnection())

	// Disconnect
	s.server.AddTool(s.toolDisconnect())

	// List Supported Drivers
	s.server.AddTool(s.toolListDrivers())

	// ===== Query Execution =====
	// Execute Query
	s.server.AddTool(s.toolExecuteQuery())

	// ===== Tables =====
	// List Tables
	s.server.AddTool(s.toolListTables())

	// Describe Tables
	s.server.AddTool(s.toolDescribeTable())

	// List Table Rows
	s.server.AddTool(s.toolListTableRows())

	// Get Full Table Schema
	s.server.AddTool(s.toolGetTableSchemaFull())

	// ===== Stored Procedures =====
	// List Stored Procedures
	s.server.AddTool(s.toolListProcedures())

	// Get Procedure Source Code
	s.server.AddTool(s.toolGetProcedureCode())

	// Execute Procedure
	s.server.AddTool(s.toolExecuteProcedure())

	// ===== Functions =====
	// List Functions
	s.server.AddTool(s.toolListFunctions())

	// Get Function Source Code
	s.server.AddTool(s.toolGetFunctionCode())

	// ===== Views =====
	// List Views
	s.server.AddTool(s.toolListViews())

	// Get View Definition
	s.server.AddTool(s.toolGetViewDefinition())

	// ===== Triggers =====
	// List Triggers
	s.server.AddTool(s.toolListTriggers())

	// Get Trigger Source Code
	s.server.AddTool(s.toolGetTriggerCode())

	// ===== Database Info =====
	// Search Object
	s.server.AddTool(s.toolSearchObjects())

	// Get Database Information
	s.server.AddTool(s.toolGetDatabaseInfo())
}

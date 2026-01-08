package mcp

func (s *DatabaseMCP) registerTools() {
	// Execute Query
	s.server.AddTool(s.toolExecuteQuery())

	// List Tables
	s.server.AddTool(s.toolListTables())

	// Describe Tables
	s.server.AddTool(s.toolDescribeTable())

	// List Table Rows
	s.server.AddTool(s.toolListTableRows())

	// Get Full Table Schema
	s.server.AddTool(s.toolGetTableSchemaFull())

	// List Stored Procedures
	s.server.AddTool(s.toolListProcedures())

	// Get Procedure Source Code
	s.server.AddTool(s.toolGetProcedureCode())

	// List Functions
	s.server.AddTool(s.toolListFunctions())

	// Get Function Source Code
	s.server.AddTool(s.toolGetFunctionCode())

	// Execute Procedure
	s.server.AddTool(s.toolExecuteProcedure())

	// List Views
	s.server.AddTool(s.toolListViews())

	// Get View Definition
	s.server.AddTool(s.toolGetViewDefinition())

	// List Triggers
	s.server.AddTool(s.toolListTriggers())

	// Get Trigger Source Code
	s.server.AddTool(s.toolGetTriggerCode())

	// Search Object
	s.server.AddTool(s.toolSearchObjects())

	// Get Database Information
	s.server.AddTool(s.toolGetDatabaseInfo())
}

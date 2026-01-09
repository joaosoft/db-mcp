# Code Style and Conventions

## Go Standards
- Follow standard Go conventions
- Use `go fmt` for formatting
- Use `go vet` for static analysis

## Naming Conventions
- Files: lowercase with underscores (e.g., `query_builder.go`, `sql_table.go`)
- Package: `mcp`
- Functions/Methods: CamelCase (exported) or camelCase (unexported)

## File Organization
- Group SQL operations by type (table, procedure, function, view, trigger)
- Each SQL operation type has its own file prefixed with `sql_`
- Core MCP functionality in `mcp.go` and `mcp_tools.go`

## Error Handling
- Return errors to be handled by the MCP framework
- Use descriptive error messages

## Database Abstraction
- Use QueryBuilder for all database-specific SQL syntax
- Support all 5 database types in any new SQL operations

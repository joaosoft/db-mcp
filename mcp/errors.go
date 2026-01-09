package mcp

import "errors"

// Connection errors
var (
	ErrNoConnection             = errors.New("no database connection. Use configure_datasource to connect first")
	ErrConnectionFailed         = errors.New("failed to connect to database")
	ErrConnectionTestFailed     = errors.New("connection test failed")
	ErrDriverRequired           = errors.New("driver is required")
	ErrConnectionStringRequired = errors.New("connection_string is required")
	ErrConnecting               = errors.New("error connecting to database")
	ErrTestingConnection        = errors.New("error testing connection")
)

// Argument errors
var (
	ErrInvalidArguments  = errors.New("invalid arguments")
	ErrInvalidIdentifier = errors.New("invalid identifier")
	ErrMissingRequired   = errors.New("missing required parameter")
	ErrSearchTermRequired = errors.New("search_term is required")
)

// Query errors
var (
	ErrQueryNotAllowed    = errors.New("query not allowed")
	ErrQueryEmpty         = errors.New("empty query")
	ErrQueryTooLong       = errors.New("query too long")
	ErrQuerySyntax        = errors.New("error executing query - check the syntax")
	ErrMultipleStatements = errors.New("multiple statements not allowed")
	ErrQueryRequired      = errors.New("query is required")
	ErrReadingRow         = errors.New("error reading row")
	ErrReadingResults     = errors.New("error reading results")
)

// Query validation errors
var (
	ErrOnlySelectAllowed           = errors.New("only SELECT or WITH queries are allowed")
	ErrCommandNotAllowed           = errors.New("command not allowed")
	ErrTransactionNotAllowed       = errors.New("transaction commands are not allowed")
	ErrAdminCommandNotAllowed      = errors.New("administrative command not allowed")
	ErrSecurityCommandNotAllowed   = errors.New("security command not allowed")
	ErrDangerousFunctionNotAllowed = errors.New("dangerous function not permitted")
	ErrMultipleCommandsNotAllowed  = errors.New("multiple commands are not allowed")
	ErrTooManySubqueries           = errors.New("too many subqueries")
	ErrSelectIntoNotAllowed        = errors.New("SELECT INTO is not allowed")
	ErrTooManyUnions               = errors.New("too many UNION clauses")
	ErrSuspiciousCharacter         = errors.New("suspicious control character detected")
	ErrExcessiveHexEncoding        = errors.New("excessive use of hexadecimal encoding")
	ErrExcessiveCharFunction       = errors.New("excessive use of CHAR/NCHAR (possible obfuscation)")
	ErrTimeFunctionNotAllowed      = errors.New("time function not allowed")
	ErrUnbalancedParentheses       = errors.New("unbalanced parentheses")
	ErrParenthesesTooDeep          = errors.New("parenthesis depth too large")
)

// Object errors
var (
	ErrTableNotFound     = errors.New("table not found")
	ErrViewNotFound      = errors.New("view not found")
	ErrProcedureNotFound = errors.New("procedure not found")
	ErrFunctionNotFound  = errors.New("function not found")
	ErrTriggerNotFound   = errors.New("trigger not found")
	ErrObjectNotFound    = errors.New("object not found")
)

// Feature support errors
var (
	ErrStoredProceduresNotSupported = errors.New("stored procedures are not supported by this database")
	ErrFunctionsNotSupported        = errors.New("functions are not supported by this database")
	ErrFeatureNotSupported          = errors.New("feature not supported by this database")
)

// Validation errors
var (
	ErrInvalidDriver        = errors.New("invalid database driver")
	ErrInvalidTableName     = errors.New("invalid table name")
	ErrInvalidViewName      = errors.New("invalid view name")
	ErrInvalidProcedureName = errors.New("invalid procedure name")
	ErrInvalidFunctionName  = errors.New("invalid function name")
	ErrInvalidTriggerName   = errors.New("invalid trigger name")
	ErrInvalidSchemaName    = errors.New("invalid schema name")
	ErrInvalidColumnName    = errors.New("invalid column name")
	ErrInvalidOperator      = errors.New("invalid operator")
	ErrInvalidFunctionType  = errors.New("invalid function type - use: scalar, table, or all")
)

// Data errors
var (
	ErrCodeNotAvailable       = errors.New("source code not available")
	ErrDefinitionNotAvailable = errors.New("definition not available")
	ErrNoColumnsFound         = errors.New("no columns found in the table")
	ErrColumnNotExists        = errors.New("column does not exist")
)

// Serialization errors
var (
	ErrSerializingJSON = errors.New("error serializing JSON")
)

// Operation errors
var (
	ErrListingTables      = errors.New("error listing tables")
	ErrListingViews       = errors.New("error listing views")
	ErrListingProcedures  = errors.New("error listing procedures")
	ErrListingFunctions   = errors.New("error listing functions")
	ErrListingTriggers    = errors.New("error listing triggers")
	ErrDescribingTable    = errors.New("error describing table")
	ErrCheckingTable      = errors.New("error checking table")
	ErrRetrievingColumns  = errors.New("error retrieving columns")
	ErrCountingRows       = errors.New("error counting rows")
	ErrFetchingRows       = errors.New("error fetching rows")
	ErrSearchingObjects   = errors.New("error searching objects")
	ErrFetchingCode       = errors.New("error fetching code")
	ErrExecutingProcedure = errors.New("error executing procedure")
	ErrRetrievingView     = errors.New("error retrieving view definition")
	ErrRetrievingTrigger  = errors.New("error retrieving trigger code")
)

// Filter errors
var (
	ErrContainsRequiresString   = errors.New("'contains' operator requires a string value")
	ErrStartsWithRequiresString = errors.New("'starts_with' operator requires a string value")
	ErrEndsWithRequiresString   = errors.New("'ends_with' operator requires a string value")
)

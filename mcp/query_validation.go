package mcp

import (
	"fmt"
	"regexp"
	"strings"
)

func NewSQLValidator(query string) *SQLValidator {
	return &SQLValidator{
		query:      query,
		normalized: normalizeSQL(query),
	}
}

// Normalizes SQL by removing extra spaces and comments while maintaining structure.
func normalizeSQL(sql string) string {
	// Remove line comments (-- )
	sql = reLineComments.ReplaceAllString(sql, " ")

	// Remove block comments (/* */)
	sql = reBlockComments.ReplaceAllString(sql, " ")

	// Normalize multiple spaces
	sql = reMultipleSpaces.ReplaceAllString(sql, " ")

	// Remove spaces before/after parentheses and commas
	sql = reParensAndCommas.ReplaceAllString(sql, "$1")

	return strings.TrimSpace(strings.ToUpper(sql))
}

// Remove literal strings for command parsing
func removeStringLiterals(sql string) string {
	// Remove strings enclosed in single quotes
	sql = reSingleQuotes.ReplaceAllString(sql, "''")

	// Remove strings enclosed in double quotes
	sql = reDoubleQuotes.ReplaceAllString(sql, `""`)

	// Remove strings enclosed in square brackets (SQL Server identifiers)
	sql = reSquareBrackets.ReplaceAllString(sql, "[]")

	return sql
}

// Verifies if the consultation is secure.
func (v *SQLValidator) Validate() error {
	// 1. Check if it's not empty
	if strings.TrimSpace(v.query) == "" {
		return ErrQueryEmpty
	}

	// 2. Check maximum size (prevent DoS)
	if len(v.query) > MaxQueryLength {
		return fmt.Errorf("%w (maximum %d characters)", ErrQueryTooLong, MaxQueryLength)
	}

	// 3. Check if it starts with SELECT or WITH
	if !strings.HasPrefix(v.normalized, "SELECT") && !strings.HasPrefix(v.normalized, "WITH") {
		return ErrOnlySelectAllowed
	}

	// 4. Removing literals for command parsing
	sqlWithoutLiterals := removeStringLiterals(v.normalized)

	// 5. Dangerous DML commands (outside of strings)
	dangerousDML := []string{
		"INSERT", "UPDATE", "DELETE", "TRUNCATE", "MERGE",
	}
	for _, cmd := range dangerousDML {
		if containsKeyword(sqlWithoutLiterals, cmd) {
			return fmt.Errorf("%w: %s", ErrCommandNotAllowed, cmd)
		}
	}

	// 6. Dangerous DDL commands
	dangerousDDL := []string{
		"DROP", "CREATE", "ALTER", "RENAME",
	}
	for _, cmd := range dangerousDDL {
		if containsKeyword(sqlWithoutLiterals, cmd) {
			return fmt.Errorf("%w: %s", ErrCommandNotAllowed, cmd)
		}
	}

	// 7. Execution commands
	dangerousExec := []string{
		"EXEC", "EXECUTE", "SP_EXECUTESQL", "XP_CMDSHELL",
	}
	for _, cmd := range dangerousExec {
		if containsKeyword(sqlWithoutLiterals, cmd) {
			return fmt.Errorf("%w: %s", ErrCommandNotAllowed, cmd)
		}
	}

	// 8. Transaction control commands
	transactionCmds := []string{
		"BEGIN TRANSACTION", "BEGIN TRAN", "COMMIT", "ROLLBACK", "SAVE TRANSACTION",
	}
	for _, cmd := range transactionCmds {
		if strings.Contains(sqlWithoutLiterals, cmd) {
			return fmt.Errorf("%w: %s", ErrTransactionNotAllowed, cmd)
		}
	}

	// 9. Backup/restore commands
	backupCmds := []string{
		"BACKUP", "RESTORE", "DUMP",
	}
	for _, cmd := range backupCmds {
		if containsKeyword(sqlWithoutLiterals, cmd) {
			return fmt.Errorf("%w: %s", ErrCommandNotAllowed, cmd)
		}
	}

	// 10. Administration commands
	adminCmds := []string{
		"SHUTDOWN", "RECONFIGURE", "DBCC", "KILL",
	}
	for _, cmd := range adminCmds {
		if containsKeyword(sqlWithoutLiterals, cmd) {
			return fmt.Errorf("%w: %s", ErrAdminCommandNotAllowed, cmd)
		}
	}

	// 11. Security commands
	securityCmds := []string{
		"GRANT", "REVOKE", "DENY",
	}
	for _, cmd := range securityCmds {
		if containsKeyword(sqlWithoutLiterals, cmd) {
			return fmt.Errorf("%w: %s", ErrSecurityCommandNotAllowed, cmd)
		}
	}

	// 12. Dangerous functions of the system
	dangerousFunctions := []string{
		"XP_", "SP_CONFIGURE", "SP_ADDSRVROLEMEMBER", "SP_ADDLOGIN",
		"OPENROWSET", "OPENDATASOURCE", "OPENQUERY",
		"BULK INSERT", "BCP",
	}
	for _, fn := range dangerousFunctions {
		if strings.Contains(sqlWithoutLiterals, fn) {
			return fmt.Errorf("%w: %s", ErrDangerousFunctionNotAllowed, fn)
		}
	}

	// 13. Detect multiple statements (separated by semicolon)
	if err := v.validateMultipleStatements(); err != nil {
		return err
	}

	// 14. Check INTO clause (SELECT INTO)
	if err := v.validateNoIntoClause(sqlWithoutLiterals); err != nil {
		return err
	}

	// 15. Check for attempts at stacked queries.
	if strings.Count(sqlWithoutLiterals, ";") > 0 {
		return ErrMultipleCommandsNotAllowed
	}

	// 16. Check use of UNION for bypass
	if err := v.validateUnionUsage(sqlWithoutLiterals); err != nil {
		return err
	}

	// 17.Check encoding and suspicious special characters
	if err := v.validateEncoding(); err != nil {
		return err
	}

	// 18. Check for time-based blind SQL injection attempts
	if err := v.validateNoTimingAttacks(sqlWithoutLiterals); err != nil {
		return err
	}

	// 19. Check number of subqueries (prevent DoS)
	if strings.Count(sqlWithoutLiterals, "SELECT") > MaxSubqueryCount {
		return fmt.Errorf("%w (maximum %d)", ErrTooManySubqueries, MaxSubqueryCount)
	}

	// 20. Check parenthesis depth (prevent DoS)
	if err := v.validateParenthesesDepth(); err != nil {
		return err
	}

	return nil
}

// keywordPatterns caches compiled regex patterns for keyword matching
var keywordPatterns = make(map[string]*regexp.Regexp)

// Checks if a keyword exists as a complete word (not part of another word)
func containsKeyword(sql string, keyword string) bool {
	pattern, exists := keywordPatterns[keyword]
	if !exists {
		pattern = regexp.MustCompile(`\b` + keyword + `\b`)
		keywordPatterns[keyword] = pattern
	}
	return pattern.MatchString(sql)
}

// Validates multiple statements
func (v *SQLValidator) validateMultipleStatements() error {
	// Search for semicolons outside of strings
	inString := false
	escapeNext := false

	for i, char := range v.query {
		if escapeNext {
			escapeNext = false
			continue
		}

		if char == '\\' {
			escapeNext = true
			continue
		}

		if char == '\'' {
			inString = !inString
			continue
		}

		if !inString && char == ';' {
			// Check that it is not the last character (allowed at the end)
			if i < len(v.query)-1 && strings.TrimSpace(v.query[i+1:]) != "" {
				return ErrMultipleCommandsNotAllowed
			}
		}
	}

	return nil
}

// Validates that there is no SELECT INTO statement.
func (v *SQLValidator) validateNoIntoClause(sql string) error {
	// Search for pattern SELECT ... INTO
	if reSelectInto.MatchString(sql) {
		return ErrSelectIntoNotAllowed
	}
	return nil
}

// validateUnionUsage validates UNION clause usage (allows only legitimate queries)
func (v *SQLValidator) validateUnionUsage(sql string) error {
	// Count UNIONs
	unionCount := strings.Count(sql, "UNION")
	if unionCount > MaxUnionCount {
		return fmt.Errorf("%w (maximum %d)", ErrTooManyUnions, MaxUnionCount)
	}

	return nil
}

// Validates encoding and special characters
func (v *SQLValidator) validateEncoding() error {
	// Checking for suspicious control characters
	for _, char := range v.query {
		if char < 32 && char != '\n' && char != '\r' && char != '\t' {
			return ErrSuspiciousCharacter
		}
	}

	// Check for hexadecimal encoding attempts (0x...)
	if strings.Contains(v.normalized, "0X") {
		// Allow only in safe contexts (simple comparisons)
		matches := reHexPattern.FindAllString(v.normalized, -1)
		if len(matches) > MaxHexEncodingCount {
			return ErrExcessiveHexEncoding
		}
	}

	// Check CHAR / NCHAR used to obfuscate commands
	matches := reCharNCharPattern.FindAllString(v.normalized, -1)
	if len(matches) > MaxCharFunctionCount {
		return ErrExcessiveCharFunction
	}

	return nil
}

// Validates timing attack attempts.
func (v *SQLValidator) validateNoTimingAttacks(sql string) error {
	timingFunctions := []string{
		"WAITFOR", "DELAY", "SLEEP", "BENCHMARK",
	}

	for _, fn := range timingFunctions {
		if containsKeyword(sql, fn) {
			return fmt.Errorf("%w: %s", ErrTimeFunctionNotAllowed, fn)
		}
	}

	return nil
}

// Validate parenthesis depth (prevent DoS)
func (v *SQLValidator) validateParenthesesDepth() error {
	depth := 0
	maxDepth := 0

	for _, char := range v.query {
		if char == '(' {
			depth++
			if depth > maxDepth {
				maxDepth = depth
			}
		} else if char == ')' {
			depth--
		}
	}

	if depth != 0 {
		return ErrUnbalancedParentheses
	}

	if maxDepth > MaxParenthesesDepth {
		return fmt.Errorf("%w (maximum %d)", ErrParenthesesTooDeep, MaxParenthesesDepth)
	}

	return nil
}

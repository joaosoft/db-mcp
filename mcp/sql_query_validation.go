package mcp

import (
	"fmt"
	"regexp"
	"strings"
)

// Structure for SQL analysis
type SQLValidator struct {
	query      string
	normalized string
}

func NewSQLValidator(query string) *SQLValidator {
	return &SQLValidator{
		query:      query,
		normalized: normalizeSQL(query),
	}
}

// Normalizes SQL by removing extra spaces and comments while maintaining structure.
func normalizeSQL(sql string) string {
	// Remove line comments (-- )
	sql = regexp.MustCompile(`--[^\n]*`).ReplaceAllString(sql, " ")

	// Remove block comments (/* */)
	sql = regexp.MustCompile(`/\*.*?\*/`).ReplaceAllString(sql, " ")

	// Normalize multiple spaces
	sql = regexp.MustCompile(`\s+`).ReplaceAllString(sql, " ")

	// Remove spaces before/after parentheses and commas
	sql = regexp.MustCompile(`\s*([(),;])\s*`).ReplaceAllString(sql, "$1")

	return strings.TrimSpace(strings.ToUpper(sql))
}

// Remove literal strings for command parsing
func removeStringLiterals(sql string) string {
	// Remove strings enclosed in single quotes
	sql = regexp.MustCompile(`'[^']*'`).ReplaceAllString(sql, "''")

	// Remove strings enclosed in double quotes
	sql = regexp.MustCompile(`"[^"]*"`).ReplaceAllString(sql, `""`)

	// Remove strings enclosed in square brackets (SQL Server identifiers)
	sql = regexp.MustCompile(`\[[^\]]*\]`).ReplaceAllString(sql, "[]")

	return sql
}

// Verifies if the consultation is secure.
func (v *SQLValidator) Validate() error {
	// 1. Check if it's not empty
	if strings.TrimSpace(v.query) == "" {
		return fmt.Errorf("empty query")
	}

	// 2. Check maximum size (prevent DoS)
	if len(v.query) > 50000 {
		return fmt.Errorf("query too long (maximum 50000 characters)")
	}

	// 3. Check if it starts with SELECT or WITH
	if !strings.HasPrefix(v.normalized, "SELECT") && !strings.HasPrefix(v.normalized, "WITH") {
		return fmt.Errorf("Only SELECT or WITH queries are allowed")
	}

	// 4. Removing literals for command parsing
	sqlWithoutLiterals := removeStringLiterals(v.normalized)

	// 5. Dangerous DML commands (outside of strings)
	dangerousDML := []string{
		"INSERT", "UPDATE", "DELETE", "TRUNCATE", "MERGE",
	}
	for _, cmd := range dangerousDML {
		if containsKeyword(sqlWithoutLiterals, cmd) {
			return fmt.Errorf("command not allowed: %s", cmd)
		}
	}

	// 6. Dangerous DDL commands
	dangerousDDL := []string{
		"DROP", "CREATE", "ALTER", "RENAME",
	}
	for _, cmd := range dangerousDDL {
		if containsKeyword(sqlWithoutLiterals, cmd) {
			return fmt.Errorf("command not allowed: %s", cmd)
		}
	}

	// 7. Execution commands
	dangerousExec := []string{
		"EXEC", "EXECUTE", "SP_EXECUTESQL", "XP_CMDSHELL",
	}
	for _, cmd := range dangerousExec {
		if containsKeyword(sqlWithoutLiterals, cmd) {
			return fmt.Errorf("command not allowed: %s", cmd)
		}
	}

	// 8. Transaction control commands
	transactionCmds := []string{
		"BEGIN TRANSACTION", "BEGIN TRAN", "COMMIT", "ROLLBACK", "SAVE TRANSACTION",
	}
	for _, cmd := range transactionCmds {
		if strings.Contains(sqlWithoutLiterals, cmd) {
			return fmt.Errorf("Transaction commands are not allowed: %s", cmd)
		}
	}

	// 9. Backup/restore commands
	backupCmds := []string{
		"BACKUP", "RESTORE", "DUMP",
	}
	for _, cmd := range backupCmds {
		if containsKeyword(sqlWithoutLiterals, cmd) {
			return fmt.Errorf("command not allowed: %s", cmd)
		}
	}

	// 10. Administration commands
	adminCmds := []string{
		"SHUTDOWN", "RECONFIGURE", "DBCC", "KILL",
	}
	for _, cmd := range adminCmds {
		if containsKeyword(sqlWithoutLiterals, cmd) {
			return fmt.Errorf("administrative command not allowed: %s", cmd)
		}
	}

	// 11. Security commands
	securityCmds := []string{
		"GRANT", "REVOKE", "DENY",
	}
	for _, cmd := range securityCmds {
		if containsKeyword(sqlWithoutLiterals, cmd) {
			return fmt.Errorf("security command not allowed: %s", cmd)
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
			return fmt.Errorf("dangerous function not permitted: %s", fn)
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
		return fmt.Errorf("Multiple commands are not allowed")
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
	if strings.Count(sqlWithoutLiterals, "SELECT") > 10 {
		return fmt.Errorf("many subqueries (maximum 10)")
	}

	// 20. Check parenthesis depth (prevent DoS)
	if err := v.validateParenthesesDepth(); err != nil {
		return err
	}

	return nil
}

// Checks if a keyword exists as a complete word (not part of another word)
func containsKeyword(sql string, keyword string) bool {
	// Add spaces/delimiters before and after
	pattern := regexp.MustCompile(`\b` + keyword + `\b`)
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
				return fmt.Errorf("multiple commands are not allowed")
			}
		}
	}

	return nil
}

// Validates that there is no SELECT INTO statement.
func (v *SQLValidator) validateNoIntoClause(sql string) error {
	// Search for pattern SELECT ... INTO
	pattern := regexp.MustCompile(`SELECT\s+.*\s+INTO\s+`)
	if pattern.MatchString(sql) {
		return fmt.Errorf("SELECT INTO is not allowed")
	}
	return nil
}

// Valida uso de UNION (permitir apenas para queries legítimas)
func (v *SQLValidator) validateUnionUsage(sql string) error {
	// Count UNIONs
	unionCount := strings.Count(sql, "UNION")
	if unionCount > 5 {
		return fmt.Errorf("many UNION clauses (maximum 5)")
	}

	// Check if there is UNION ALL or just UNION.
	// (UNION ALL is more common in attacks)
	if unionCount > 0 {
		// Allow, but log
		// You can add logging here
	}

	return nil
}

// Validates encoding and special characters
func (v *SQLValidator) validateEncoding() error {
	// Checking for suspicious control characters
	for _, char := range v.query {
		if char < 32 && char != '\n' && char != '\r' && char != '\t' {
			return fmt.Errorf("suspicious control character detected")
		}
	}

	// Check for hexadecimal encoding attempts (0x...)
	if strings.Contains(v.normalized, "0X") {
		// Allow only in safe contexts (simple comparisons)
		hexPattern := regexp.MustCompile(`0X[0-9A-F]+`)
		matches := hexPattern.FindAllString(v.normalized, -1)
		if len(matches) > 3 {
			return fmt.Errorf("Excessive use of hexadecimal encoding")
		}
	}

	// Check CHAR / NCHAR used to obfuscate commands
	charPattern := regexp.MustCompile(`(CHAR|NCHAR)\s*\(`)
	matches := charPattern.FindAllString(v.normalized, -1)
	if len(matches) > 10 {
		return fmt.Errorf("Excessive use of CHAR/NCHAR (possible obfuscation)")
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
			return fmt.Errorf("time function not allowed: %s", fn)
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
		return fmt.Errorf("unbalanced parentheses")
	}

	if maxDepth > 20 {
		return fmt.Errorf("very large parenthesis depth (maximum 20)")
	}

	return nil
}

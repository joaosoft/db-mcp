package mcp

import (
	"fmt"
	"regexp"
)

// Validating SQL identifiers to prevent SQL injection.
func isValidIdentifier(name string) bool {
	// It allows letters, numbers, underlining, and some common special characters
	match, _ := regexp.MatchString(`^[a-zA-Z0-9_#@$]+$`, name)
	return match && len(name) > 0 && len(name) < 128
}

// Sanitize and validate schema
func getValidSchema(args map[string]interface{}, defaultSchema string) (string, error) {
	schema := defaultSchema
	if sc, ok := args["schema"].(string); ok && sc != "" {
		schema = sc
	}
	if schema != "" && !isValidIdentifier(schema) {
		return "", fmt.Errorf("nome de schema inválido: %s", schema)
	}
	return schema, nil
}

// Helper for converting string arguments safely
func getStringArg(args map[string]interface{}, key string) (string, bool) {
	val, ok := args[key].(string)
	return val, ok
}

// Helper for converting integer arguments safely
func getIntArg(args map[string]interface{}, key string, defaultVal int) int {
	if val, ok := args[key].(float64); ok {
		return int(val)
	}
	return defaultVal
}

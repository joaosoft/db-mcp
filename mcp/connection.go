package mcp

import (
	"context"
	"database/sql"
	"log"
	"os"
)

// newDbConnection creates a new database connection from environment variables.
// Returns nil connection if DB_CONNECTION_STRING is not set (allows dynamic configuration later).
// If connection string is set but connection fails, logs a warning and returns nil connection
// (server starts without connection, allowing user to reconfigure via tools).
func newDbConnection() (*sql.DB, string, error) {
	// Get database driver type (default to sqlserver for backward compatibility)
	driver := os.Getenv("DB_DRIVER")
	if driver == "" {
		driver = string(DriverSQLServer)
	}

	// Connection configuration from environment variable
	connString := os.Getenv("DB_CONNECTION_STRING")
	if connString == "" {
		// No connection string provided - server will start without database connection
		// Use configure_datasource tool to connect later
		return nil, driver, nil
	}

	db, err := sql.Open(driver, connString)
	if err != nil {
		// Log warning but don't fail - allow server to start
		log.Printf("Warning: Could not open database connection: %v. Server starting without database connection. Use configure_datasource to connect.", err)
		return nil, driver, nil
	}

	// Configure connection pool
	db.SetMaxOpenConns(DBMaxOpenConns)
	db.SetMaxIdleConns(DBMaxIdleConns)
	db.SetConnMaxLifetime(DBConnMaxLifetime)

	// Test connection with timeout
	ctx, cancel := context.WithTimeout(context.Background(), DBPingTimeout)
	defer cancel()

	if err = db.PingContext(ctx); err != nil {
		// Log warning but don't fail - allow server to start
		log.Printf("Warning: Could not connect to database: %v. Server starting without database connection. Use configure_datasource to connect.", err)
		db.Close()
		return nil, driver, nil
	}

	return db, driver, nil
}

// requireConnection checks if a database connection is available
func (s *DbMCPServer) requireConnection() error {
	if s.db == nil {
		return ErrNoConnection
	}
	return nil
}

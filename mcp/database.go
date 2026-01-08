package mcp

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"time"
)

func newDbConnection() (*sql.DB, string, error) {
	// Get database driver type (default to sqlserver for backward compatibility)
	driver := os.Getenv("DB_DRIVER")
	if driver == "" {
		driver = "sqlserver"
	}

	// Connection configuration over environment variable
	connString := os.Getenv("DB_CONNECTION_STRING")
	connString = "server=BEDC-DEV-LIS-MAD_BE.global.waiteri.com;user id=poc_readonly;password=jUK9BvXLxRDzJ97HHBA7;database=masterdata_dev_be;port=1433"
	if connString == "" {
		return nil, "", errors.New("DB_CONNECTION_STRING not defined")
	}

	db, err := sql.Open(driver, connString)
	if err != nil {
		return nil, "", errors.New(fmt.Sprintf("Error connecting to database: %v", err))
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Test connection with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err = db.PingContext(ctx); err != nil {
		return nil, "", errors.New(fmt.Sprintf("Error testing connection: %v", err))
	}

	return db, driver, nil
}

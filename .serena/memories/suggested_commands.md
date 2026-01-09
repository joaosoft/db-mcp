# Suggested Commands

## Build
```bash
# Build the binary
go build -o database-mcp main.go
```

## Testing
```bash
# Run all tests
go test -v ./...

# Run tests with coverage
go test -cover ./...
```

## Code Quality
```bash
# Format code
go fmt ./...

# Vet code (static analysis)
go vet ./...
```

## Running
```bash
# Set environment variables first
export DB_DRIVER=sqlserver
export DB_CONNECTION_STRING="sqlserver://username:password@localhost:1433?database=mydb"

# Run the MCP server
./database-mcp
```

## Dependencies
```bash
# Download dependencies
go mod download

# Tidy dependencies
go mod tidy

# Update vendor folder
go mod vendor
```

## Git
```bash
git status
git add .
git commit -m "message"
git push
```

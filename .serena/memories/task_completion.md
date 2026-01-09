# Task Completion Checklist

Before considering a task complete, run:

## 1. Format Code
```bash
go fmt ./...
```

## 2. Static Analysis
```bash
go vet ./...
```

## 3. Run Tests
```bash
go test -v ./...
```

## 4. Build
```bash
go build -o database-mcp main.go
```

## 5. If dependencies changed
```bash
go mod tidy
```

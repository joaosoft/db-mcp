.PHONY: build clean test test-coverage lint fmt vet run install deps tidy vendor help

# Binary name
BINARY_NAME=db-mcp

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt
GOVET=$(GOCMD) vet

# Build flags
LDFLAGS=-ldflags "-s -w"

# Default target
all: clean lint test build

## build: Build the binary
build:
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) main.go

## build-debug: Build with debug symbols
build-debug:
	$(GOBUILD) -o $(BINARY_NAME) main.go

## clean: Remove build artifacts
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)

## test: Run tests
test:
	$(GOTEST) -v ./...

## test-coverage: Run tests with coverage report
test-coverage:
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

## test-short: Run tests without verbose output
test-short:
	$(GOTEST) ./...

## lint: Run all linters
lint: fmt vet

## fmt: Format code
fmt:
	$(GOFMT) ./...

## vet: Run go vet
vet:
	$(GOVET) ./...

## deps: Download dependencies
deps:
	$(GOMOD) download

## tidy: Tidy go.mod
tidy:
	$(GOMOD) tidy

## vendor: Create vendor directory
vendor:
	$(GOMOD) vendor

## run: Build and run the server
run: build
	./$(BINARY_NAME)

## install: Install the binary to GOPATH/bin
install:
	$(GOCMD) install

## check: Run all checks (fmt, vet, test)
check: fmt vet test

## help: Show this help message
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'

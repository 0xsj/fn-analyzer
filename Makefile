# Variables
BINARY_NAME=fn-analyzer
BUILD_DIR=build
VERSION=1.0.0
LDFLAGS=-ldflags "-X main.Version=$(VERSION)"
CONFIG_FILE=config.json
QUERIES_FILE=critical-queries.json

# Detect OS
GOOS=$(shell go env GOOS)
GOARCH=$(shell go env GOARCH)

# Default target: build
.PHONY: all
all: build

# Build the application
.PHONY: build
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/analyzer

# Test database connection
.PHONY: test-db
test-db: build
	@echo "Testing database connection..."
	@$(BUILD_DIR)/$(BINARY_NAME) --test-connection

# Run the application
.PHONY: run
run: build
	@echo "Running $(BINARY_NAME)..."
	@$(BUILD_DIR)/$(BINARY_NAME) --config $(CONFIG_FILE)

# Run with specific queries file
.PHONY: run-queries
run-queries: build
	@echo "Running $(BINARY_NAME) with specified queries file..."
	@$(BUILD_DIR)/$(BINARY_NAME) --config $(CONFIG_FILE) --queries $(QUERIES_FILE)

# Run before fixes analysis
.PHONY: run-before
run-before: build
	@echo "Running before-fixes analysis..."
	@$(BUILD_DIR)/$(BINARY_NAME) --config $(CONFIG_FILE) --queries $(QUERIES_FILE) --label before_fixes

# Run after fixes analysis
.PHONY: run-after
run-after: build
	@echo "Running after-fixes analysis..."
	@$(BUILD_DIR)/$(BINARY_NAME) --config $(CONFIG_FILE) --queries $(QUERIES_FILE) --label after_fixes

# Run top 20 queries only
.PHONY: run-top20
run-top20: build generate-top20
	@echo "Running top 20 queries analysis..."
	@$(BUILD_DIR)/$(BINARY_NAME) --config $(CONFIG_FILE) --queries top20-queries.json --label top20_analysis

# Generate top 20 queries file (requires jq)
.PHONY: generate-top20
generate-top20:
	@echo "Generating top 20 queries file..."
	@jq '[.[] | select(.weight >= 7)] | sort_by(-.weight) | .[0:20]' $(QUERIES_FILE) > top20-queries.json
	@echo "Saved top 20 queries to top20-queries.json"

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning up..."
	@rm -rf $(BUILD_DIR)

# Install dependencies
.PHONY: deps
deps:
	@echo "Installing dependencies..."
	@go mod tidy

# Format the code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	@gofmt -s -w .

# Verify that code passes formatting check
.PHONY: verify-fmt
verify-fmt:
	@echo "Verifying code formatting..."
	@test -z "$(shell gofmt -l .)"

# Run all analysis (before, after, comparison)
.PHONY: analyze-all
analyze-all: build run-before run-after
	@echo "Analysis complete. Check performance-results directory for reports."

# Help
.PHONY: help
help:
	@echo "FN Analyzer - Performance testing tool for databases"
	@echo ""
	@echo "Usage:"
	@echo "  make build       Build the application"
	@echo "  make test-db     Test database connection only"
	@echo "  make run         Run the application with default config"
	@echo "  make run-queries Run with specific queries file"
	@echo "  make run-before  Run before-fixes analysis"
	@echo "  make run-after   Run after-fixes analysis"
	@echo "  make run-top20   Run analysis on top 20 queries"
	@echo "  make clean       Clean build artifacts"
	@echo "  make deps        Install dependencies"
	@echo "  make fmt         Format the code"
	@echo "  make config      Create example config file"
	@echo "  make analyze-all Run complete before/after analysis"
	@echo "  make help        Show this help message"
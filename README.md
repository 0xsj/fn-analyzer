# Database Performance Analyzer

The Database Performance Analyzer helps you identify and measure the impact of database schema fixes, particularly focusing on model inconsistencies and relationship issues. It's designed to work with critical queries extracted from model validation tests, providing precise metrics on how schema changes affect query performance.

## Features

- 📊 **Detailed Performance Metrics**: Measure execution time, row counts, and error rates
- 🔄 **Before/After Comparison**: Compare performance metrics before and after schema fixes
- 🌐 **Connection Pool Management**: Test under realistic load with concurrent connections
- 📈 **Query Complexity Analysis**: Categorize queries by complexity level for deeper insights
- 📋 **Multiple Output Formats**: Generate detailed JSON reports and CSV files for easy analysis
- 🔍 **Error Pattern Detection**: Identify and classify database error patterns

## Installation

### Prerequisites

- Go 1.20 or higher
- MySQL-compatible database
- (Optional) `jq` for advanced query filtering

```bash


# Install dependencies
make deps

# Build the application
make build
```

## Quick Start

1. **Configure database connection**:

   ```bash
   make config
   ```

2. **Copy your critical queries file**: Place your critical queries JSON file (generated from model validation) in the project root as `critical-queries.json`.

3. **Test database connection**:

   ```bash
   make test-db
   ```

4. **Run analysis**:
   ```bash
   make run
   ```

## Detailed Usage

### Configuring the Analyzer

Edit `config.json` to configure the analyzer:

```json
{
  "dsn": "username:password@tcp(hostname:port)/database?tls=skip-verify",
  "queriesFile": "critical-queries.json",
  "outputDir": "performance-results",
  "iterations": 50,
  "concurrency": 5,
  "warmupIterations": 100,
  "label": "baseline",
  "timeoutSeconds": 30,
  "verbose": false
}
```

### Query JSON Format

The critical queries file must follow this format:

```json
[
  {
    "name": "consistency_TableA_TableB",
    "description": "Testing join with type mismatch: TableA.fieldX vs TableB.fieldY",
    "sql": "SELECT a.id, b.name FROM table_a a JOIN table_b b ON a.field_x = b.field_y LIMIT 100",
    "weight": 10
  },
  {
    "name": "datatype_TableC",
    "description": "Testing table with datatype issues",
    "sql": "SELECT problem_field FROM table_c LIMIT 100",
    "weight": 7
  }
]
```

Fields:

- `name`: Unique identifier for the query
- `description`: Human-readable description
- `sql`: The SQL query to test
- `weight`: Importance weight (higher = more critical)

## Running Performance Tests

### Testing Database Connection

```bash
make test-db
```

Tests the database connection without running the full analyzer.

### Running Analysis with Current Configuration

```bash
make run
```

### Running Analysis Before Schema Fixes

```bash
make run-before
```

Uses the label "before_fixes" to mark baseline measurements.

### Running Analysis After Schema Fixes

```bash
make run-after
```

Uses the label "after_fixes" to mark post-fix measurements.

### Running Analysis on Top 20 Queries

```bash
make run-top20
```

Filters to the 20 highest-weight queries for faster testing.

### Running Complete Analysis Workflow

```bash
make analyze-all
```

Runs both before and after analysis automatically.

## Understanding Reports

The analyzer generates several output files in the `performance-results` directory:

1. **JSON Reports**: `performance-{label}-{timestamp}.json`

   - Complete performance data including all metrics
   - Query execution times, row counts, and errors
   - Database connection information

2. **CSV Reports**: `performance-{label}-{timestamp}.csv`

   - Simplified format for import into spreadsheets
   - One row per query with key metrics

3. **Console Summary**
   - Top slowest queries
   - Error patterns
   - Overall performance statistics

## Common Use Cases

### Finding Problematic Relationships

Run the analyzer focusing on relationship queries:

```bash
make run-queries QUERIES_FILE=relationship-queries.json
```

### Testing Under High Concurrency

Modify your config to test with higher concurrency:

```json
{
  "concurrency": 20,
  "iterations": 100
}
```

### Troubleshooting Database Lockups

Use high concurrency with verbose logging:

```json
{
  "concurrency": 15,
  "verbose": true
}
```

### Quick Iteration During Development

To get faster feedback during development:

```json
{
  "iterations": 10,
  "warmupIterations": 20,
  "label": "quick_test"
}
```

## Advanced Usage

### Filtering Queries with jq

Generate a subset of queries:

```bash
# Generate top 20 queries by weight
jq '[.[] | select(.weight >= 7)] | sort_by(-.weight) | .[0:20]' critical-queries.json > top20-queries.json

# Generate only consistency queries
jq '[.[] | select(.name | startswith("consistency_"))]' critical-queries.json > consistency-queries.json
```

## Makefile Commands

| Command            | Description                             |
| ------------------ | --------------------------------------- |
| `make build`       | Build the application                   |
| `make test-db`     | Test database connection                |
| `make run`         | Run analysis with current configuration |
| `make run-before`  | Run "before fixes" analysis             |
| `make run-after`   | Run "after fixes" analysis              |
| `make run-top20`   | Run analysis on top 20 queries          |
| `make clean`       | Clean build artifacts                   |
| `make deps`        | Install dependencies                    |
| `make fmt`         | Format code                             |
| `make config`      | Create example config file              |
| `make analyze-all` | Run complete before/after analysis      |
| `make help`        | Show help message                       |

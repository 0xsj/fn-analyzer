// internal/analyzer/query.go
package analyzer

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/0xsj/fn-analyzer/internal/config"
	"github.com/0xsj/fn-analyzer/internal/model"
	"github.com/0xsj/fn-analyzer/pkg/utils"
)

type QueryExecutor struct {
	db          *sql.DB
	timeout     time.Duration
	verbose     bool
	concurrency int
	semaphore   chan struct{}
	mutex       sync.Mutex
}

func NewQueryExecutor(db *sql.DB, cfg config.Config) *QueryExecutor {
	return &QueryExecutor{
		db:          db,
		timeout:     cfg.Timeout,
		verbose:     cfg.Verbose,
		concurrency: cfg.Concurrency,
		semaphore:   make(chan struct{}, cfg.Concurrency),
	}
}

func (qe *QueryExecutor) ExecuteQuery(query string) model.QueryExecution {
	execution := model.QueryExecution{
		StartTime: time.Now(),
		SQL:       query,
	}

	ctx, cancel := context.WithTimeout(context.Background(), qe.timeout)
	defer cancel()

	start := time.Now()
	rows, err := qe.db.QueryContext(ctx, query)
	execution.Duration = time.Since(start)

	if err != nil {
		execution.Error = err
		execution.ErrorMessage = err.Error()
		return execution
	}
	defer rows.Close()

	var rowCount int64
	for rows.Next() {
		rowCount++
	}
	execution.RowCount = rowCount

	if err = rows.Err(); err != nil {
		execution.Error = err
		execution.ErrorMessage = err.Error()
	}

	return execution
}

func (qe *QueryExecutor) ExecuteBatch(queries []model.Query, iterations int) []model.QueryResult {
	results := make([]model.QueryResult, len(queries))
	var wg sync.WaitGroup

	for i, query := range queries {
		results[i] = model.QueryResult{
			Name:         query.Name,
			Description:  query.Description,
			SQL:          query.SQL,
			MinDuration:  time.Hour, 
			Weight:       query.Weight,
			QueryComplexity: AnalyzeQueryComplexity(query.SQL),
			Executions:   make([]model.QueryExecution, 0, iterations),
		}
	}

	for i, query := range queries {
		wg.Add(1)
		go func(idx int, q model.Query) {
			defer wg.Done()
			result := &results[idx]

			if qe.verbose {
				log.Printf("Testing query: %s", q.Name)
			}

			for iter := 0; iter < iterations; iter++ {
				qe.semaphore <- struct{}{}

				execution := qe.ExecuteQuery(q.SQL)
				
				<-qe.semaphore

				if len(result.Executions) == 0 {
					result.FirstExecutedAt = execution.StartTime
				}
				
				result.LastExecutedAt = execution.StartTime

				qe.mutex.Lock()

				result.Executions = append(result.Executions, execution)

				if execution.Error != nil {
					result.Errors++
					if len(result.ErrorDetails) < 10 { 
						result.ErrorDetails = append(result.ErrorDetails, execution.ErrorMessage)
					}
				} else {
					result.SuccessfulExecutions++
					result.TotalDuration += execution.Duration
					result.RowsAffected += execution.RowCount

					if execution.Duration < result.MinDuration {
						result.MinDuration = execution.Duration
					}
					if execution.Duration > result.MaxDuration {
						result.MaxDuration = execution.Duration
					}
				}

				qe.mutex.Unlock()

				if qe.verbose && (iter == 0 || (iter+1) % 10 == 0) {
					if execution.Error != nil {
						log.Printf("Query %s iteration %d: ERROR - %s", 
							q.Name, iter+1, execution.ErrorMessage)
					} else {
						log.Printf("Query %s iteration %d: %v, %d rows", 
							q.Name, iter+1, execution.Duration, execution.RowCount)
					}
				}
			}

			if result.SuccessfulExecutions > 0 {
				result.AvgDuration = result.TotalDuration / time.Duration(result.SuccessfulExecutions)
				
				durations := make([]time.Duration, 0, result.SuccessfulExecutions)
				for _, exec := range result.Executions {
					if exec.Error == nil {
						durations = append(durations, exec.Duration)
					}
				}
				
				if len(durations) > 0 {
					stats := utils.CalculateStats(durations)
					result.Percentile95 = stats.P95
					result.Percentile99 = stats.P99
					result.StdDevDuration = stats.StdDev
					result.MedianDuration = stats.Median
				}
			}

			if qe.verbose {
				avgMs := float64(result.AvgDuration.Microseconds()) / 1000
				p95Ms := float64(result.Percentile95.Microseconds()) / 1000
				
				log.Printf("Results for %s: %.2f ms avg, %.2f ms p95, %d rows, %s complexity", 
					q.Name, avgMs, p95Ms, result.RowsAffected, result.QueryComplexity)
			}
		}(i, query)
	}

	wg.Wait()
	return results
}

func CreateTestQueries(allQueries []model.Query, testType string, limit int) ([]model.Query, error) {
	switch testType {
	case "all":
		return allQueries, nil
		
	case "consistency":
		return filterQueriesByType(allQueries, "consistency", limit)
		
	case "datatype":
		return filterQueriesByType(allQueries, "datatype", limit)
		
	case "relationship":
		return filterQueriesByType(allQueries, "relationship", limit)
		
	case "top":
		sortedQueries := make([]model.Query, len(allQueries))
		copy(sortedQueries, allQueries)
		sort.Slice(sortedQueries, func(i, j int) bool {
			return sortedQueries[i].Weight > sortedQueries[j].Weight
		})
		
		if limit > 0 && limit < len(sortedQueries) {
			return sortedQueries[:limit], nil
		}
		return sortedQueries, nil
		
	default:
		return nil, fmt.Errorf("unknown test type: %s", testType)
	}
}

func filterQueriesByType(allQueries []model.Query, queryType string, limit int) ([]model.Query, error) {
	var filtered []model.Query
	
	for _, q := range allQueries {
		if strings.HasPrefix(strings.ToLower(q.Name), strings.ToLower(queryType)) {
			filtered = append(filtered, q)
		}
	}
	
	if len(filtered) == 0 {
		return nil, fmt.Errorf("no queries found of type: %s", queryType)
	}
	
	if limit > 0 && limit < len(filtered) {
		return filtered[:limit], nil
	}
	
	return filtered, nil
}

func SaveTestQueries(queries []model.Query, outputPath string) error {
	data, err := json.MarshalIndent(queries, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling queries: %w", err)
	}
	
	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("error writing queries file: %w", err)
	}
	
	log.Printf("Saved %d queries to %s", len(queries), outputPath)
	return nil
}

func ClassifyErrors(results []model.QueryResult) map[string]int {
	errorTypes := make(map[string]int)
	
	for _, result := range results {
		for _, errMsg := range result.ErrorDetails {
			errType := classifyErrorMessage(errMsg)
			errorTypes[errType]++
		}
	}
	
	return errorTypes
}

func classifyErrorMessage(errMsg string) string {
	errMsg = strings.ToLower(errMsg)
	
	if strings.Contains(errMsg, "deadlock") {
		return "Deadlock"
	} else if strings.Contains(errMsg, "lock wait timeout") {
		return "Lock timeout"
	} else if strings.Contains(errMsg, "foreign key constraint") {
		return "Foreign key constraint"
	} else if strings.Contains(errMsg, "duplicate entry") {
		return "Duplicate entry"
	} else if strings.Contains(errMsg, "truncated") || strings.Contains(errMsg, "out of range") {
		return "Data truncation/range"
	} else if strings.Contains(errMsg, "convert") || strings.Contains(errMsg, "illegal mix") {
		return "Type conversion"
	} else if strings.Contains(errMsg, "context deadline") || strings.Contains(errMsg, "timeout") {
		return "Query timeout"
	} else {
		return "Other error"
	}
}

func GenerateQueryExplain(db *sql.DB, query string) (string, error) {
	if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(query)), "select") {
		return "EXPLAIN not available for non-SELECT queries", nil
	}
	
	explainQuery := "EXPLAIN FORMAT=JSON " + query
	var explainResult string
	
	err := db.QueryRow(explainQuery).Scan(&explainResult)
	if err != nil {
		rows, err := db.Query("EXPLAIN " + query)
		if err != nil {
			return "", fmt.Errorf("error getting query explain plan: %w", err)
		}
		defer rows.Close()
		
		var result strings.Builder
		columns, err := rows.Columns()
		if err != nil {
			return "", err
		}
		
		result.WriteString(strings.Join(columns, " | "))
		result.WriteString("\n")
		for range columns {
			result.WriteString("--- | ")
		}
		result.WriteString("\n")
		
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}
		
		for rows.Next() {
			if err := rows.Scan(valuePtrs...); err != nil {
				return "", err
			}
			
			for i, val := range values {
				var valStr string
				b, ok := val.([]byte)
				if ok {
					valStr = string(b)
				} else {
					valStr = fmt.Sprintf("%v", val)
				}
				
				if i > 0 {
					result.WriteString(" | ")
				}
				result.WriteString(valStr)
			}
			result.WriteString("\n")
		}
		
		return result.String(), nil
	}
	
	return explainResult, nil
}
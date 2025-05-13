// internal/analyzer/analyzer.go
package analyzer

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/0xsj/fn-analyzer/internal/config"
	"github.com/0xsj/fn-analyzer/internal/database"
	"github.com/0xsj/fn-analyzer/internal/model"
	"github.com/0xsj/fn-analyzer/internal/report"
)

type Analyzer struct {
	db          *sql.DB
	queries     []model.Query
	config      config.Config
	concurrency int
	iterations  int
	timeout     time.Duration
	verbose     bool
}

func NewAnalyzer(db *sql.DB, queries []model.Query, cfg config.Config) *Analyzer {
	return &Analyzer{
		db:          db,
		queries:     queries,
		config:      cfg,
		concurrency: cfg.Concurrency,
		iterations:  cfg.Iterations,
		timeout:     cfg.Timeout,
		verbose:     cfg.Verbose,
	}
}

func LoadQueries(path string) ([]model.Query, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading queries file: %w", err)
	}

	var queries []model.Query
	if err := json.Unmarshal(data, &queries); err != nil {
		return nil, fmt.Errorf("error parsing queries file: %w", err)
	}

	return queries, nil
}

func WarmupConnectionPool(db *sql.DB, iterations int) error {
	log.Printf("Warming up connection pool with %d iterations...", iterations)
	
	start := time.Now()
	warmupQuery := "SELECT 1"
	
	var wg sync.WaitGroup
	
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := db.Exec(warmupQuery)
			if err != nil {
				log.Printf("Warmup error: %v", err)
			}
		}()
	}
	
	wg.Wait()
	
	log.Printf("Warmup completed in %v", time.Since(start))
	return nil
}

func (a *Analyzer) Run() ([]model.QueryResult, error) {
	var results []model.QueryResult
	resultsMutex := sync.Mutex{}
	semaphore := make(chan struct{}, a.concurrency)

	for _, query := range a.queries {
		result := model.QueryResult{
			Name:        query.Name,
			Description: query.Description,
			SQL:         query.SQL,
			MinDuration: time.Hour, 
			Weight:      query.Weight,
			QueryComplexity: AnalyzeQueryComplexity(query.SQL),
			Executions:  make([]model.QueryExecution, 0, a.iterations),
		}

		var durations []time.Duration
		var wg sync.WaitGroup
		resultMutex := sync.Mutex{}

		log.Printf("Testing query: %s", query.Name)

		for i := 0; i < a.iterations; i++ {
			wg.Add(1)
			semaphore <- struct{}{} 

			go func(iteration int) {
				defer wg.Done()
				defer func() { <-semaphore }() 

				queryResult := a.executeQuery(query.SQL)
				
				resultMutex.Lock()
				defer resultMutex.Unlock()

				if len(result.Executions) == 0 {
					result.FirstExecutedAt = queryResult.startTime
				}
				
				result.LastExecutedAt = queryResult.startTime

				execution := model.QueryExecution{
					SQL:          query.SQL,
					StartTime:    queryResult.startTime,
					Duration:     queryResult.duration,
					RowCount:     queryResult.rowCount,
				}

				if queryResult.err != nil {
					execution.ErrorMessage = queryResult.err.Error()
					result.Errors++
					if len(result.ErrorDetails) < 10 { 
						result.ErrorDetails = append(result.ErrorDetails, queryResult.err.Error())
					}
					
					result.Executions = append(result.Executions, execution)
					return
				}

				result.SuccessfulExecutions++ 
				result.TotalDuration += queryResult.duration
				result.RowsAffected += queryResult.rowCount
				durations = append(durations, queryResult.duration)

				result.Executions = append(result.Executions, execution)

				if queryResult.duration < result.MinDuration {
					result.MinDuration = queryResult.duration
				}
				if queryResult.duration > result.MaxDuration {
					result.MaxDuration = queryResult.duration
				}

				if a.verbose && (iteration == 0 || (iteration+1) % 10 == 0) {
					log.Printf("Query %s iteration %d: %v, %d rows", 
						query.Name, iteration+1, queryResult.duration, queryResult.rowCount)
				}
			}(i)
		}

		wg.Wait()

		if result.SuccessfulExecutions > 0 {
			result.AvgDuration = result.TotalDuration / time.Duration(result.SuccessfulExecutions)
		}

		if len(durations) > 0 {
			sort.Slice(durations, func(i, j int) bool {
				return durations[i] < durations[j]
			})
			idx95 := int(float64(len(durations)) * 0.95)
			if idx95 >= len(durations) {
				idx95 = len(durations) - 1
			}
			result.Percentile95 = durations[idx95]
		}

		resultsMutex.Lock()
		results = append(results, result)
		resultsMutex.Unlock()

		avgMs := float64(result.AvgDuration.Microseconds()) / 1000
		p95Ms := float64(result.Percentile95.Microseconds()) / 1000
		
		log.Printf("  Results: %.2f ms avg, %.2f ms p95, %d rows, %s complexity", 
			avgMs, p95Ms, result.RowsAffected, result.QueryComplexity)
	}

	return results, nil
}

type queryResult struct {
	duration  time.Duration
	rowCount  int64
	err       error
	startTime time.Time
}

func (a *Analyzer) executeQuery(sql string) queryResult {
	result := queryResult{
		startTime: time.Now(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), a.timeout)
	defer cancel()

	rows, err := a.db.QueryContext(ctx, sql)
	result.duration = time.Since(result.startTime)
	
	if err != nil {
		result.err = err
		return result
	}
	defer rows.Close()

	for rows.Next() {
		result.rowCount++
	}

	if err = rows.Err(); err != nil {
		result.err = err
	}

	return result
}

func GenerateReports(results []model.QueryResult, connInfo database.ConnectionInfo, cfg config.Config, duration time.Duration) error {
	summary := calculateSummary(results)
	
	testResult := model.TestResult{
		Timestamp:      time.Now(),
		Label:          cfg.Label,
		Config:         cfg,
		TotalDuration:  duration,
		QueryResults:   results,
		ConnectionInfo: connInfo,
		Summary:        summary,
	}
	
	if err := report.SaveJSON(testResult, cfg.OutputDir); err != nil {
		return fmt.Errorf("error saving JSON report: %w", err)
	}
	
	if err := report.SaveCSV(testResult, cfg.OutputDir); err != nil {
		return fmt.Errorf("error saving CSV report: %w", err)
	}
	
	report.PrintSummary(testResult)
	
	return nil
}

func calculateSummary(results []model.QueryResult) model.ResultSummary {
	summary := model.ResultSummary{
		TotalQueries: len(results),
		QueriesByComplexity: make(map[string]int),
	}
	
	var totalDuration time.Duration
	var maxDuration time.Duration
	
	for _, result := range results {
		summary.TotalExecutions += len(result.Executions)  
		summary.SuccessfulExecutions += result.SuccessfulExecutions 
		summary.FailedExecutions += result.Errors
		summary.TotalRowsReturned += result.RowsAffected
		
		if result.Errors == 0 {
			summary.SuccessfulQueries++
		} else {
            summary.FailedQueries++
        }
		
		totalDuration += result.AvgDuration
		if result.MaxDuration > maxDuration {
			maxDuration = result.MaxDuration
		}
		
		summary.QueriesByComplexity[result.QueryComplexity]++
	}
	
	if summary.TotalQueries > 0 {
		avgDuration := totalDuration / time.Duration(summary.TotalQueries)
		summary.AvgDurationMs = float64(avgDuration.Microseconds()) / 1000
		summary.MaxDurationMs = float64(maxDuration.Microseconds()) / 1000
	}
	
	return summary
}
// internal/report/formatter.go
package report

import (
	"fmt"
	"sort"
	"time"

	"github.com/0xsj/fn-analyzer/internal/model"
)

func PrintSummary(result model.TestResult) {
	fmt.Println("\n====== PERFORMANCE TEST SUMMARY ======")
	fmt.Printf("Test Label: %s\n", result.Label)
	fmt.Printf("Total Duration: %v\n", result.TotalDuration)
	fmt.Printf("Queries: %d total, %d successful, %d with errors\n", 
		result.Summary.TotalQueries, 
		result.Summary.SuccessfulQueries,
		result.Summary.TotalQueries - result.Summary.SuccessfulQueries)
	fmt.Printf("Average Query Time: %.2f ms\n", result.Summary.AvgDurationMs)
	fmt.Printf("Max Query Time: %.2f ms\n", result.Summary.MaxDurationMs)
	fmt.Printf("Total Rows Returned: %d\n", result.Summary.TotalRowsReturned)
	
	fmt.Println("\nQuery Complexity Distribution:")
	complexities := make([]string, 0, len(result.Summary.QueriesByComplexity))
	for complexity := range result.Summary.QueriesByComplexity {
		complexities = append(complexities, complexity)
	}
	sort.Strings(complexities)
	
	for _, complexity := range complexities {
		count := result.Summary.QueriesByComplexity[complexity]
		fmt.Printf("  %s: %d queries (%.1f%%)\n", 
			complexity, 
			count, 
			float64(count)/float64(result.Summary.TotalQueries)*100)
	}
	
	fmt.Println("\nTop 5 Slowest Queries:")
	sortedResults := make([]model.QueryResult, len(result.QueryResults))
	copy(sortedResults, result.QueryResults)
	sort.Slice(sortedResults, func(i, j int) bool {
		return sortedResults[i].AvgDuration > sortedResults[j].AvgDuration
	})
	
	for i, q := range sortedResults {
		if i >= 5 {
			break
		}
		avgMs := float64(q.AvgDuration.Microseconds()) / 1000
		fmt.Printf("  %d. %s: %.2f ms avg, %d rows, %s complexity\n", 
			i+1, q.Name, avgMs, q.RowsAffected, q.QueryComplexity)
	}
	
	fmt.Println("\nTop 5 Queries with Errors:")
	sort.Slice(sortedResults, func(i, j int) bool {
		return sortedResults[i].Errors > sortedResults[j].Errors
	})
	
	errorCount := 0
	for _, q := range sortedResults {
		if q.Errors == 0 {
			continue
		}
		
		errorCount++
		if errorCount > 5 {
			break
		}
		
		fmt.Printf("  %d. %s: %d errors\n", errorCount, q.Name, q.Errors)
		if len(q.ErrorDetails) > 0 {
			fmt.Printf("     First error: %s\n", q.ErrorDetails[0])
		}
	}
	
	if errorCount == 0 {
		fmt.Println("  No queries with errors")
	}
	
	fmt.Println("\nDatabase Information:")
	fmt.Printf("  Version: %s\n", result.ConnectionInfo.Version)
	fmt.Printf("  Threads Running: %d\n", result.ConnectionInfo.ThreadsRunning)
	fmt.Printf("  Threads Connected: %d\n", result.ConnectionInfo.ThreadsConnected)
	fmt.Printf("  Open Tables: %d\n", result.ConnectionInfo.OpenTables)
	fmt.Printf("  Slow Queries: %d\n", result.ConnectionInfo.SlowQueries)
	fmt.Printf("  Questions/sec: %.2f\n", result.ConnectionInfo.QuestionsPerSec)
	
	fmt.Println("\nTest Completed At:", time.Now().Format(time.RFC1123))
	fmt.Println("======================================")
}

func FormatDuration(d time.Duration) string {
	if d < time.Microsecond {
		return fmt.Sprintf("%.2f ns", float64(d.Nanoseconds()))
	} else if d < time.Millisecond {
		return fmt.Sprintf("%.2f μs", float64(d.Nanoseconds())/1000)
	} else if d < time.Second {
		return fmt.Sprintf("%.2f ms", float64(d.Nanoseconds())/1000000)
	} else if d < time.Minute {
		return fmt.Sprintf("%.2f s", d.Seconds())
	} else {
		return fmt.Sprintf("%.2f min", d.Minutes())
	}
}
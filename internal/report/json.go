// internal/report/json.go
package report

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/0xsj/fn-analyzer/internal/model"
)

func SaveJSON(result model.TestResult, outputDir string) error {
	timestamp := time.Now().Format("20060102-150405")
	label := result.Label
	if label == "" {
		label = "test"
	}
	
	filename := filepath.Join(outputDir, fmt.Sprintf("performance-%s-%s.json", label, timestamp))
	
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling results: %w", err)
	}
	
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("error writing results file: %w", err)
	}
	
	log.Printf("JSON results saved to %s", filename)
	return nil
}

func SaveSummaryJSON(result model.TestResult, outputDir string) error {
	timestamp := time.Now().Format("20060102-150405")
	label := result.Label
	if label == "" {
		label = "test"
	}
	
	filename := filepath.Join(outputDir, fmt.Sprintf("summary-%s-%s.json", label, timestamp))
	
	summary := struct {
		Timestamp      time.Time            `json:"timestamp"`
		Label          string               `json:"label"`
		TotalDuration  string               `json:"totalDuration"`
		Summary        model.ResultSummary  `json:"summary"`
		ConnectionInfo interface{}          `json:"connectionInfo"`
		TopQueries     []interface{}        `json:"topQueries"`
	}{
		Timestamp:      result.Timestamp,
		Label:          result.Label,
		TotalDuration:  result.TotalDuration.String(),
		Summary:        result.Summary,
		ConnectionInfo: result.ConnectionInfo,
	}
	
	if len(result.QueryResults) > 0 {
		sortedResults := make([]model.QueryResult, len(result.QueryResults))
		copy(sortedResults, result.QueryResults)
		sort.Slice(sortedResults, func(i, j int) bool {
			return sortedResults[i].AvgDuration > sortedResults[j].AvgDuration
		})
		
		topQueries := make([]interface{}, 0, 5)
		
		for i, q := range sortedResults {
			if i >= 5 {
				break
			}
			
			type querySummary struct {
				Name        string  `json:"name"`
				AvgDuration float64 `json:"avgDurationMs"`
				Executions  int     `json:"executions"`
				Errors      int     `json:"errors"`
				Rows        int64   `json:"rows"`
				Complexity  string  `json:"complexity"`
			}
			
			qs := querySummary{
				Name:        q.Name,
				AvgDuration: float64(q.AvgDuration.Microseconds()) / 1000,
				Executions:  q.SuccessfulExecutions, 
				Errors:      q.Errors,
				Rows:        q.RowsAffected,
				Complexity:  q.QueryComplexity,
			}
			
			topQueries = append(topQueries, qs)
		}
		
		summary.TopQueries = topQueries
	}
	
	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling summary: %w", err)
	}
	
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("error writing summary file: %w", err)
	}
	
	log.Printf("Summary JSON saved to %s", filename)
	return nil
}

func SaveComparisonJSON(before, after model.TestResult, outputDir string) error {
	timestamp := time.Now().Format("20060102-150405")
	filename := filepath.Join(outputDir, fmt.Sprintf("comparison-%s-vs-%s-%s.json", 
		before.Label, after.Label, timestamp))
	
	afterMap := make(map[string]model.QueryResult)
	for _, q := range after.QueryResults {
		afterMap[q.Name] = q
	}
	
	comparisons := make([]model.QueryComparison, 0, len(before.QueryResults))
	
	for _, beforeQ := range before.QueryResults {
		afterQ, found := afterMap[beforeQ.Name]
		if !found {
			continue
		}
		
		beforeAvgMs := float64(beforeQ.AvgDuration.Microseconds()) / 1000
		afterAvgMs := float64(afterQ.AvgDuration.Microseconds()) / 1000
		
		var improvementPct float64
		if beforeAvgMs > 0 {
			improvementPct = (beforeAvgMs - afterAvgMs) / beforeAvgMs * 100
		}
		
		comparison := model.QueryComparison{
			Name:               beforeQ.Name,
			BeforeAvgMs:        beforeAvgMs,
			AfterAvgMs:         afterAvgMs,
			ImprovementPercent: improvementPct,
			BeforeErrors:       beforeQ.Errors,
			AfterErrors:        afterQ.Errors,
			BeforeRows:         beforeQ.RowsAffected,
			AfterRows:          afterQ.RowsAffected,
		}
		
		comparisons = append(comparisons, comparison)
	}
	
	sort.Slice(comparisons, func(i, j int) bool {
		return comparisons[i].ImprovementPercent > comparisons[j].ImprovementPercent
	})
	
	var beforeTotal, afterTotal time.Duration
	var beforeCount, afterCount int
	
	for _, q := range before.QueryResults {
		if q.SuccessfulExecutions > 0 {
			beforeTotal += q.AvgDuration
			beforeCount++
		}
	}
	
	for _, q := range after.QueryResults {
		if q.SuccessfulExecutions > 0 {
			afterTotal += q.AvgDuration
			afterCount++
		}
	}
	
	var avgTimeImprovement float64
	if beforeCount > 0 && afterCount > 0 {
		beforeAvg := float64(beforeTotal.Microseconds()) / float64(beforeCount) / 1000
		afterAvg := float64(afterTotal.Microseconds()) / float64(afterCount) / 1000
		
		if beforeAvg > 0 {
			avgTimeImprovement = (beforeAvg - afterAvg) / beforeAvg * 100
		}
	}
	
	comparison := model.ComparisonResult{
		Before: before,
		After:  after,
		ImprovementSummary: model.ImprovementStats{
			AvgTimeImprovement: avgTimeImprovement,
		},
		QueryComparisons: comparisons,
	}
	
	data, err := json.MarshalIndent(comparison, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling comparison: %w", err)
	}
	
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("error writing comparison file: %w", err)
	}
	
	log.Printf("Comparison results saved to %s", filename)
	return nil
}
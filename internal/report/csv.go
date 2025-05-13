// internal/report/csv.go
package report

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/0xsj/fn-analyzer/internal/model"
)

func SaveCSV(result model.TestResult, outputDir string) error {
	timestamp := time.Now().Format("20060102-150405")
	label := result.Label
	if label == "" {
		label = "test"
	}

	filename := filepath.Join(outputDir, fmt.Sprintf("performance-%s-%s.csv", label, timestamp))

	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("error creating CSV file: %w", err)
	}
	defer f.Close()

	f.WriteString("name,description,executions,errors,avg_ms,p95_ms,min_ms,max_ms,rows,complexity\n")

	for _, q := range result.QueryResults {
		avg := float64(q.AvgDuration.Microseconds()) / 1000
		p95 := float64(q.Percentile95.Microseconds()) / 1000
		min := float64(q.MinDuration.Microseconds()) / 1000
		max := float64(q.MaxDuration.Microseconds()) / 1000

		desc := strings.ReplaceAll(q.Description, "\"", "\"\"")
		desc = strings.ReplaceAll(desc, ",", " ")

		line := fmt.Sprintf("\"%s\",\"%s\",%d,%d,%.2f,%.2f,%.2f,%.2f,%d,%s\n",
			q.Name, desc, len(q.Executions), q.Errors,
			avg, p95, min, max, q.RowsAffected, q.QueryComplexity)

		f.WriteString(line)
	}

	log.Printf("CSV results saved to %s", filename)
	return nil
}

func SaveDetailedCSV(result model.TestResult, outputDir string) error {
	timestamp := time.Now().Format("20060102-150405")
	label := result.Label
	if label == "" {
		label = "test"
	}

	filename := filepath.Join(outputDir, fmt.Sprintf("performance-detailed-%s-%s.csv", label, timestamp))

	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("error creating detailed CSV file: %w", err)
	}
	defer f.Close()

	f.WriteString("name,description,sql,executions,errors,avg_ms,p95_ms,min_ms,max_ms,rows,complexity\n")

	for _, q := range result.QueryResults {
		avg := float64(q.AvgDuration.Microseconds()) / 1000
		p95 := float64(q.Percentile95.Microseconds()) / 1000
		min := float64(q.MinDuration.Microseconds()) / 1000
		max := float64(q.MaxDuration.Microseconds()) / 1000

		desc := strings.ReplaceAll(q.Description, "\"", "\"\"")
		desc = strings.ReplaceAll(desc, ",", " ")

		sql := strings.ReplaceAll(q.SQL, "\"", "\"\"")
		sql = strings.ReplaceAll(sql, ",", " ")
		sql = strings.ReplaceAll(sql, "\n", " ")

		line := fmt.Sprintf("\"%s\",\"%s\",%d,%d,%.2f,%.2f,%.2f,%.2f,%d,%s\n",
			q.Name, desc, len(q.Executions), q.Errors,
			avg, p95, min, max, q.RowsAffected, q.QueryComplexity)

		f.WriteString(line)
	}

	log.Printf("Detailed CSV results saved to %s", filename)
	return nil
}

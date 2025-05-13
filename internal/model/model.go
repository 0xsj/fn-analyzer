// internal/model/model.go
package model

import (
	"time"

	"github.com/0xsj/fn-analyzer/internal/config"
	"github.com/0xsj/fn-analyzer/internal/database"
)

type Query struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	SQL         string `json:"sql"`
	Weight      int    `json:"weight"`
}

// QueryExecution represents a single execution of a query
type QueryExecution struct {
	SQL           string        `json:"sql"`
	StartTime     time.Time     `json:"startTime"`
	Duration      time.Duration `json:"duration"`
	RowCount      int64         `json:"rowCount"`
	Error         error         `json:"-"` 
	ErrorMessage  string        `json:"error,omitempty"`
}

// QueryResult represents the performance metrics for a query
type QueryResult struct {
	Name             string           `json:"name"`
	Description      string           `json:"description"`
	SQL              string           `json:"sql"`
	Executions       []QueryExecution `json:"executions,omitempty"`
	SuccessfulExecutions int          `json:"successfulExecutions"`
	Errors           int              `json:"errors"`
	ErrorDetails     []string         `json:"errorDetails,omitempty"`
	TotalDuration    time.Duration    `json:"totalDurationNs"`
	AvgDuration      time.Duration    `json:"avgDurationNs"`
	MinDuration      time.Duration    `json:"minDurationNs"`
	MaxDuration      time.Duration    `json:"maxDurationNs"`
	MedianDuration   time.Duration    `json:"medianDurationNs"`
	StdDevDuration   time.Duration    `json:"stdDevDurationNs"`
	Percentile95     time.Duration    `json:"percentile95Ns"`
	Percentile99     time.Duration    `json:"percentile99Ns"`
	RowsAffected     int64            `json:"rowsAffected"`
	Weight           int              `json:"weight"`
	QueryComplexity  string           `json:"queryComplexity"`
	FirstExecutedAt  time.Time        `json:"firstExecutedAt"`
	LastExecutedAt   time.Time        `json:"lastExecutedAt"`
	ExplainPlan      string           `json:"explainPlan,omitempty"`
}

// TestResult represents the overall results of a performance test
type TestResult struct {
	Timestamp      time.Time         `json:"timestamp"`
	Label          string            `json:"label"`
	Config         config.Config     `json:"config"`
	TotalDuration  time.Duration     `json:"totalDurationNs"`
	QueryResults   []QueryResult     `json:"queryResults"`
	ConnectionInfo database.ConnectionInfo `json:"connectionInfo"`
	MetricsHistory []database.DBMetrics  `json:"metricsHistory,omitempty"`
	Summary        ResultSummary     `json:"summary"`
}

// ResultSummary provides aggregate statistics for the test
type ResultSummary struct {
	TotalQueries           int               `json:"totalQueries"`
	SuccessfulQueries      int               `json:"successfulQueries"`
	FailedQueries          int               `json:"failedQueries"`
	TotalExecutions        int               `json:"totalExecutions"`
	SuccessfulExecutions   int               `json:"successfulExecutions"`
	FailedExecutions       int               `json:"failedExecutions"`
	AvgDurationMs          float64           `json:"avgDurationMs"`
	MedianDurationMs       float64           `json:"medianDurationMs"`
	StdDevDurationMs       float64           `json:"stdDevDurationMs"`
	MaxDurationMs          float64           `json:"maxDurationMs"`
	P95DurationMs          float64           `json:"p95DurationMs"`
	P99DurationMs          float64           `json:"p99DurationMs"`
	TotalRowsReturned      int64             `json:"totalRowsReturned"`
	QueriesByComplexity    map[string]int    `json:"queriesByComplexity"`
	ErrorsByType           map[string]int    `json:"errorsByType"`
}

// ComparisonResult represents a comparison between two test runs
type ComparisonResult struct {
	Before              TestResult      `json:"before"`
	After               TestResult      `json:"after"`
	ImprovementSummary  ImprovementStats `json:"improvementSummary"`
	QueryComparisons    []QueryComparison `json:"queryComparisons"`
	ErrorsReduced       map[string]int   `json:"errorsReduced"`
}

// ImprovementStats holds performance improvement statistics
type ImprovementStats struct {
	AvgTimeImprovement      float64 `json:"avgTimeImprovement"`
	MedianTimeImprovement   float64 `json:"medianTimeImprovement"`
	P95TimeImprovement      float64 `json:"p95TimeImprovement"`
	MaxTimeImprovement      float64 `json:"maxTimeImprovement"`
	ErrorReduction          float64 `json:"errorReduction"`
	SuccessRateImprovement  float64 `json:"successRateImprovement"`
}

// QueryComparison compares before/after metrics for a single query
type QueryComparison struct {
	Name                string  `json:"name"`
	BeforeAvgMs         float64 `json:"beforeAvgMs"`
	AfterAvgMs          float64 `json:"afterAvgMs"`
	ImprovementPercent  float64 `json:"improvementPercent"`
	BeforeErrors        int     `json:"beforeErrors"`
	AfterErrors         int     `json:"afterErrors"`
	BeforeRows          int64   `json:"beforeRows"`
	AfterRows           int64   `json:"afterRows"`
}
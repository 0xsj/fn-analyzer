// cmd/analyzer/main.go
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/0xsj/fn-analyzer/internal/analyzer"
	"github.com/0xsj/fn-analyzer/internal/config"
	"github.com/0xsj/fn-analyzer/internal/database"
)

var (
	Version = "1.0.0"
)

func main() {
	start := time.Now()

	configFile := flag.String("config", "config.json", "Path to config file")
	queriesFile := flag.String("queries", "", "Path to queries file (overrides config)")
	outputDir := flag.String("output", "", "Output directory (overrides config)")
	label := flag.String("label", "", "Test run label (overrides config)")
	verbose := flag.Bool("verbose", false, "Verbose output")
	testConnection := flag.Bool("test-connection", false, "Test database connection only")
	versionFlag := flag.Bool("version", false, "Print version and exit")
	flag.Parse()

	if *versionFlag {
		fmt.Printf("DB Analyzer v%s\n", Version)
		return
	}

	cfg, err := config.LoadConfig(*configFile)
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	if *queriesFile != "" {
		cfg.QueriesFile = *queriesFile
	}
	if *outputDir != "" {
		cfg.OutputDir = *outputDir
	}
	if *label != "" {
		cfg.Label = *label
	}
	if *verbose {
		cfg.Verbose = true
	}

	if *testConnection {
		if err := database.TestConnection(cfg.DSN); err != nil {
			log.Fatalf("Connection test failed: %v", err)
		}
		return
	}

	if err := os.MkdirAll(cfg.OutputDir, 0755); err != nil {
		log.Fatalf("Error creating output directory: %v", err)
	}

	queries, err := analyzer.LoadQueries(cfg.QueriesFile)
	if err != nil {
		log.Fatalf("Error loading queries: %v", err)
	}

	log.Printf("Loaded %d queries from %s", len(queries), cfg.QueriesFile)

	db, err := database.Connect(cfg.DSN, cfg.Concurrency)
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}
	defer db.Close()

	if err := analyzer.WarmupConnectionPool(db, cfg.WarmupIterations); err != nil {
		log.Fatalf("Error during warmup: %v", err)
	}

	connInfo, err := database.GetConnectionInfo(db)
	if err != nil {
		log.Printf("Warning: couldn't get complete connection info: %v", err)
	}

	log.Printf("Starting performance test with %d queries, %d iterations each, concurrency %d",
		len(queries), cfg.Iterations, cfg.Concurrency)

	a := analyzer.NewAnalyzer(db, queries, *cfg)

	results, err := a.Run()
	if err != nil {
		log.Fatalf("Error during test: %v", err)
	}

	err = analyzer.GenerateReports(results, connInfo, *cfg, time.Since(start))
	if err != nil {
		log.Fatalf("Error generating reports: %v", err)
	}

	log.Printf("Test completed in %v", time.Since(start))
}
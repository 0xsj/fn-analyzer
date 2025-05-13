// internal/database/connection.go
package database

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func Connect(dsn string, concurrency int) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("error connecting to database: %w", err)
	}

	db.SetMaxOpenConns(concurrency * 2) 
	db.SetMaxIdleConns(concurrency)
	db.SetConnMaxLifetime(time.Minute * 5)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("error pinging database: %w", err)
	}

	return db, nil
}

func TestConnection(dsn string) error {
	log.Println("Testing database connection...")
	
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("error opening database connection: %w", err)
	}
	defer db.Close()
	
	startTime := time.Now()
	if err := db.Ping(); err != nil {
		return fmt.Errorf("error connecting to database: %w", err)
	}
	pingTime := time.Since(startTime)
	
	log.Printf("✓ Database connection successful! (Ping time: %v)", pingTime)
	
	var version string
	if err := db.QueryRow("SELECT VERSION()").Scan(&version); err != nil {
		log.Printf("Warning: Could not get database version: %v", err)
	} else {
		log.Printf("✓ Connected to MySQL server version: %s", version)
	}
	
	info, err := GetConnectionInfo(db)
	if err != nil {
		log.Printf("Warning: Could not get detailed connection info: %v", err)
	} else {
		log.Printf("✓ Database statistics:")
		log.Printf("  - Threads running: %d", info.ThreadsRunning)
		log.Printf("  - Threads connected: %d", info.ThreadsConnected)
		log.Printf("  - Open tables: %d", info.OpenTables)
		log.Printf("  - Slow queries: %d", info.SlowQueries)
		log.Printf("  - Uptime: %d seconds", info.Uptime)
		log.Printf("  - Questions per second: %.2f", info.QuestionsPerSec)
	}
	
	startTime = time.Now()
	rows, err := db.Query("SELECT 1")
	if err != nil {
		log.Printf("Warning: Simple query test failed: %v", err)
	} else {
		rows.Close()
		queryTime := time.Since(startTime)
		log.Printf("✓ Simple query test successful! (Query time: %v)", queryTime)
	}
	
	startTime = time.Now()
	rows, err = db.Query("SHOW TABLES")
	if err != nil {
		log.Printf("Warning: Unable to list tables: %v", err)
	} else {
		var tableCount int
		for rows.Next() {
			tableCount++
		}
		rows.Close()
		queryTime := time.Since(startTime)
		log.Printf("✓ Found %d tables in the database (Query time: %v)", tableCount, queryTime)
	}
	
	return nil
}

type ConnectionInfo struct {
	Version          string  `json:"version"`
	ThreadsRunning   int     `json:"threadsRunning"`
	ThreadsConnected int     `json:"threadsConnected"`
	OpenTables       int     `json:"openTables"`
	SlowQueries      int     `json:"slowQueries"`
	Uptime           int     `json:"uptimeSeconds"`
	QuestionsPerSec  float64 `json:"questionsPerSecond"`
}

func GetConnectionInfo(db *sql.DB) (ConnectionInfo, error) {
	info := ConnectionInfo{}

	var version string
	if err := db.QueryRow("SELECT VERSION()").Scan(&version); err != nil {
		return info, err
	}
	info.Version = version

	rows, err := db.Query("SHOW GLOBAL STATUS WHERE Variable_name IN ('Threads_running', 'Threads_connected', 'Open_tables', 'Slow_queries', 'Uptime', 'Questions')")
	if err != nil {
		return info, err
	}
	defer rows.Close()

	var uptime, questions int
	for rows.Next() {
		var name, value string
		if err := rows.Scan(&name, &value); err != nil {
			return info, err
		}

		switch name {
		case "Threads_running":
			fmt.Sscanf(value, "%d", &info.ThreadsRunning)
		case "Threads_connected":
			fmt.Sscanf(value, "%d", &info.ThreadsConnected)
		case "Open_tables":
			fmt.Sscanf(value, "%d", &info.OpenTables)
		case "Slow_queries":
			fmt.Sscanf(value, "%d", &info.SlowQueries)
		case "Uptime":
			fmt.Sscanf(value, "%d", &uptime)
			info.Uptime = uptime
		case "Questions":
			fmt.Sscanf(value, "%d", &questions)
		}
	}

	if uptime > 0 {
		info.QuestionsPerSec = float64(questions) / float64(uptime)
	}

	return info, nil
}
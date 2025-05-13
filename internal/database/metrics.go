// internal/database/metrics.go
package database

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"
)

type DBMetrics struct {
	ThreadsRunning         int     `json:"threadsRunning"`
	ThreadsConnected       int     `json:"threadsConnected"`
	ThreadsCreated         int     `json:"threadsCreated"`
	OpenTables             int     `json:"openTables"`
	OpenFiles              int     `json:"openFiles"`
	SlowQueries            int     `json:"slowQueries"`
	InnodbRowsRead         int64   `json:"innodbRowsRead"`
	InnodbRowsInserted     int64   `json:"innodbRowsInserted"`
	InnodbRowsUpdated      int64   `json:"innodbRowsUpdated"`
	InnodbRowsDeleted      int64   `json:"innodbRowsDeleted"`
	QPS                    float64 `json:"queriesPerSecond"`
	LockTimeAvg            float64 `json:"avgLockTimeMs"`
	TableCacheHitRate      float64 `json:"tableCacheHitRate"`
	BufferPoolHitRate      float64 `json:"bufferPoolHitRate"`
	DeadlocksTotal         int     `json:"deadlocksTotal"`
	ActiveTransactions     int     `json:"activeTransactions"`
	MemoryUsedBytes        int64   `json:"memoryUsedBytes"`
	LongRunningTransCount  int     `json:"longRunningTransactions"`
	InnodbHistoryListLen   int     `json:"innodbHistoryListLength"`
	InnodbBufferPoolStatus string  `json:"innodbBufferPoolStatus"`
}

func GetDetailedMetrics(db *sql.DB) (DBMetrics, error) {
	metrics := DBMetrics{}

	rows, err := db.Query("SHOW GLOBAL STATUS")
	if err != nil {
		return metrics, fmt.Errorf("error getting global status: %w", err)
	}
	defer rows.Close()

	statusVars := make(map[string]string)
	for rows.Next() {
		var name, value string
		if err := rows.Scan(&name, &value); err != nil {
			return metrics, err
		}
		statusVars[name] = value
	}

	parseIntVar(&metrics.ThreadsRunning, statusVars, "Threads_running")
	parseIntVar(&metrics.ThreadsConnected, statusVars, "Threads_connected")
	parseIntVar(&metrics.ThreadsCreated, statusVars, "Threads_created")
	parseIntVar(&metrics.OpenTables, statusVars, "Open_tables")
	parseIntVar(&metrics.OpenFiles, statusVars, "Open_files")
	parseIntVar(&metrics.SlowQueries, statusVars, "Slow_queries")
	parseIntVar64(&metrics.InnodbRowsRead, statusVars, "Innodb_rows_read")
	parseIntVar64(&metrics.InnodbRowsInserted, statusVars, "Innodb_rows_inserted")
	parseIntVar64(&metrics.InnodbRowsUpdated, statusVars, "Innodb_rows_updated")
	parseIntVar64(&metrics.InnodbRowsDeleted, statusVars, "Innodb_rows_deleted")
	parseIntVar(&metrics.DeadlocksTotal, statusVars, "Innodb_deadlocks")

	if openTableDefs, ok := statusVars["Opened_table_definitions"]; ok {
		if tableOpenCache, ok := statusVars["Table_open_cache"]; ok {
			var opened, cache int
			fmt.Sscanf(openTableDefs, "%d", &opened)
			fmt.Sscanf(tableOpenCache, "%d", &cache)
			if opened > 0 {
				metrics.TableCacheHitRate = 100.0 - (float64(opened) / float64(opened+cache) * 100.0)
			}
		}
	}

	if readRequests, ok := statusVars["Innodb_buffer_pool_read_requests"]; ok {
		if reads, ok := statusVars["Innodb_buffer_pool_reads"]; ok {
			var requests, diskReads int64
			fmt.Sscanf(readRequests, "%d", &requests)
			fmt.Sscanf(reads, "%d", &diskReads)
			if requests > 0 {
				metrics.BufferPoolHitRate = (1.0 - float64(diskReads)/float64(requests)) * 100.0
			}
		}
	}

	parseIntVar(&metrics.InnodbHistoryListLen, statusVars, "Innodb_history_list_length")

	if uptime, ok := statusVars["Uptime"]; ok {
		if questions, ok := statusVars["Questions"]; ok {
			var up, q int
			fmt.Sscanf(uptime, "%d", &up)
			fmt.Sscanf(questions, "%d", &q)
			if up > 0 {
				metrics.QPS = float64(q) / float64(up)
			}
		}
	}

	parseIntVar64(&metrics.MemoryUsedBytes, statusVars, "Global_memory_used")

	var activeTrans int
	err = db.QueryRow("SELECT COUNT(*) FROM information_schema.innodb_trx").Scan(&activeTrans)
	if err == nil {
		metrics.ActiveTransactions = activeTrans
	}

	var longTrans int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM information_schema.innodb_trx 
		WHERE trx_started < NOW() - INTERVAL 10 SECOND
	`).Scan(&longTrans)
	if err == nil {
		metrics.LongRunningTransCount = longTrans
	}

	var bufferPoolStatus string
	err = db.QueryRow("SHOW ENGINE INNODB STATUS").Scan(&bufferPoolStatus)
	if err == nil {
		if idx := strings.Index(bufferPoolStatus, "BUFFER POOL AND MEMORY"); idx >= 0 {
			endIdx := strings.Index(bufferPoolStatus[idx:], "---")
			if endIdx > 0 {
				metrics.InnodbBufferPoolStatus = bufferPoolStatus[idx : idx+endIdx]
			}
		}
	}

	return metrics, nil
}

func RunMetricsCollector(db *sql.DB, interval time.Duration, metricsCallback func(DBMetrics)) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			metrics, err := GetDetailedMetrics(db)
			if err != nil {
				log.Printf("Error collecting metrics: %v", err)
				continue
			}

			metricsCallback(metrics)
		}
	}()
}

func MonitorDeadlocks(db *sql.DB, callback func(string)) error {
	var enabled string
	err := db.QueryRow("SELECT @@event_scheduler").Scan(&enabled)
	if err != nil {
		return fmt.Errorf("error checking event scheduler: %w", err)
	}

	if enabled != "ON" {
		log.Println("Warning: MySQL event scheduler is not enabled, deadlock monitoring will be limited")
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS deadlock_monitor (
			id INT AUTO_INCREMENT PRIMARY KEY,
			detected_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			deadlock_info TEXT,
			is_processed BOOLEAN DEFAULT FALSE
		)
	`)
	if err != nil {
		return fmt.Errorf("error creating deadlock monitor table: %w", err)
	}

	_, err = db.Exec(`
		CREATE EVENT IF NOT EXISTS capture_deadlocks
		ON SCHEDULE EVERY 10 SECOND
		DO
		BEGIN
			DECLARE deadlocks_before INT;
			DECLARE deadlocks_after INT;
			DECLARE deadlock_info TEXT;
			
			SELECT VARIABLE_VALUE INTO deadlocks_before FROM performance_schema.global_status WHERE VARIABLE_NAME = 'Innodb_deadlocks';
			DO SLEEP(1);
			SELECT VARIABLE_VALUE INTO deadlocks_after FROM performance_schema.global_status WHERE VARIABLE_NAME = 'Innodb_deadlocks';
			
			IF deadlocks_after > deadlocks_before THEN
				SELECT SUBSTRING(event_data, LOCATE('TRANSACTION', event_data)) INTO deadlock_info
				FROM performance_schema.events_statements_history
				WHERE EVENT_NAME = 'statement/sql/kill'
				ORDER BY timer_end DESC LIMIT 1;
				
				IF deadlock_info IS NOT NULL THEN
					INSERT INTO deadlock_monitor (deadlock_info) VALUES (deadlock_info);
				ELSE
					INSERT INTO deadlock_monitor (deadlock_info) VALUES ('Deadlock detected but details not available');
				END IF;
			END IF;
		END;
	`)
	if err != nil {
		log.Printf("Warning: Could not create deadlock monitor event: %v", err)
	}

	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			rows, err := db.Query(`
				SELECT id, detected_at, deadlock_info 
				FROM deadlock_monitor 
				WHERE is_processed = FALSE
				ORDER BY detected_at DESC
			`)
			if err != nil {
				log.Printf("Error checking for deadlocks: %v", err)
				continue
			}

			for rows.Next() {
				var id int
				var detectedAt time.Time
				var info string
				if err := rows.Scan(&id, &detectedAt, &info); err != nil {
					log.Printf("Error scanning deadlock info: %v", err)
					continue
				}

				_, err = db.Exec("UPDATE deadlock_monitor SET is_processed = TRUE WHERE id = ?", id)
				if err != nil {
					log.Printf("Error marking deadlock as processed: %v", err)
				}

				deadlockInfo := fmt.Sprintf("DEADLOCK DETECTED at %s:\n%s", detectedAt.Format(time.RFC3339), info)
				callback(deadlockInfo)
			}
			rows.Close()
		}
	}()

	return nil
}

func parseIntVar(target *int, vars map[string]string, key string) {
	if val, ok := vars[key]; ok {
		fmt.Sscanf(val, "%d", target)
	}
}

func parseIntVar64(target *int64, vars map[string]string, key string) {
	if val, ok := vars[key]; ok {
		fmt.Sscanf(val, "%d", target)
	}
}

// func parseFloatVar(target *float64, vars map[string]string, key string) {
// 	if val, ok := vars[key]; ok {
// 		fmt.Sscanf(val, "%f", target)
// 	}
// }

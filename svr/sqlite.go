// db/sqlite.go
package svr

import (
	"database/sql"
	"log"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3" // SQLite driver
)

func InitDB( dataSourceName string)  (DB *sql.DB, err error) {
	// Ensure the directory for the SQLite file exists (if it's in a subdirectory)
	// For this example, we'll assume it's in the current directory.

	// Check if the database file exists. If not, os.Create will make it.
	// sql.Open will also create it if it doesn't exist, but this is explicit.
	if _, err := os.Stat(dataSourceName); os.IsNotExist(err) {
		log.Printf("Database file %s does not exist, will be created.", dataSourceName)
	}

	DB, err = sql.Open("sqlite3", dataSourceName)
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}

	// Recommended for SQLite to improve concurrency and prevent "database is locked" errors
	// WAL mode allows one writer and multiple readers to operate concurrently.
	_, err = DB.Exec("PRAGMA journal_mode=WAL;")
	if err != nil {
		log.Fatalf("Failed to set WAL mode: %v", err)
	}

	// SQLite typically performs best with a single writer.
	// Setting MaxOpenConns to 1 serializes write access through the pool.
	// Reads can still be concurrent if WAL mode is enabled.
	DB.SetMaxOpenConns(32)
	DB.SetMaxIdleConns(8) // Usually same as MaxOpenConns for SQLite
	DB.SetConnMaxLifetime(time.Minute * 5)

	// Execute PRAGMA statements
	// Note: The order can matter for some PRAGMAs.
	// `page_size` should ideally be set on an empty database or before any tables are created.
	// If the database already exists with data and a different page_size, this PRAGMA might be ignored
	// or require a VACUUM to take effect.

	pragmas := []string{
		"PRAGMA journal_mode = WAL;",   // Already set via DSN for go-sqlite3, but can be here too
		"PRAGMA synchronous = NORMAL;", // Or OFF, if you dare (and understand the risks)
		"PRAGMA busy_timeout = 40000;", // Already set via DSN
		"PRAGMA cache_size = -200001;", // Approx 200MB (negative value is KiB for cache_size)
		"PRAGMA temp_store = MEMORY;",
		"PRAGMA default_transaction_mode = IMMEDIATE;", // Go's sql package might override this per transaction
		"PRAGMA logging_mode = OFF;",

		// Optional
		"PRAGMA foreign_keys = OFF;",        // Be careful with this; usually ON is safer for data integrity
		"PRAGMA mmap_size = 268435456;",     // 256MB, test carefully for stability and performance
		"PRAGMA wal_autocheckpoint = 4000;", // In pages, default is 1000. So 4000 * page_size
		"PRAGMA page_size = 8192;",          // CRITICAL: Must be set on an EMPTY database or before any data.
		// If the DB exists, this will likely be ignored or error unless the DB is vacuumed.
		// It's safer to set this when the DB is first created.
		// For an existing DB, you'd typically need to:
		// 1. PRAGMA page_size=8192;
		// 2. VACUUM;
		// This can be a long operation.
	}

	// Special handling for page_size if the database is new or you intend to VACUUM
	// For a *new* database, set page_size *before* createTable.
	// If the database `dataSourceName` does not exist yet, `sql.Open` followed by an Exec
	// will create it. `page_size` should be one of the first PRAGMAs.
	// We'll try to set it. If the DB exists and has a different page_size, this PRAGMA
	// will be a no-op unless the DB is empty or vacuumed.
	// if _, err := DB.Exec("PRAGMA page_size = 8192;"); err != nil {
	// 	// This might error if the database is not empty and has a different page size
	// 	log.Printf("Warning/Error setting page_size: %v. This is OK if DB already exists with a different page size and is not empty.", err)
	// } else {
	// 	log.Println("Attempted to set PRAGMA page_size = 8192.")
	// }

	log.Println("Executing PRAGMA settings...")
	for _, pragma := range pragmas {
		_, err = DB.Exec(pragma)
		if err != nil {
			// Log as a warning for most pragmas, but could be fatal for critical ones
			log.Printf("Warning/Error executing PRAGMA '%s': %v", pragma, err)
		} else {
			log.Printf("Successfully executed PRAGMA '%s'", pragma)
		}
	}

	if err = DB.Ping(); err != nil {
		log.Fatalf("Error pinging database: %v", err)
	}

	log.Println("Database connection established and WAL mode enabled.")

	// 1. 删除旧表（如果存在）
	_, err = DB.Exec("DROP TABLE IF EXISTS users")
	if err != nil {
		log.Fatalf("Error dropping users table: %v", err)
	}

	// 2. 创建新表
	createUserTableSQL := `
		CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL COLLATE NOCASE,
			age INTEGER NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT NULL,
			deleted_at DATETIME DEFAULT NULL
		)
	 `
	_, err = DB.Exec(createUserTableSQL)
	if err != nil {
		log.Fatalf("Error creating user table: %v", err)
	}

	// 3. 创建索引（单独执行）
	_, err = DB.Exec(`
		 CREATE INDEX user_deleted_at_age_name_id_1747242058824 ON users (deleted_at, age, name, id)
	 `)
	if err != nil {
		log.Fatalf("Error creating index: %v", err)
	}

	log.Println("users table checked/created successfully.")


	return DB,nil
}

// Package db provides database connectivity and operations.
package db

import (
	"database/sql"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

const (
	helmDBDir  = ".helm"
	helmDBFile = "helm.db"
)

// DB wraps the sqlc-generated querier with migration management.
type DB struct {
	*Queries
	sqlDB *sql.DB
	path  string
}

// Open opens or creates the SQLite database at the given project path.
// If projectPath is empty, it uses the global ~/.helm/helm.db.
func Open(projectPath string) (*DB, error) {
	var dbPath string

	if projectPath != "" {
		dir := filepath.Join(projectPath, helmDBDir)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create project db dir: %w", err)
		}
		dbPath = filepath.Join(dir, helmDBFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("get home dir: %w", err)
		}
		dir := filepath.Join(home, helmDBDir)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create global db dir: %w", err)
		}
		dbPath = filepath.Join(dir, helmDBFile)
	}

	sqlDB, err := sql.Open("sqlite", dbPath+"?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	// Enable WAL mode
	if _, err := sqlDB.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return nil, fmt.Errorf("enable WAL: %w", err)
	}

	db := &DB{
		Queries: New(sqlDB),
		sqlDB:   sqlDB,
		path:    dbPath,
	}

	if err := db.migrate(); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return db, nil
}

// Path returns the database file path.
func (db *DB) Path() string {
	return db.path
}

// Close closes the underlying database connection.
func (db *DB) Close() error {
	return db.sqlDB.Close()
}

// migrate applies all pending migrations in order.
func (db *DB) migrate() error {
	// Create migrations tracking table
	if _, err := db.sqlDB.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY,
		applied_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
	)`); err != nil {
		return fmt.Errorf("create migrations table: %w", err)
	}

	// Get applied versions
	rows, err := db.sqlDB.Query("SELECT version FROM schema_migrations ORDER BY version")
	if err != nil {
		return fmt.Errorf("query applied migrations: %w", err)
	}
	defer rows.Close()

	applied := make(map[int]bool)
	for rows.Next() {
		var v int
		if err := rows.Scan(&v); err != nil {
			return fmt.Errorf("scan version: %w", err)
		}
		applied[v] = true
	}

	// Read migration files
	entries, err := migrationFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	// Sort by version number extracted from filename (e.g., 001_initial.sql -> 1)
	type migFile struct {
		version int
		name    string
	}
	var files []migFile
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".sql") || strings.Contains(name, ".down.") {
			continue
		}
		parts := strings.SplitN(name, "_", 2)
		if len(parts) < 2 {
			continue
		}
		ver, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}
		files = append(files, migFile{version: ver, name: name})
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].version < files[j].version
	})

	// Apply pending migrations
	for _, f := range files {
		if applied[f.version] {
			continue
		}

		content, err := migrationFS.ReadFile("migrations/" + f.name)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", f.name, err)
		}

		// Execute migration in transaction
		tx, err := db.sqlDB.Begin()
		if err != nil {
			return fmt.Errorf("begin migration tx: %w", err)
		}

		if _, err := tx.Exec(string(content)); err != nil {
			tx.Rollback()
			return fmt.Errorf("apply migration %s: %w", f.name, err)
		}

		if _, err := tx.Exec("INSERT INTO schema_migrations (version) VALUES (?)", f.version); err != nil {
			tx.Rollback()
			return fmt.Errorf("record migration %s: %w", f.name, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %s: %w", f.name, err)
		}
	}

	return nil
}

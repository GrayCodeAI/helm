// Package db provides database migration capabilities.
package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/yourname/helm/internal/errors"
	"github.com/yourname/helm/internal/logger"
)

// Migration represents a database migration
type Migration struct {
	Version   int64
	Name      string
	UpSQL     string
	DownSQL   string
	AppliedAt *time.Time
}

// Migrator handles database migrations
type Migrator struct {
	db     *sql.DB
	logger *logger.Logger
	table  string
}

// NewMigrator creates a new migrator
func NewMigrator(db *sql.DB, log *logger.Logger) *Migrator {
	if log == nil {
		log = logger.GetDefault()
	}
	return &Migrator{
		db:     db,
		logger: log,
		table:  "schema_migrations",
	}
}

// SetTable sets the migrations table name
func (m *Migrator) SetTable(name string) {
	m.table = name
}

// Init creates the migrations table
func (m *Migrator) Init(ctx context.Context) error {
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			version INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`, m.table)

	if _, err := m.db.ExecContext(ctx, query); err != nil {
		return errors.Wrap(err, errors.CodeDatabase, "create migrations table")
	}

	m.logger.Info("Migrations table initialized")
	return nil
}

// Status returns current migration status
func (m *Migrator) Status(ctx context.Context) (*MigrationStatus, error) {
	if err := m.Init(ctx); err != nil {
		return nil, err
	}

	// Get current version
	var currentVersion int64
	err := m.db.QueryRowContext(ctx,
		fmt.Sprintf("SELECT COALESCE(MAX(version), 0) FROM %s", m.table),
	).Scan(&currentVersion)
	if err != nil {
		return nil, errors.Wrap(err, errors.CodeDatabase, "get current version")
	}

	// Get applied migrations
	rows, err := m.db.QueryContext(ctx,
		fmt.Sprintf("SELECT version, name, applied_at FROM %s ORDER BY version", m.table),
	)
	if err != nil {
		return nil, errors.Wrap(err, errors.CodeDatabase, "get applied migrations")
	}
	defer rows.Close()

	var applied []AppliedMigration
	for rows.Next() {
		var am AppliedMigration
		if err := rows.Scan(&am.Version, &am.Name, &am.AppliedAt); err != nil {
			continue
		}
		applied = append(applied, am)
	}

	return &MigrationStatus{
		CurrentVersion: currentVersion,
		AppliedCount:   len(applied),
		Applied:        applied,
	}, nil
}

// Up runs all pending migrations
func (m *Migrator) Up(ctx context.Context, migrations []Migration) error {
	if err := m.Init(ctx); err != nil {
		return err
	}

	// Get current version
	status, err := m.Status(ctx)
	if err != nil {
		return err
	}

	// Filter and sort pending migrations
	var pending []Migration
	for _, mig := range migrations {
		if mig.Version > status.CurrentVersion {
			pending = append(pending, mig)
		}
	}

	sort.Slice(pending, func(i, j int) bool {
		return pending[i].Version < pending[j].Version
	})

	if len(pending) == 0 {
		m.logger.Info("No pending migrations")
		return nil
	}

	// Run migrations
	for _, mig := range pending {
		m.logger.Info("Applying migration %d: %s", mig.Version, mig.Name)

		if err := m.applyMigration(ctx, mig); err != nil {
			return errors.Wrapf(err, errors.CodeDatabase, "apply migration %d", mig.Version)
		}

		m.logger.Info("Migration %d applied successfully", mig.Version)
	}

	return nil
}

// Down rolls back migrations
func (m *Migrator) Down(ctx context.Context, migrations []Migration, steps int) error {
	if err := m.Init(ctx); err != nil {
		return err
	}

	// Get applied migrations
	status, err := m.Status(ctx)
	if err != nil {
		return err
	}

	if status.AppliedCount == 0 {
		m.logger.Info("No migrations to rollback")
		return nil
	}

	// Find migrations to rollback
	var toRollback []Migration
	for i := len(status.Applied) - 1; i >= 0 && len(toRollback) < steps; i-- {
		for _, mig := range migrations {
			if mig.Version == status.Applied[i].Version {
				toRollback = append(toRollback, mig)
				break
			}
		}
	}

	// Rollback
	for _, mig := range toRollback {
		m.logger.Info("Rolling back migration %d: %s", mig.Version, mig.Name)

		if err := m.rollbackMigration(ctx, mig); err != nil {
			return errors.Wrapf(err, errors.CodeDatabase, "rollback migration %d", mig.Version)
		}

		m.logger.Info("Migration %d rolled back successfully", mig.Version)
	}

	return nil
}

// applyMigration applies a single migration
func (m *Migrator) applyMigration(ctx context.Context, mig Migration) error {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Execute migration
	if _, err := tx.ExecContext(ctx, mig.UpSQL); err != nil {
		return fmt.Errorf("execute migration: %w", err)
	}

	// Record migration
	_, err = tx.ExecContext(ctx,
		fmt.Sprintf("INSERT INTO %s (version, name) VALUES (?, ?)", m.table),
		mig.Version, mig.Name,
	)
	if err != nil {
		return fmt.Errorf("record migration: %w", err)
	}

	return tx.Commit()
}

// rollbackMigration rolls back a single migration
func (m *Migrator) rollbackMigration(ctx context.Context, mig Migration) error {
	if mig.DownSQL == "" {
		return fmt.Errorf("migration %d has no down SQL", mig.Version)
	}

	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Execute rollback
	if _, err := tx.ExecContext(ctx, mig.DownSQL); err != nil {
		return fmt.Errorf("execute rollback: %w", err)
	}

	// Remove migration record
	_, err = tx.ExecContext(ctx,
		fmt.Sprintf("DELETE FROM %s WHERE version = ?", m.table),
		mig.Version,
	)
	if err != nil {
		return fmt.Errorf("remove migration record: %w", err)
	}

	return tx.Commit()
}

// MigrationStatus represents migration status
type MigrationStatus struct {
	CurrentVersion int64
	AppliedCount   int
	Applied        []AppliedMigration
}

// AppliedMigration represents an applied migration
type AppliedMigration struct {
	Version   int64
	Name      string
	AppliedAt time.Time
}

// LoadMigrationsFromFS loads migrations from embedded filesystem
func LoadMigrationsFromFS(fs embed.FS, dir string) ([]Migration, error) {
	entries, err := fs.ReadDir(dir)
	if err != nil {
		return nil, errors.Wrap(err, errors.CodeInternal, "read migrations directory")
	}

	var migrations []Migration
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".sql") {
			continue
		}

		// Parse version from filename (e.g., "001_initial.up.sql")
		parts := strings.Split(name, "_")
		if len(parts) < 2 {
			continue
		}

		version, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			continue
		}

		// Read file content
		content, err := fs.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return nil, errors.Wrapf(err, errors.CodeInternal, "read migration file %s", name)
		}

		// Determine if up or down
		var mig *Migration
		for i, m := range migrations {
			if m.Version == version {
				mig = &migrations[i]
				break
			}
		}

		if mig == nil {
			migrations = append(migrations, Migration{
				Version: version,
				Name:    strings.Join(parts[1:len(parts)-1], "_"),
			})
			mig = &migrations[len(migrations)-1]
		}

		if strings.Contains(name, ".down.") {
			mig.DownSQL = string(content)
		} else {
			mig.UpSQL = string(content)
		}
	}

	// Sort by version
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

// LoadMigrationsFromStrings loads migrations from SQL strings
func LoadMigrationsFromStrings(migrationMap map[int64]struct{ Up, Down string }) []Migration {
	var migrations []Migration
	for version, sql := range migrationMap {
		migrations = append(migrations, Migration{
			Version: version,
			UpSQL:   sql.Up,
			DownSQL: sql.Down,
		})
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations
}

// CreateMigration creates a new migration file
func CreateMigration(dir, name string, version int64) (string, string, error) {
	baseName := fmt.Sprintf("%03d_%s", version, name)
	upFile := filepath.Join(dir, baseName+".up.sql")
	downFile := filepath.Join(dir, baseName+".down.sql")

	// Create directory
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", "", errors.Wrap(err, errors.CodeInternal, "create migrations directory")
	}

	// Create up file
	upContent := fmt.Sprintf("-- Migration: %s\n-- Version: %d\n\n", name, version)
	if err := os.WriteFile(upFile, []byte(upContent), 0644); err != nil {
		return "", "", errors.Wrap(err, errors.CodeInternal, "create up migration file")
	}

	// Create down file
	downContent := fmt.Sprintf("-- Rollback: %s\n-- Version: %d\n\n", name, version)
	if err := os.WriteFile(downFile, []byte(downContent), 0644); err != nil {
		return "", "", errors.Wrap(err, errors.CodeInternal, "create down migration file")
	}

	return upFile, downFile, nil
}

// CLI helper functions

// PrintStatus prints migration status
func (m *Migrator) PrintStatus(ctx context.Context) error {
	status, err := m.Status(ctx)
	if err != nil {
		return err
	}

	fmt.Printf("Current Version: %d\n", status.CurrentVersion)
	fmt.Printf("Applied Migrations: %d\n\n", status.AppliedCount)

	if len(status.Applied) > 0 {
		fmt.Println("Applied Migrations:")
		for _, mig := range status.Applied {
			fmt.Printf("  %d - %s (applied: %s)\n",
				mig.Version, mig.Name, mig.AppliedAt.Format("2006-01-02 15:04:05"))
		}
	}

	return nil
}

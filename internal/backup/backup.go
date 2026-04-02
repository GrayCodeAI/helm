// Package backup provides database backup and restore capabilities.
package backup

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/yourname/helm/internal/errors"
	"github.com/yourname/helm/internal/logger"
)

// Manager handles backup and restore operations
type Manager struct {
	db     *sql.DB
	logger *logger.Logger
}

// NewManager creates a new backup manager
func NewManager(db *sql.DB, log *logger.Logger) *Manager {
	if log == nil {
		log = logger.GetDefault()
	}
	return &Manager{db: db, logger: log}
}

// Backup creates a backup of the database
func (m *Manager) Backup(ctx context.Context, destPath string) error {
	m.logger.Info("Starting database backup to %s", destPath)
	start := time.Now()

	// Ensure destination directory exists
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return errors.Wrap(err, errors.CodeInternal, "create backup directory")
	}

	// Create backup file
	file, err := os.Create(destPath)
	if err != nil {
		return errors.Wrap(err, errors.CodeInternal, "create backup file")
	}
	defer file.Close()

	// Create gzip writer
	gzWriter := gzip.NewWriter(file)
	defer gzWriter.Close()

	// Create tar writer
	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	// Backup main database
	if err := m.backupDatabase(ctx, tarWriter, "main"); err != nil {
		return errors.Wrap(err, errors.CodeDatabase, "backup database")
	}

	duration := time.Since(start)
	m.logger.Info("Backup completed in %s: %s", duration, destPath)

	return nil
}

// backupDatabase backs up a single database
func (m *Manager) backupDatabase(ctx context.Context, tw *tar.Writer, name string) error {
	// Get database path
	var dbPath string
	err := m.db.QueryRowContext(ctx, "PRAGMA database_list").Scan(&name, &dbPath, &name)
	if err != nil {
		// Just get the path differently for SQLite
		var seq int
		err = m.db.QueryRowContext(ctx, "PRAGMA database_list").Scan(&seq, &name, &dbPath)
		if err != nil {
			return err
		}
	}

	// Open database file
	file, err := os.Open(dbPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Get file info
	stat, err := file.Stat()
	if err != nil {
		return err
	}

	// Create tar header
	header := &tar.Header{
		Name:    filepath.Base(dbPath),
		Size:    stat.Size(),
		Mode:    int64(stat.Mode()),
		ModTime: stat.ModTime(),
	}

	if err := tw.WriteHeader(header); err != nil {
		return err
	}

	// Copy file content
	if _, err := io.Copy(tw, file); err != nil {
		return err
	}

	return nil
}

// Restore restores a database from backup
func (m *Manager) Restore(ctx context.Context, backupPath string, destPath string) error {
	m.logger.Info("Starting database restore from %s to %s", backupPath, destPath)
	start := time.Now()

	// Open backup file
	file, err := os.Open(backupPath)
	if err != nil {
		return errors.Wrap(err, errors.CodeInternal, "open backup file")
	}
	defer file.Close()

	// Create gzip reader
	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return errors.Wrap(err, errors.CodeInternal, "create gzip reader")
	}
	defer gzReader.Close()

	// Create tar reader
	tr := tar.NewReader(gzReader)

	// Ensure destination directory exists
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return errors.Wrap(err, errors.CodeInternal, "create destination directory")
	}

	// Extract files
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return errors.Wrap(err, errors.CodeInternal, "read tar header")
		}

		// Build destination path
		destFile := filepath.Join(filepath.Dir(destPath), header.Name)

		// Restore file
		if err := m.restoreFile(tr, destFile, header); err != nil {
			return errors.Wrap(err, errors.CodeInternal, "restore file")
		}
	}

	duration := time.Since(start)
	m.logger.Info("Restore completed in %s", duration)

	return nil
}

// restoreFile restores a single file
func (m *Manager) restoreFile(tr *tar.Reader, destPath string, header *tar.Header) error {
	// Create file
	file, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Set permissions
	if err := os.Chmod(destPath, os.FileMode(header.Mode)); err != nil {
		return err
	}

	// Copy content
	if _, err := io.Copy(file, tr); err != nil {
		return err
	}

	return nil
}

// ListBackups lists available backups
func (m *Manager) ListBackups(dir string) ([]BackupInfo, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, errors.Wrap(err, errors.CodeInternal, "read backup directory")
	}

	var backups []BackupInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Check if it's a backup file
		if !isBackupFile(entry.Name()) {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		backups = append(backups, BackupInfo{
			Name:    entry.Name(),
			Path:    filepath.Join(dir, entry.Name()),
			Size:    info.Size(),
			Created: info.ModTime(),
		})
	}

	return backups, nil
}

// BackupInfo represents backup metadata
type BackupInfo struct {
	Name    string
	Path    string
	Size    int64
	Created time.Time
}

// DeleteBackup deletes a backup
func (m *Manager) DeleteBackup(path string) error {
	if err := os.Remove(path); err != nil {
		return errors.Wrap(err, errors.CodeInternal, "delete backup")
	}
	m.logger.Info("Backup deleted: %s", path)
	return nil
}

// AutoBackup performs automatic backup
func (m *Manager) AutoBackup(ctx context.Context, dir string, retention int) (string, error) {
	// Generate backup filename
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("helm_backup_%s.tar.gz", timestamp)
	backupPath := filepath.Join(dir, filename)

	// Perform backup
	if err := m.Backup(ctx, backupPath); err != nil {
		return "", err
	}

	// Cleanup old backups
	if retention > 0 {
		if err := m.CleanupOldBackups(dir, retention); err != nil {
			m.logger.Error("Failed to cleanup old backups: %v", err)
		}
	}

	return backupPath, nil
}

// CleanupOldBackups removes old backups keeping only the specified number
func (m *Manager) CleanupOldBackups(dir string, keep int) error {
	backups, err := m.ListBackups(dir)
	if err != nil {
		return err
	}

	if len(backups) <= keep {
		return nil
	}

	// Sort by creation time (newest first)
	for i := 0; i < len(backups)-1; i++ {
		for j := i + 1; j < len(backups); j++ {
			if backups[i].Created.Before(backups[j].Created) {
				backups[i], backups[j] = backups[j], backups[i]
			}
		}
	}

	// Delete old backups
	for i := keep; i < len(backups); i++ {
		if err := m.DeleteBackup(backups[i].Path); err != nil {
			m.logger.Error("Failed to delete old backup %s: %v", backups[i].Path, err)
		}
	}

	return nil
}

// ValidateBackup validates a backup file
func (m *Manager) ValidateBackup(ctx context.Context, backupPath string) error {
	// Open backup file
	file, err := os.Open(backupPath)
	if err != nil {
		return errors.Wrap(err, errors.CodeInternal, "open backup file")
	}
	defer file.Close()

	// Try to read as gzip
	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return errors.Wrap(err, errors.CodeInternal, "invalid gzip format")
	}
	gzReader.Close()

	// Reset file pointer
	file.Seek(0, 0)

	// Verify tar format
	gzReader, _ = gzip.NewReader(file)
	defer gzReader.Close()

	tr := tar.NewReader(gzReader)
	fileCount := 0

	for {
		_, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return errors.Wrap(err, errors.CodeInternal, "invalid tar format")
		}
		fileCount++
	}

	if fileCount == 0 {
		return errors.New(CodeInternal, "backup contains no files")
	}

	return nil
}

// ScheduleBackup schedules periodic backups
func ScheduleBackup(m *Manager, dir string, interval time.Duration, retention int) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		if _, err := m.AutoBackup(ctx, dir, retention); err != nil {
			m.logger.Error("Scheduled backup failed: %v", err)
		}
		cancel()
	}
}

// Helper functions

func isBackupFile(name string) bool {
	return filepath.Ext(name) == ".gz" && len(name) > 7
}

// Code for error creation
const CodeInternal = errors.CodeInternal

// Export creates a database export (CSV/JSON format)
func (m *Manager) Export(ctx context.Context, table, format, destPath string) error {
	// This would export specific tables in various formats
	return fmt.Errorf("export not yet implemented")
}

// Import imports data from file
func (m *Manager) Import(ctx context.Context, table, format, srcPath string) error {
	// This would import data from various formats
	return fmt.Errorf("import not yet implemented")
}

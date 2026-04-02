// Package cmd provides the CLI commands for HELM.
package cmd

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/yourname/helm/internal/db"
)

// exportCmd exports data
var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export data",
	Long:  `Export sessions, memory, or full data to a file.`,
}

var exportSessionsCmd = &cobra.Command{
	Use:   "sessions [output-file]",
	Short: "Export sessions",
	Long:  `Export all sessions to a JSON file.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		project, _ := os.Getwd()

		// Get output path
		output := fmt.Sprintf("helm_sessions_%s.json", time.Now().Format("20060102_150405"))
		if len(args) > 0 {
			output = args[0]
		}

		// Fetch sessions
		sessions, err := application.DB.ListSessions(ctx, db.ListSessionsParams{
			Project: project,
			Limit:   10000,
		})
		if err != nil {
			return fmt.Errorf("fetch sessions: %w", err)
		}

		// Create export structure
		export := SessionExport{
			Version:      "1.0",
			ExportedAt:   time.Now(),
			Project:      project,
			SessionCount: int64(len(sessions)),
			Sessions:     sessions,
		}

		// Write to file
		file, err := os.Create(output)
		if err != nil {
			return fmt.Errorf("create file: %w", err)
		}
		defer file.Close()

		enc := json.NewEncoder(file)
		enc.SetIndent("", "  ")
		if err := enc.Encode(export); err != nil {
			return fmt.Errorf("encode sessions: %w", err)
		}

		fmt.Printf("✓ Exported %d sessions to %s\n", len(sessions), output)
		return nil
	},
}

var exportMemoryCmd = &cobra.Command{
	Use:   "memory [output-file]",
	Short: "Export memory",
	Long:  `Export all project memory to a JSON file.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		project, _ := os.Getwd()

		output := fmt.Sprintf("helm_memory_%s.json", time.Now().Format("20060102_150405"))
		if len(args) > 0 {
			output = args[0]
		}

		memories, err := application.DB.ListMemories(ctx, project)
		if err != nil {
			return fmt.Errorf("fetch memories: %w", err)
		}

		export := MemoryExport{
			Version:     "1.0",
			ExportedAt:  time.Now(),
			Project:     project,
			MemoryCount: int64(len(memories)),
			Memories:    memories,
		}

		file, err := os.Create(output)
		if err != nil {
			return fmt.Errorf("create file: %w", err)
		}
		defer file.Close()

		enc := json.NewEncoder(file)
		enc.SetIndent("", "  ")
		if err := enc.Encode(export); err != nil {
			return fmt.Errorf("encode memories: %w", err)
		}

		fmt.Printf("✓ Exported %d memories to %s\n", len(memories), output)
		return nil
	},
}

var exportFullCmd = &cobra.Command{
	Use:   "full [output-file]",
	Short: "Export all data",
	Long:  `Export all project data (sessions, memory, costs) to a tar.gz archive.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		project, _ := os.Getwd()

		output := fmt.Sprintf("helm_export_%s.tar.gz", time.Now().Format("20060102_150405"))
		if len(args) > 0 {
			output = args[0]
		}

		// Create archive
		file, err := os.Create(output)
		if err != nil {
			return fmt.Errorf("create archive: %w", err)
		}
		defer file.Close()

		gzWriter := gzip.NewWriter(file)
		defer gzWriter.Close()

		tarWriter := tar.NewWriter(gzWriter)
		defer tarWriter.Close()

		// Export sessions
		sessions, _ := application.DB.ListSessions(ctx, db.ListSessionsParams{
			Project: project,
			Limit:   10000,
		})

		if err := addToArchive(tarWriter, "sessions.json", sessions); err != nil {
			return err
		}

		// Export memory
		memories, _ := application.DB.ListMemories(ctx, project)
		if err := addToArchive(tarWriter, "memory.json", memories); err != nil {
			return err
		}

		// Export cost records
		costs, _ := application.DB.ListCostRecords(ctx, db.ListCostRecordsParams{
			Project: project,
			Limit:   10000,
		})
		if err := addToArchive(tarWriter, "costs.json", costs); err != nil {
			return err
		}

		// Export config
		config := map[string]interface{}{
			"exported_at": time.Now(),
			"project":     project,
			"version":     "1.0",
		}
		if err := addToArchive(tarWriter, "meta.json", config); err != nil {
			return err
		}

		fmt.Printf("✓ Exported full data to %s\n", output)
		fmt.Printf("  Sessions: %d\n", len(sessions))
		fmt.Printf("  Memories: %d\n", len(memories))
		fmt.Printf("  Costs:    %d\n", len(costs))

		return nil
	},
}

// Export types

type SessionExport struct {
	Version      string      `json:"version"`
	ExportedAt   time.Time   `json:"exported_at"`
	Project      string      `json:"project"`
	SessionCount int64       `json:"session_count"`
	Sessions     interface{} `json:"sessions"`
}

type MemoryExport struct {
	Version     string      `json:"version"`
	ExportedAt  time.Time   `json:"exported_at"`
	Project     string      `json:"project"`
	MemoryCount int64       `json:"memory_count"`
	Memories    interface{} `json:"memories"`
}

func addToArchive(tw *tar.Writer, name string, data interface{}) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	header := &tar.Header{
		Name: name,
		Size: int64(len(jsonData)),
		Mode: 0644,
	}

	if err := tw.WriteHeader(header); err != nil {
		return err
	}

	_, err = tw.Write(jsonData)
	return err
}

func init() {
	rootCmd.AddCommand(exportCmd)

	exportCmd.AddCommand(exportSessionsCmd)
	exportCmd.AddCommand(exportMemoryCmd)
	exportCmd.AddCommand(exportFullCmd)
}

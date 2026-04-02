// Package cmd provides the CLI commands for HELM.
package cmd

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"github.com/yourname/helm/internal/db"
)

// importCmd imports data
var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Import data",
	Long:  `Import sessions, memory, or full data from a file.`,
}

var importSessionsCmd = &cobra.Command{
	Use:   "sessions [input-file]",
	Short: "Import sessions",
	Long:  `Import sessions from a JSON file.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		input := args[0]

		// Read file
		data, err := os.ReadFile(input)
		if err != nil {
			return fmt.Errorf("read file: %w", err)
		}

		// Parse export
		var export SessionExport
		if err := json.Unmarshal(data, &export); err != nil {
			return fmt.Errorf("parse sessions: %w", err)
		}

		// Import sessions
		sessions, ok := export.Sessions.([]interface{})
		if !ok {
			return fmt.Errorf("invalid sessions format")
		}

		imported := 0
		for _, s := range sessions {
			// Convert to session and insert
			sessionData, err := json.Marshal(s)
			if err != nil {
				continue
			}

			var session db.Session
			if err := json.Unmarshal(sessionData, &session); err != nil {
				continue
			}

			// Insert session
			_, err = application.DB.CreateSession(ctx, db.CreateSessionParams{
				ID:               session.ID,
				Provider:         session.Provider,
				Model:            session.Model,
				Project:          session.Project,
				Prompt:           session.Prompt,
				Status:           session.Status,
				InputTokens:      session.InputTokens,
				OutputTokens:     session.OutputTokens,
				CacheReadTokens:  session.CacheReadTokens,
				CacheWriteTokens: session.CacheWriteTokens,
				Cost:             session.Cost,
				Summary:          session.Summary,
				Tags:             session.Tags,
				RawPath:          session.RawPath,
			})
			if err == nil {
				imported++
			}
		}

		fmt.Printf("✓ Imported %d sessions from %s\n", imported, input)
		return nil
	},
}

var importMemoryCmd = &cobra.Command{
	Use:   "memory [input-file]",
	Short: "Import memory",
	Long:  `Import project memory from a JSON file.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		input := args[0]

		data, err := os.ReadFile(input)
		if err != nil {
			return fmt.Errorf("read file: %w", err)
		}

		var export MemoryExport
		if err := json.Unmarshal(data, &export); err != nil {
			return fmt.Errorf("parse memories: %w", err)
		}

		memories, ok := export.Memories.([]interface{})
		if !ok {
			return fmt.Errorf("invalid memories format")
		}

		imported := 0
		for _, m := range memories {
			memoryData, err := json.Marshal(m)
			if err != nil {
				continue
			}

			var memory db.Memory
			if err := json.Unmarshal(memoryData, &memory); err != nil {
				continue
			}

			err = application.DB.UpsertMemory(ctx, db.UpsertMemoryParams{
				ID:         memory.ID,
				Project:    memory.Project,
				Type:       memory.Type,
				Key:        memory.Key,
				Value:      memory.Value,
				Source:     memory.Source,
				Confidence: memory.Confidence,
			})
			if err == nil {
				imported++
			}
		}

		fmt.Printf("✓ Imported %d memories from %s\n", imported, input)
		return nil
	},
}

var importFullCmd = &cobra.Command{
	Use:   "full [input-file]",
	Short: "Import all data",
	Long:  `Import all project data from a tar.gz archive.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		input := args[0]

		file, err := os.Open(input)
		if err != nil {
			return fmt.Errorf("open archive: %w", err)
		}
		defer file.Close()

		gzReader, err := gzip.NewReader(file)
		if err != nil {
			return fmt.Errorf("create gzip reader: %w", err)
		}
		defer gzReader.Close()

		tr := tar.NewReader(gzReader)

		for {
			header, err := tr.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				return fmt.Errorf("read archive: %w", err)
			}

			// Read file content
			content, err := io.ReadAll(tr)
			if err != nil {
				return fmt.Errorf("read file content: %w", err)
			}

			switch header.Name {
			case "sessions.json":
				if err := importSessionsFromJSON(ctx, content); err != nil {
					return fmt.Errorf("import sessions: %w", err)
				}
			case "memory.json":
				if err := importMemoriesFromJSON(ctx, content); err != nil {
					return fmt.Errorf("import memories: %w", err)
				}
			case "costs.json":
				if err := importCostsFromJSON(ctx, content); err != nil {
					return fmt.Errorf("import costs: %w", err)
				}
			}
		}

		fmt.Printf("✓ Imported full data from %s\n", input)
		return nil
	},
}

func importSessionsFromJSON(ctx context.Context, data []byte) error {
	var export SessionExport
	if err := json.Unmarshal(data, &export); err != nil {
		return err
	}

	sessions, ok := export.Sessions.([]interface{})
	if !ok {
		return fmt.Errorf("invalid sessions format")
	}

	imported := 0
	for _, s := range sessions {
		sessionData, err := json.Marshal(s)
		if err != nil {
			continue
		}

		var session db.Session
		if err := json.Unmarshal(sessionData, &session); err != nil {
			continue
		}

		_, err = application.DB.CreateSession(ctx, db.CreateSessionParams{
			ID:               session.ID,
			Provider:         session.Provider,
			Model:            session.Model,
			Project:          session.Project,
			Prompt:           session.Prompt,
			Status:           session.Status,
			InputTokens:      session.InputTokens,
			OutputTokens:     session.OutputTokens,
			CacheReadTokens:  session.CacheReadTokens,
			CacheWriteTokens: session.CacheWriteTokens,
			Cost:             session.Cost,
			Summary:          session.Summary,
			Tags:             session.Tags,
			RawPath:          session.RawPath,
		})
		if err == nil {
			imported++
		}
	}

	fmt.Printf("  Sessions: %d imported\n", imported)
	return nil
}

func importMemoriesFromJSON(ctx context.Context, data []byte) error {
	var export MemoryExport
	if err := json.Unmarshal(data, &export); err != nil {
		return err
	}

	memories, ok := export.Memories.([]interface{})
	if !ok {
		return fmt.Errorf("invalid memories format")
	}

	imported := 0
	for _, m := range memories {
		memoryData, err := json.Marshal(m)
		if err != nil {
			continue
		}

		var memory db.Memory
		if err := json.Unmarshal(memoryData, &memory); err != nil {
			continue
		}

		err = application.DB.UpsertMemory(ctx, db.UpsertMemoryParams{
			ID:         memory.ID,
			Project:    memory.Project,
			Type:       memory.Type,
			Key:        memory.Key,
			Value:      memory.Value,
			Source:     memory.Source,
			Confidence: memory.Confidence,
		})
		if err == nil {
			imported++
		}
	}

	fmt.Printf("  Memories: %d imported\n", imported)
	return nil
}

func importCostsFromJSON(ctx context.Context, data []byte) error {
	var costs []db.CostRecord
	if err := json.Unmarshal(data, &costs); err != nil {
		return err
	}

	imported := 0
	for _, c := range costs {
		_, err := application.DB.CreateCostRecord(ctx, db.CreateCostRecordParams{
			ID:               c.ID,
			SessionID:        c.SessionID,
			Project:          c.Project,
			Provider:         c.Provider,
			Model:            c.Model,
			InputTokens:      c.InputTokens,
			OutputTokens:     c.OutputTokens,
			CacheReadTokens:  c.CacheReadTokens,
			CacheWriteTokens: c.CacheWriteTokens,
			TotalCost:        c.TotalCost,
		})
		if err == nil {
			imported++
		}
	}

	fmt.Printf("  Costs: %d imported\n", imported)
	return nil
}

func init() {
	rootCmd.AddCommand(importCmd)

	importCmd.AddCommand(importSessionsCmd)
	importCmd.AddCommand(importMemoryCmd)
	importCmd.AddCommand(importFullCmd)
}

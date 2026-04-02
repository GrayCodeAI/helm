// Package cmd provides the CLI commands for HELM.
package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/yourname/helm/internal/db"
	"github.com/yourname/helm/internal/memory"
)

// memoryCmd represents the memory command.
var memoryCmd = &cobra.Command{
	Use:   "memory",
	Short: "Manage project memory",
	Long:  `View, add, and manage persistent project memory that survives sessions.`,
}

// memoryListCmd lists all memories.
var memoryListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all memories for the current project",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		project, _ := os.Getwd()

		memories, err := application.DB.ListMemories(ctx, project)
		if err != nil {
			return fmt.Errorf("list memories: %w", err)
		}

		if len(memories) == 0 {
			fmt.Println("No memories found for this project.")
			return nil
		}

		fmt.Printf("Memories for %s:\n\n", project)
		for _, m := range memories {
			fmt.Printf("[%s] %s: %s\n", m.Type, m.Key, m.Value)
		}

		return nil
	},
}

// memorySetCmd sets a memory value.
var memorySetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a memory value",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		project, _ := os.Getwd()
		key := args[0]
		value := args[1]

		memType, _ := cmd.Flags().GetString("type")
		if memType == "" {
			memType = string(memory.TypeFact)
		}

		_, err := application.DB.CreateMemory(ctx, db.CreateMemoryParams{
			ID:      uuid.New().String(),
			Project: project,
			Type:    memType,
			Key:     key,
			Value:   value,
			Source:  "manual",
		})
		if err != nil {
			return fmt.Errorf("create memory: %w", err)
		}

		fmt.Printf("✓ Set memory: %s = %s\n", key, value)
		return nil
	},
}

// memoryGetCmd gets a memory value.
var memoryGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a memory value",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		project, _ := os.Getwd()
		key := args[0]

		m, err := application.DB.GetMemory(ctx, db.GetMemoryParams{
			Project: project,
			Key:     key,
		})
		if err != nil {
			return fmt.Errorf("memory not found: %w", err)
		}

		fmt.Printf("%s: %s\n", m.Key, m.Value)
		return nil
	},
}

// memoryForgetCmd removes a memory.
var memoryForgetCmd = &cobra.Command{
	Use:   "forget <key>",
	Short: "Remove a memory",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		project, _ := os.Getwd()
		key := args[0]

		// Get memory ID first
		m, err := application.DB.GetMemory(ctx, db.GetMemoryParams{
			Project: project,
			Key:     key,
		})
		if err != nil {
			return fmt.Errorf("memory not found: %w", err)
		}

		err = application.DB.DeleteMemory(ctx, m.ID)
		if err != nil {
			return fmt.Errorf("delete memory: %w", err)
		}

		fmt.Printf("✓ Forgot memory: %s\n", key)
		return nil
	},
}

// memoryRecallCmd shows current project memory context.
var memoryRecallCmd = &cobra.Command{
	Use:   "recall",
	Short: "Show current project memory context",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		project, _ := os.Getwd()

		memories, err := application.DB.ListMemories(ctx, project)
		if err != nil {
			return fmt.Errorf("list memories: %w", err)
		}

		if len(memories) == 0 {
			fmt.Println("No project memory yet. Use `helm memory set` to add some.")
			return nil
		}

		// Group by type
		byType := make(map[string][]db.Memory)
		for _, m := range memories {
			byType[m.Type] = append(byType[m.Type], m)
		}

		fmt.Printf("Project Memory for %s:\n\n", project)

		for _, t := range []string{"convention", "decision", "preference", "fact", "correction", "skill"} {
			if items, ok := byType[t]; ok {
				fmt.Printf("[%s]\n", strings.ToUpper(t))
				for _, m := range items {
					fmt.Printf("  %s: %s\n", m.Key, m.Value)
				}
				fmt.Println()
			}
		}

		return nil
	},
}

func init() {
	memorySetCmd.Flags().String("type", "fact", "Memory type (convention, decision, preference, fact, correction, skill)")

	memoryCmd.AddCommand(memoryListCmd)
	memoryCmd.AddCommand(memorySetCmd)
	memoryCmd.AddCommand(memoryGetCmd)
	memoryCmd.AddCommand(memoryForgetCmd)
	memoryCmd.AddCommand(memoryRecallCmd)

	rootCmd.AddCommand(memoryCmd)
}

// Package cmd provides the CLI commands for HELM.
package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

// promptCmd represents the prompts command.
var promptCmd = &cobra.Command{
	Use:   "prompts",
	Short: "Manage prompt library",
	Long:  `View and manage the prompt library for quick agent launches.`,
}

// promptListCmd lists all prompts.
var promptListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available prompts",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		prompts, err := application.DB.ListPrompts(ctx)
		if err != nil {
			return fmt.Errorf("list prompts: %w", err)
		}

		if len(prompts) == 0 {
			fmt.Println("No prompts found. Built-in prompts will be loaded on first run.")
			return nil
		}

		fmt.Println("Available Prompts:")
		fmt.Println()
		for _, p := range prompts {
			complexity := ""
			if p.Complexity.Valid {
				complexity = fmt.Sprintf(" [%s]", p.Complexity.String)
			}
			desc := ""
			if p.Description.Valid {
				desc = fmt.Sprintf(" - %s", p.Description.String)
			}
			fmt.Printf("  %-20s%s%s\n", p.Name, complexity, desc)
		}

		return nil
	},
}

// promptShowCmd shows a specific prompt.
var promptShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Show a specific prompt template",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		name := args[0]

		p, err := application.DB.GetPrompt(ctx, name)
		if err != nil {
			return fmt.Errorf("prompt not found: %w", err)
		}

		fmt.Printf("Name: %s\n", p.Name)
		if p.Description.Valid {
			fmt.Printf("Description: %s\n", p.Description.String)
		}
		if p.Complexity.Valid {
			fmt.Printf("Complexity: %s\n", p.Complexity.String)
		}
		fmt.Printf("Source: %s\n", p.Source)
		fmt.Println()
		fmt.Println("Template:")
		fmt.Println(p.Template)

		return nil
	},
}

// promptRunCmd runs a prompt from the library.
var promptRunCmd = &cobra.Command{
	Use:   "run <name>",
	Short: "Run a prompt from the library",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		name := args[0]

		p, err := application.DB.GetPrompt(ctx, name)
		if err != nil {
			return fmt.Errorf("prompt not found: %w", err)
		}

		// Run the prompt template
		return runSession(p.Template)
	},
}

func init() {
	promptCmd.AddCommand(promptListCmd)
	promptCmd.AddCommand(promptShowCmd)
	promptCmd.AddCommand(promptRunCmd)

	rootCmd.AddCommand(promptCmd)
}

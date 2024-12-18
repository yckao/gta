package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yckao/gta/pkg/logger"
	"github.com/yckao/gta/pkg/provider"
)

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Clean up temporary IAM role bindings",
	Long: `Clean up temporary IAM role bindings in a project. If a user is specified,
only bindings for that user will be cleaned up.

Example:
  # List all temporary bindings that would be cleaned
  gta clean --project=my-project --dry-run

  # Clean up all temporary bindings
  gta clean --project=my-project

  # Clean up temporary bindings for a specific user
  gta clean --project=my-project --user=user@example.com`,
	RunE: runClean,
}

func init() {
	flags := cleanCmd.Flags()
	flags.StringVarP(&project, "project", "p", "", "Project ID")
	flags.StringVarP(&user, "user", "u", "", "Filter bindings by user")
	flags.BoolVarP(&dryRun, "dry-run", "d", false, "Preview bindings that would be cleaned without making any changes")

	cleanCmd.MarkFlagRequired("project")
}

func runClean(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	if dryRun {
		logger.Info("Running in dry-run mode - no changes will be made")
	}

	p, err := provider.NewGCPProvider(ctx, dryRun)
	if err != nil {
		return fmt.Errorf("failed to create GCP provider: %v", err)
	}

	opts := &provider.GCPOptions{
		Project: project,
		User:    user,
	}

	if err := p.CleanTemporaryBindings(opts); err != nil {
		return fmt.Errorf("failed to clean temporary bindings: %v", err)
	}

	return nil
}

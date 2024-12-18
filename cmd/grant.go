package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/yckao/gta/pkg/logger"
	"github.com/yckao/gta/pkg/provider"
)

var grantCmd = &cobra.Command{
	Use:   "grant [roles...]",
	Short: "Grant temporary IAM roles",
	Long: `Grant temporary IAM roles in various cloud providers.
The roles will be automatically revoked when the program exits or receives an interrupt signal.

Example:
  # Grant roles to current user
  gta grant roles/viewer roles/editor --project=my-project

  # Grant roles to specific user
  gta grant roles/viewer roles/editor --project=my-project --user=user@example.com

  # Preview changes without applying them
  gta grant roles/viewer --project=my-project --dry-run`,
	Args: cobra.MinimumNArgs(1),
	RunE: runGrant,
}

func init() {
	flags := grantCmd.Flags()
	flags.StringVarP(&project, "project", "p", "", "Project ID (required)")
	flags.StringVarP(&user, "user", "u", "", "User or service account to grant the role to (defaults to current user)")
	flags.DurationVarP(&ttl, "ttl", "t", 1*time.Hour, "Time-to-live for the granted permission")
	flags.BoolVarP(&dryRun, "dry-run", "d", false, "Preview changes without applying them")

	grantCmd.MarkFlagRequired("project")
}

func runGrant(cmd *cobra.Command, args []string) error {
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
		Roles:   args,
		User:    user,
		TTL:     ttl,
	}

	if err := p.Grant(opts); err != nil {
		return fmt.Errorf("failed to grant roles: %v", err)
	}

	if dryRun {
		return nil
	}

	// Set up signal handling for cleanup
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	logger.Info("Waiting for interrupt signal to revoke roles (Ctrl+C to exit)...")
	<-sigChan

	logger.Info("Revoking roles...")
	if err := p.Revoke(opts); err != nil {
		return fmt.Errorf("failed to revoke roles: %v", err)
	}

	return nil
}

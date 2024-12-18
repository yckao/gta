package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yckao/gta/pkg/provider"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List temporary IAM role bindings",
	Long: `List temporary IAM role bindings in a project. If a user is specified,
only bindings for that user will be shown.

Example:
  gta list --project=my-project
  gta list --project=my-project --user=user@example.com`,
	RunE: runList,
}

func init() {
	flags := listCmd.Flags()
	flags.StringVarP(&project, "project", "p", "", "Project ID")
	flags.StringVarP(&user, "user", "u", "", "Filter bindings by user")

	listCmd.MarkFlagRequired("project")
}

func runList(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	p, err := provider.NewGCPProvider(ctx, false)
	if err != nil {
		return fmt.Errorf("failed to create GCP provider: %v", err)
	}

	opts := &provider.GCPOptions{
		Project: project,
		User:    user,
	}

	if err := p.ListTemporaryBindings(opts); err != nil {
		return fmt.Errorf("failed to list temporary bindings: %v", err)
	}

	return nil
}

package cmd

import (
	"github.com/seiji/menukey/internal/plan"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status [config.yaml]",
	Short: "Show synchronization status",
	Long: `Status compares the declared shortcuts with this Mac without changing
anything. --check exits with status 1 when synchronization is needed.`,
	Args: cobra.RangeArgs(0, 1),
	RunE: runStatus,
}

var (
	statusApp       string
	statusAll       bool
	statusCheck     bool
	statusUnmanaged bool
)

func init() {
	statusCmd.Flags().StringVar(&statusApp, "app", "", "Only consider this bundle identifier")
	statusCmd.Flags().BoolVar(&statusAll, "all", false, "Also list shortcuts that already match")
	statusCmd.Flags().BoolVar(&statusCheck, "check", false, "Exit with status 1 when changes are pending")
	statusCmd.Flags().BoolVar(&statusUnmanaged, "unmanaged", false, "Also list shortcuts not declared in the configuration")
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	path, err := resolveConfigPath(args)
	if err != nil {
		return err
	}
	return statusConfig(cmd, path, statusApp, statusAll, statusUnmanaged, statusCheck)
}

func statusConfig(cmd *cobra.Command, path, app string, all, unmanaged, check bool) error {
	cfg, err := loadOrCreateConfig(path)
	if err != nil {
		return err
	}

	plans, err := plan.Build(cmd.Context(), backend, cfg, app)
	if err != nil {
		return err
	}

	pending := printPlans(cmd.OutOrStdout(), plans, all, unmanaged)
	if pending > 0 && check {
		return &exitError{code: 1}
	}
	return nil
}

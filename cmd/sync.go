package cmd

import (
	"fmt"

	"github.com/seiji/menukey/internal/plan"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync [config.yaml]",
	Short: "Synchronize declared shortcuts to this Mac",
	Long: `Sync makes this Mac's application menu shortcuts match the declared
shortcuts. It adds and updates declared entries, but never deletes shortcuts
that are absent from the configuration.`,
	Args: cobra.RangeArgs(0, 1),
	RunE: runSync,
}

var (
	syncApp    string
	syncDryRun bool
)

func init() {
	syncCmd.Flags().StringVar(&syncApp, "app", "", "Only synchronize this bundle identifier")
	syncCmd.Flags().BoolVar(&syncDryRun, "dry-run", false, "Show what would change without writing")
	rootCmd.AddCommand(syncCmd)
}

func runSync(cmd *cobra.Command, args []string) error {
	path, err := resolveConfigPath(args)
	if err != nil {
		return err
	}
	return syncConfig(cmd, path, syncApp, syncDryRun)
}

func syncConfig(cmd *cobra.Command, path, app string, dryRun bool) error {
	cfg, err := loadOrCreateConfig(path)
	if err != nil {
		return err
	}

	ctx := cmd.Context()
	plans, err := plan.Build(ctx, backend, cfg, app)
	if err != nil {
		return err
	}

	out := cmd.OutOrStdout()
	pending := printPlans(out, plans, false, false)
	if pending == 0 {
		fmt.Fprintln(out, "\nNothing to sync.")
		return nil
	}
	if dryRun {
		fmt.Fprintln(out, "\nDry run: nothing was written.")
		return nil
	}
	if err := plan.Apply(ctx, backend, plans); err != nil {
		return err
	}

	fmt.Fprintf(out, "\nSynced %s.\n", pluralize(pending, "change"))
	// Applications read NSUserKeyEquivalents when they build their menus, so a
	// running app keeps showing the old shortcut until it is restarted.
	fmt.Fprintln(out, "Restart the affected applications for the new shortcuts to take effect.")
	return nil
}

func pluralize(n int, noun string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, noun)
	}
	return fmt.Sprintf("%d %ss", n, noun)
}

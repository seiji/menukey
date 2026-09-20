// Package cmd implements the menukey command line interface.
package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
)

// version is overridden at build time with -ldflags "-X ...cmd.version=v0.1.0".
var version = "dev"

var rootCmd = &cobra.Command{
	Use:   "menukey",
	Short: "Declaratively manage macOS application menu shortcuts",
	Long: `menukey manages macOS application menu keyboard shortcuts from a YAML file.

Shortcuts are stored in each application's NSUserKeyEquivalents preference,
keyed by the menu item title as the application displays it. menukey merges the
shortcuts you declare into that preference and leaves every other entry alone.`,
	Version:       version,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// exitError carries an exit status that is not an error condition, such as
// `status --check` reporting that synchronization is pending.
type exitError struct {
	code int
}

func (e *exitError) Error() string {
	return fmt.Sprintf("exit status %d", e.code)
}

// Execute runs the root command and returns the process exit status.
func Execute() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		var exit *exitError
		if errors.As(err, &exit) {
			return exit.code
		}
		fmt.Fprintln(os.Stderr, "Error:", err)
		return 1
	}
	return 0
}

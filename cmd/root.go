// Package cmd implements the menukey command line interface.
package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"

	"github.com/spf13/cobra"
)

const (
	developmentVersion = "dev"
	modulePath         = "github.com/seiji/menukey"
)

// version is overridden at build time with -ldflags "-X ...cmd.version=v0.1.0".
var version = developmentVersion

func currentVersion() string {
	if version != developmentVersion {
		return version
	}

	info, ok := debug.ReadBuildInfo()
	if !ok {
		return version
	}
	return resolvedVersion(version, info)
}

func resolvedVersion(linkedVersion string, info *debug.BuildInfo) string {
	if linkedVersion != developmentVersion {
		return linkedVersion
	}
	if info == nil || info.Main.Path != modulePath || info.Main.Version == "" || info.Main.Version == "(devel)" {
		return linkedVersion
	}
	return info.Main.Version
}

var rootCmd = &cobra.Command{
	Use:   "menukey",
	Short: "Declaratively manage macOS App Shortcuts",
	Long: `menukey declaratively manages macOS App Shortcuts.

Shortcuts are stored in each application's NSUserKeyEquivalents preference,
keyed by the menu item title as the application displays it. menukey merges the
shortcuts you declare into that preference and leaves every other entry alone.`,
	Version:       currentVersion(),
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

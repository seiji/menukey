package cmd

import (
	"fmt"
	"maps"
	"slices"

	"github.com/seiji/menukey/internal/config"
	"github.com/seiji/menukey/internal/keyspec"
	"github.com/spf13/cobra"
)

var importCmd = &cobra.Command{
	Use:   "import <bundle-id-or-app-path>",
	Short: "Import an application's current shortcuts into the configuration",
	Long: `Import reads an application's current shortcuts and writes them to the
configuration. It refuses to replace an application already in the
configuration unless --replace is supplied.`,
	Args: cobra.ExactArgs(1),
	RunE: runImport,
}

var (
	importName    string
	importReplace bool
)

func init() {
	importCmd.Flags().StringVar(&importName, "name", "", "Human readable app name (overrides the name in an app bundle)")
	importCmd.Flags().BoolVar(&importReplace, "replace", false, "Replace an existing application entry")
	rootCmd.AddCommand(importCmd)
}

func runImport(cmd *cobra.Command, args []string) error {
	app, err := appForImport(cmd.Context(), args[0], importName)
	if err != nil {
		return err
	}
	bundleID := app.Bundle
	path, err := defaultConfigPath()
	if err != nil {
		return err
	}
	cfg, err := loadOrCreateConfig(path)
	if err != nil {
		return err
	}

	i := appIndex(cfg, bundleID)
	if i >= 0 && !importReplace {
		return fmt.Errorf("%s is already declared (use --replace to overwrite its shortcuts)", bundleID)
	}

	entries, err := backend.Read(cmd.Context(), bundleID)
	if err != nil {
		return err
	}
	if i >= 0 && !cmd.Flags().Changed("name") {
		app.Name = cfg.Apps[i].Name
	}
	app.Shortcuts = []config.Shortcut{}
	for _, menu := range slices.Sorted(maps.Keys(entries)) {
		key, err := keyspec.Decode(entries[menu])
		if err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "Warning: skipping %q: %v\n", menu, err)
			continue
		}
		app.Shortcuts = append(app.Shortcuts, config.Shortcut{Menu: menu, Key: key.String()})
	}
	if i < 0 {
		cfg.Apps = append(cfg.Apps, app)
	} else {
		cfg.Apps[i] = app
	}
	if err := cfg.Save(path); err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Imported %s for %s into %s.\n", pluralize(len(app.Shortcuts), "shortcut"), bundleID, path)
	return nil
}

package cmd

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/seiji/menukey/internal/appinfo"
	"github.com/seiji/menukey/internal/config"
	"github.com/spf13/cobra"
)

func loadConfigForEdit(path string) (*config.Config, error) {
	return loadOrCreateConfig(path)
}

func appIndex(cfg *config.Config, bundle string) int {
	for i, app := range cfg.Apps {
		if app.Bundle == bundle {
			return i
		}
	}
	return -1
}

func sortShortcuts(shortcuts []config.Shortcut) {
	slices.SortFunc(shortcuts, func(a, b config.Shortcut) int {
		return strings.Compare(a.Menu, b.Menu)
	})
}

var (
	inspectApplication = appinfo.Inspect
	findApplication    = appinfo.FindByBundle
	searchApplications = appinfo.Search
)

func appFromArgument(ctx context.Context, argument, name string) (config.App, error) {
	if filepath.Ext(filepath.Clean(argument)) != ".app" {
		return config.App{Bundle: argument, Name: name, Shortcuts: []config.Shortcut{}}, nil
	}
	info, err := inspectApplication(ctx, argument)
	if err != nil {
		return config.App{}, err
	}
	return configApp(info, name), nil
}

func appForImport(ctx context.Context, argument, name string) (config.App, error) {
	if filepath.Ext(filepath.Clean(argument)) == ".app" {
		return appFromArgument(ctx, argument, name)
	}
	info, err := findApplication(ctx, argument)
	if err != nil {
		return config.App{}, err
	}
	return configApp(info, name), nil
}

func configApp(info appinfo.App, name string) config.App {
	if name != "" {
		info.Name = name
	}
	return config.App{Bundle: info.Bundle, Name: info.Name, Shortcuts: []config.Shortcut{}}
}

var appAddCmd = &cobra.Command{
	Use:   "add <bundle-id-or-app-path>",
	Short: "Add an application to the configuration",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path, err := defaultConfigPath()
		if err != nil {
			return err
		}
		app, err := appFromArgument(cmd.Context(), args[0], appAddName)
		if err != nil {
			return err
		}
		cfg, err := loadConfigForEdit(path)
		if err != nil {
			return err
		}
		if appIndex(cfg, app.Bundle) >= 0 {
			return fmt.Errorf("%s is already declared", app.Bundle)
		}
		cfg.Apps = append(cfg.Apps, app)
		if err := cfg.Save(path); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Added %s to %s.\n", app.Label(), path)
		return nil
	},
}

var appAddName string

var appSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Find installed macOS applications",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		apps, err := searchApplications(cmd.Context(), args[0])
		if err != nil {
			return err
		}
		for _, app := range apps {
			fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\n", app.Name, app.Bundle, app.Path)
		}
		return nil
	},
}

var shortcutSetCmd = &cobra.Command{
	Use:   "set <bundle-id> <menu> <key>",
	Short: "Add or update a shortcut",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		path, err := defaultConfigPath()
		if err != nil {
			return err
		}
		cfg, err := loadConfigForEdit(path)
		if err != nil {
			return err
		}
		i := appIndex(cfg, args[0])
		if i < 0 {
			return fmt.Errorf("%s is not declared; add it with `menukey add %s`", args[0], args[0])
		}
		app := &cfg.Apps[i]
		found := false
		for j := range app.Shortcuts {
			if app.Shortcuts[j].Menu == args[1] {
				app.Shortcuts[j].Key = args[2]
				found = true
				break
			}
		}
		if !found {
			app.Shortcuts = append(app.Shortcuts, config.Shortcut{Menu: args[1], Key: args[2]})
		}
		sortShortcuts(app.Shortcuts)
		if err := cfg.Save(path); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Set %s: %s = %s.\n", args[0], args[1], args[2])
		return nil
	},
}

var unsetCmd = &cobra.Command{
	Use:   "unset <bundle-id> [menu]",
	Short: "Remove an application or shortcut from the configuration",
	Long: `Without a menu title, unset removes the application's declaration.
With a menu title, it removes that shortcut's declaration. sync does not delete
shortcuts from macOS, so existing shortcuts remain in place.`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		path, err := defaultConfigPath()
		if err != nil {
			return err
		}
		cfg, err := loadConfigForEdit(path)
		if err != nil {
			return err
		}
		i := appIndex(cfg, args[0])
		if i < 0 {
			return fmt.Errorf("%s is not declared", args[0])
		}
		if len(args) == 1 {
			cfg.Apps = append(cfg.Apps[:i], cfg.Apps[i+1:]...)
			if err := cfg.Save(path); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Removed %s from %s.\n", args[0], path)
			return nil
		}

		app := &cfg.Apps[i]
		for j, shortcut := range app.Shortcuts {
			if shortcut.Menu == args[1] {
				app.Shortcuts = append(app.Shortcuts[:j], app.Shortcuts[j+1:]...)
				if err := cfg.Save(path); err != nil {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Removed %s: %s.\n", args[0], args[1])
				return nil
			}
		}
		return fmt.Errorf("%s has no shortcut named %q", args[0], args[1])
	},
}

func init() {
	appAddCmd.Flags().StringVar(&appAddName, "name", "", "Human readable app name (overrides the name in an app bundle)")
	rootCmd.AddCommand(appAddCmd, appSearchCmd, shortcutSetCmd, unsetCmd)
}

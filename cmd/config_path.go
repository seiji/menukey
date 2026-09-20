package cmd

import (
	"fmt"
	"os"

	"github.com/seiji/menukey/internal/config"
)

var configFile string

func init() {
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "Configuration file (default: ~/.config/menukey/config.yaml)")
}

// resolveConfigPath accepts the optional positional file used by sync and
// status. --config is preferred for scripts because it is unambiguous when a
// command has other positional arguments.
func resolveConfigPath(args []string) (string, error) {
	if len(args) > 1 {
		return "", fmt.Errorf("expected at most one configuration file")
	}
	if configFile != "" {
		if len(args) == 1 {
			return "", fmt.Errorf("specify either a configuration file or --config, not both")
		}
		return configFile, nil
	}
	if len(args) == 1 {
		return args[0], nil
	}
	return config.DefaultPath()
}

func defaultConfigPath() (string, error) {
	if configFile != "" {
		return configFile, nil
	}
	return config.DefaultPath()
}

// loadOrCreateConfig returns the desired state. A missing configuration means
// no applications are managed yet, so create that empty state on first use.
func loadOrCreateConfig(path string) (*config.Config, error) {
	cfg, err := config.Load(path)
	if !os.IsNotExist(err) {
		return cfg, err
	}
	cfg = &config.Config{Version: config.Version, Apps: []config.App{}}
	if err := cfg.Save(path); err != nil {
		return nil, err
	}
	return cfg, nil
}

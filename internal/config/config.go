// Package config loads and validates menukey YAML configuration files.
package config

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/seiji/menukey/internal/keyspec"
	"gopkg.in/yaml.v3"
)

// Version is the only configuration schema version understood by this build.
const Version = 1

// Config is a parsed menukey configuration file.
type Config struct {
	Version int   `yaml:"version"`
	Apps    []App `yaml:"apps"`
}

// App is the set of shortcuts declared for one application.
type App struct {
	// Bundle is the application bundle identifier, e.g. com.google.Chrome.
	Bundle string `yaml:"bundle"`
	// Name is an optional human readable name used in status output.
	Name      string     `yaml:"name,omitempty"`
	Shortcuts []Shortcut `yaml:"shortcuts"`
}

// Shortcut binds a menu item title to a key equivalent.
type Shortcut struct {
	// Menu is the menu item title exactly as the application displays it;
	// NSUserKeyEquivalents is keyed by that title.
	Menu string `yaml:"menu"`
	Key  string `yaml:"key"`

	// Parsed is filled in by validation so that callers do not parse twice.
	Parsed keyspec.Key `yaml:"-"`
}

// Label returns the name to show for the app in CLI output.
func (a App) Label() string {
	if a.Name == "" {
		return a.Bundle
	}
	return fmt.Sprintf("%s (%s)", a.Name, a.Bundle)
}

// DefaultPath returns the configuration path used when no file is specified.
// XDG_CONFIG_HOME takes precedence; otherwise the conventional ~/.config path
// is used so that the file is easy to keep in a dotfiles repository.
func DefaultPath() (string, error) {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "menukey", "config.yaml"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("locate home directory: %w", err)
	}
	return filepath.Join(home, ".config", "menukey", "config.yaml"), nil
}

// Load reads and validates a configuration file.
func Load(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	cfg, err := Parse(f)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return cfg, nil
}

// Parse reads and validates a configuration from r.
func Parse(r io.Reader) (*Config, error) {
	dec := yaml.NewDecoder(r)
	// Reject unknown fields so that a misspelled key is an error rather than a
	// silently ignored shortcut.
	dec.KnownFields(true)

	var cfg Config
	if err := dec.Decode(&cfg); err != nil {
		if errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("configuration is empty")
		}
		return nil, err
	}

	// A configuration is one document. Accepting a later document would make
	// apply silently ignore settings that the user may expect it to process.
	var extra any
	if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return nil, fmt.Errorf("configuration must contain only one YAML document")
		}
		return nil, err
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Validate checks the configuration and fills in the parsed key of every
// shortcut.
func (c *Config) Validate() error {
	if c.Version != Version {
		return fmt.Errorf("unsupported version %d, want %d", c.Version, Version)
	}
	seenBundles := make(map[string]int, len(c.Apps))
	for i := range c.Apps {
		app := &c.Apps[i]
		if strings.TrimSpace(app.Bundle) == "" {
			return fmt.Errorf("apps[%d]: bundle is required", i)
		}
		if prev, dup := seenBundles[app.Bundle]; dup {
			return fmt.Errorf("apps[%d]: bundle %q is already declared in apps[%d]", i, app.Bundle, prev)
		}
		seenBundles[app.Bundle] = i

		if err := app.validate(); err != nil {
			return fmt.Errorf("apps[%d] (%s): %w", i, app.Bundle, err)
		}
	}
	return nil
}

func (a *App) validate() error {
	seenMenus := make(map[string]int, len(a.Shortcuts))
	seenKeys := make(map[keyspec.Key]string, len(a.Shortcuts))

	for i := range a.Shortcuts {
		s := &a.Shortcuts[i]
		if strings.TrimSpace(s.Menu) == "" {
			return fmt.Errorf("shortcuts[%d]: menu is required", i)
		}
		if prev, dup := seenMenus[s.Menu]; dup {
			return fmt.Errorf("shortcuts[%d]: menu %q is already declared in shortcuts[%d]", i, s.Menu, prev)
		}
		seenMenus[s.Menu] = i

		key, err := keyspec.Parse(s.Key)
		if err != nil {
			return fmt.Errorf("shortcuts[%d] (%s): %w", i, s.Menu, err)
		}
		// Two menu items sharing a key equivalent means one of them silently
		// never fires, so reject it before writing anything.
		if prev, dup := seenKeys[key]; dup {
			return fmt.Errorf("shortcuts[%d] (%s): key %q is already assigned to %q", i, s.Menu, key, prev)
		}
		seenKeys[key] = s.Menu
		s.Parsed = key
	}
	return nil
}

// Save validates and atomically writes a configuration file, creating its
// parent directory when necessary.
func (c *Config) Save(path string) error {
	if err := c.Validate(); err != nil {
		return err
	}
	out, err := c.Marshal()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	f, err := os.CreateTemp(filepath.Dir(path), ".menukey-*.yaml")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if _, err := f.Write(out); err != nil {
		f.Close()
		return err
	}
	if err := f.Chmod(0o644); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// Marshal renders the configuration as YAML, the format used by import.
func (c *Config) Marshal() ([]byte, error) {
	var b strings.Builder
	enc := yaml.NewEncoder(&b)
	enc.SetIndent(2)
	if err := enc.Encode(c); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	return []byte(b.String()), nil
}

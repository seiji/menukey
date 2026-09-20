package cmd

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/seiji/menukey/internal/appinfo"
	"github.com/seiji/menukey/internal/config"
	"github.com/seiji/menukey/internal/prefs"
	"github.com/spf13/cobra"
)

func TestSyncUsesDefaultStyleWorkflow(t *testing.T) {
	path := t.TempDir() + "/config.yaml"
	cfg := &config.Config{Version: config.Version, Apps: []config.App{{
		Bundle:    "com.google.Chrome",
		Shortcuts: []config.Shortcut{{Menu: "New Tab", Key: "ctrl+t"}},
	}}}
	if err := cfg.Save(path); err != nil {
		t.Fatal(err)
	}

	oldBackend, oldConfigFile, oldApp, oldDryRun := backend, configFile, syncApp, syncDryRun
	defer func() {
		backend, configFile, syncApp, syncDryRun = oldBackend, oldConfigFile, oldApp, oldDryRun
	}()
	memory := prefs.NewMemoryBackend(nil)
	backend, configFile, syncApp, syncDryRun = memory, path, "", false

	var out bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.SetOut(&out)
	if err := runSync(cmd, nil); err != nil {
		t.Fatalf("runSync failed: %v", err)
	}
	if got, want := len(memory.Writes), 1; got != want {
		t.Fatalf("writes = %d, want %d", got, want)
	}
	if !strings.Contains(out.String(), "Synced 1 change.") {
		t.Errorf("output = %q, want sync completion", out.String())
	}
}

func TestMissingConfigurationIsCreated(t *testing.T) {
	path := t.TempDir() + "/config.yaml"
	cfg, err := loadOrCreateConfig(path)
	if err != nil {
		t.Fatalf("loadOrCreateConfig failed: %v", err)
	}
	if len(cfg.Apps) != 0 {
		t.Errorf("apps = %d, want none", len(cfg.Apps))
	}
	if _, err := config.Load(path); err != nil {
		t.Errorf("created configuration cannot be loaded: %v", err)
	}
}

func TestAppSearchPrintsApplications(t *testing.T) {
	oldSearch := searchApplications
	defer func() { searchApplications = oldSearch }()
	searchApplications = func(_ context.Context, _ string) ([]appinfo.App, error) {
		return []appinfo.App{{Name: "Google Chrome", Bundle: "com.google.Chrome", Path: "/Applications/Google Chrome.app"}}, nil
	}

	var out bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.SetOut(&out)
	if err := appSearchCmd.RunE(cmd, []string{"chrome"}); err != nil {
		t.Fatalf("app search failed: %v", err)
	}
	if got, want := out.String(), "Google Chrome\tcom.google.Chrome\t/Applications/Google Chrome.app\n"; got != want {
		t.Errorf("output = %q, want %q", got, want)
	}
}

func TestAppAddReadsAnApplicationBundle(t *testing.T) {
	path := t.TempDir() + "/config.yaml"
	oldConfigFile, oldAppName, oldInspect := configFile, appAddName, inspectApplication
	defer func() {
		configFile, appAddName, inspectApplication = oldConfigFile, oldAppName, oldInspect
	}()
	configFile, appAddName = path, ""
	inspectApplication = func(_ context.Context, path string) (appinfo.App, error) {
		return appinfo.App{Name: "Google Chrome", Bundle: "com.google.Chrome", Path: path}, nil
	}

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.SetOut(&bytes.Buffer{})
	if err := appAddCmd.RunE(cmd, []string{"/Applications/Google Chrome.app"}); err != nil {
		t.Fatalf("app add failed: %v", err)
	}
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := cfg.Apps[0].Bundle, "com.google.Chrome"; got != want {
		t.Errorf("bundle = %q, want %q", got, want)
	}
	if got, want := cfg.Apps[0].Name, "Google Chrome"; got != want {
		t.Errorf("name = %q, want %q", got, want)
	}
}

func TestUnsetRemovesShortcutThenApplication(t *testing.T) {
	path := t.TempDir() + "/config.yaml"
	cfg := &config.Config{Version: config.Version, Apps: []config.App{{
		Bundle:    "com.google.Chrome",
		Shortcuts: []config.Shortcut{{Menu: "New Tab", Key: "ctrl+t"}},
	}}}
	if err := cfg.Save(path); err != nil {
		t.Fatal(err)
	}
	oldConfigFile := configFile
	defer func() { configFile = oldConfigFile }()
	configFile = path

	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	if err := unsetCmd.RunE(cmd, []string{"com.google.Chrome", "New Tab"}); err != nil {
		t.Fatalf("unset shortcut failed: %v", err)
	}
	if err := unsetCmd.RunE(cmd, []string{"com.google.Chrome"}); err != nil {
		t.Fatalf("unset app failed: %v", err)
	}
	loaded, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := len(loaded.Apps); got != 0 {
		t.Errorf("apps = %d, want none", got)
	}
}

func TestConfigurationCommandsCreateAConfig(t *testing.T) {
	path := t.TempDir() + "/config.yaml"
	oldConfigFile, oldAppName := configFile, appAddName
	defer func() {
		configFile, appAddName = oldConfigFile, oldAppName
	}()
	configFile, appAddName = path, "Chrome"

	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	if err := appAddCmd.RunE(cmd, []string{"com.google.Chrome"}); err != nil {
		t.Fatalf("app add failed: %v", err)
	}
	if err := shortcutSetCmd.RunE(cmd, []string{"com.google.Chrome", "New Tab", "ctrl+t"}); err != nil {
		t.Fatalf("shortcut set failed: %v", err)
	}

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := cfg.Apps[0].Name, "Chrome"; got != want {
		t.Errorf("name = %q, want %q", got, want)
	}
	if got, want := cfg.Apps[0].Shortcuts[0].Key, "ctrl+t"; got != want {
		t.Errorf("key = %q, want %q", got, want)
	}
}

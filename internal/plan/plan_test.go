package plan

import (
	"context"
	"strings"
	"testing"

	"github.com/seiji/menukey/internal/config"
	"github.com/seiji/menukey/internal/prefs"
)

const chromeYAML = `
version: 1
apps:
  - bundle: com.google.Chrome
    name: Google Chrome
    shortcuts:
      - menu: "New Tab"
        key: "ctrl+t"
      - menu: "Close Tab"
        key: "ctrl+w"
      - menu: "Reopen Closed Tab"
        key: "ctrl+shift+t"
      - menu: "Bookmark This Tab..."
        key: "ctrl+d"
`

func loadConfig(t *testing.T, yaml string) *config.Config {
	t.Helper()
	cfg, err := config.Parse(strings.NewReader(yaml))
	if err != nil {
		t.Fatalf("config.Parse failed: %v", err)
	}
	return cfg
}

func TestBuild(t *testing.T) {
	backend := prefs.NewMemoryBackend(map[string]map[string]string{
		"com.google.Chrome": {
			"New Tab": "^t",
			// Same shortcut, spelled the way macOS may store it.
			"Reopen Closed Tab": "$^T",
			"Close Tab":         "@w",
		},
	})

	plans, err := Build(context.Background(), backend, loadConfig(t, chromeYAML), "")
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	if len(plans) != 1 {
		t.Fatalf("got %d plans, want 1", len(plans))
	}

	want := []Change{
		{Menu: "New Tab", Type: Unchanged, Before: "ctrl+t", After: "ctrl+t", Encoded: "^t"},
		{Menu: "Close Tab", Type: Update, Before: "cmd+w", After: "ctrl+w", Encoded: "^w"},
		{Menu: "Reopen Closed Tab", Type: Unchanged, Before: "ctrl+shift+t", After: "ctrl+shift+t", Encoded: "^$t"},
		{Menu: "Bookmark This Tab...", Type: Add, After: "ctrl+d", Encoded: "^d"},
	}

	got := plans[0].Changes
	if len(got) != len(want) {
		t.Fatalf("got %d changes, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("changes[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}

	if pending := plans[0].Pending(); len(pending) != 2 {
		t.Errorf("got %d pending changes, want 2: %+v", len(pending), pending)
	}
	if !plans[0].HasPending() {
		t.Error("HasPending() = false, want true")
	}
}

// A shortcut stored in a form menukey cannot decode still shows up, and is
// overwritten rather than skipped.
func TestBuildUndecodableCurrentValue(t *testing.T) {
	backend := prefs.NewMemoryBackend(map[string]map[string]string{
		"com.google.Chrome": {"New Tab": "bogus"},
	})

	plans, err := Build(context.Background(), backend, loadConfig(t, chromeYAML), "")
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	got := plans[0].Changes[0]
	if got.Type != Update || got.Before != "bogus" {
		t.Errorf("changes[0] = %+v, want an Update from \"bogus\"", got)
	}
}

func TestBuildNoPendingChanges(t *testing.T) {
	backend := prefs.NewMemoryBackend(map[string]map[string]string{
		"com.google.Chrome": {
			"New Tab":              "^t",
			"Close Tab":            "^w",
			"Reopen Closed Tab":    "^$t",
			"Bookmark This Tab...": "^d",
		},
	})

	plans, err := Build(context.Background(), backend, loadConfig(t, chromeYAML), "")
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	if plans[0].HasPending() {
		t.Errorf("HasPending() = true, want false: %+v", plans[0].Pending())
	}
}

func TestBuildBundleFilter(t *testing.T) {
	yaml := chromeYAML + `  - bundle: com.apple.Safari
    shortcuts:
      - menu: "New Tab"
        key: "ctrl+t"
`
	backend := prefs.NewMemoryBackend(nil)
	cfg := loadConfig(t, yaml)

	plans, err := Build(context.Background(), backend, cfg, "com.apple.Safari")
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	if len(plans) != 1 || plans[0].App.Bundle != "com.apple.Safari" {
		t.Fatalf("got %d plans for %v, want only com.apple.Safari", len(plans), plans)
	}

	if _, err := Build(context.Background(), backend, cfg, "com.unknown.App"); err == nil {
		t.Error("Build with an unknown bundle filter succeeded, want error")
	}
}

func TestBuildCollectsUnmanagedShortcuts(t *testing.T) {
	backend := prefs.NewMemoryBackend(map[string]map[string]string{
		"com.google.Chrome": {
			"New Tab":          "^t",
			"Reload This Page": "^r",
			"Bookmark Page":    "^d",
		},
	})

	plans, err := Build(context.Background(), backend, loadConfig(t, chromeYAML), "")
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if len(plans[0].Unmanaged) != 2 {
		t.Fatalf("got %d unmanaged shortcuts, want 2: %v", len(plans[0].Unmanaged), plans[0].Unmanaged)
	}

	// Unmanaged shortcuts are sorted by menu title for stable output.
	if plans[0].Unmanaged[0].Menu != "Bookmark Page" || plans[0].Unmanaged[0].Key != "ctrl+d" {
		t.Errorf("Unmanaged[0] = %+v, want Bookmark Page: ctrl+d", plans[0].Unmanaged[0])
	}
	if plans[0].Unmanaged[1].Menu != "Reload This Page" || plans[0].Unmanaged[1].Key != "ctrl+r" {
		t.Errorf("Unmanaged[1] = %+v, want Reload This Page: ctrl+r", plans[0].Unmanaged[1])
	}
}

func TestApplyWritesOnlyPendingChanges(t *testing.T) {
	backend := prefs.NewMemoryBackend(map[string]map[string]string{
		"com.google.Chrome": {
			"New Tab": "^t",
			// Not declared in the configuration: must survive apply.
			"Reload This Page": "^r",
		},
	})

	plans, err := Build(context.Background(), backend, loadConfig(t, chromeYAML), "")
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	if err := Apply(context.Background(), backend, plans); err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	if len(backend.Writes) != 1 {
		t.Fatalf("got %d writes, want 1", len(backend.Writes))
	}
	written := backend.Writes[0].Entries
	if _, ok := written["New Tab"]; ok {
		t.Errorf("apply rewrote an unchanged shortcut: %v", written)
	}

	want := map[string]string{
		"New Tab":              "^t",
		"Close Tab":            "^w",
		"Reopen Closed Tab":    "^$t",
		"Bookmark This Tab...": "^d",
		"Reload This Page":     "^r",
	}
	got := backend.Domains["com.google.Chrome"]
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for menu, value := range want {
		if got[menu] != value {
			t.Errorf("entries[%q] = %q, want %q", menu, got[menu], value)
		}
	}
}

func TestApplyWithNothingPending(t *testing.T) {
	backend := prefs.NewMemoryBackend(map[string]map[string]string{
		"com.google.Chrome": {
			"New Tab":              "^t",
			"Close Tab":            "^w",
			"Reopen Closed Tab":    "^$t",
			"Bookmark This Tab...": "^d",
		},
	})

	plans, err := Build(context.Background(), backend, loadConfig(t, chromeYAML), "")
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	if err := Apply(context.Background(), backend, plans); err != nil {
		t.Fatalf("Apply failed: %v", err)
	}
	if len(backend.Writes) != 0 {
		t.Errorf("got %d writes, want none: %+v", len(backend.Writes), backend.Writes)
	}
}

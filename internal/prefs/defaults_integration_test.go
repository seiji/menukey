//go:build integration

package prefs

import (
	"context"
	"maps"
	"os/exec"
	"testing"
)

// testDomain is a scratch preference domain; no real application reads it.
const testDomain = "com.seiji.menukey.test"

// TestDefaultsBackendRoundTrip exercises the real defaults(1) plumbing: menu
// titles with spaces and punctuation, key equivalents containing function key
// code points, and the merge semantics of -dict-add.
func TestDefaultsBackendRoundTrip(t *testing.T) {
	ctx := context.Background()
	b := DefaultsBackend{}

	t.Cleanup(func() {
		if err := exec.Command("defaults", "delete", testDomain).Run(); err != nil {
			t.Logf("cleaning up %s failed: %v", testDomain, err)
		}
	})

	entries, err := b.Read(ctx, testDomain)
	if err != nil {
		t.Fatalf("Read of an unset domain failed: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("%s already holds %v; remove it and re-run", testDomain, entries)
	}

	first := map[string]string{
		"New Tab":              "^t",
		"Bookmark This Tab...": "^d",
		"Scroll Left":          "~", // opt+left
	}
	if err := b.Write(ctx, testDomain, first); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	got, err := b.Read(ctx, testDomain)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if !maps.Equal(got, first) {
		t.Fatalf("Read returned %v, want %v", got, first)
	}

	// A second write must merge rather than replace.
	second := map[string]string{"New Tab": "^n", "Close Tab": "^w"}
	if err := b.Write(ctx, testDomain, second); err != nil {
		t.Fatalf("second Write failed: %v", err)
	}

	want := maps.Clone(first)
	maps.Copy(want, second)

	got, err = b.Read(ctx, testDomain)
	if err != nil {
		t.Fatalf("Read after merge failed: %v", err)
	}
	if !maps.Equal(got, want) {
		t.Errorf("Read returned %v, want %v", got, want)
	}
}

func TestDefaultsBackendReadRejectsBadDomain(t *testing.T) {
	if _, err := (DefaultsBackend{}).Read(context.Background(), "-currentHost"); err == nil {
		t.Error("Read with a flag-like bundle identifier succeeded, want error")
	}
}

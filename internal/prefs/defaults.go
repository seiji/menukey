package prefs

import (
	"bytes"
	"context"
	"fmt"
	"maps"
	"os/exec"
	"slices"
	"strings"
)

// DefaultsBackend reads and writes preferences through defaults(1), which goes
// through cfprefsd and therefore stays consistent with what the preference
// system has cached.
type DefaultsBackend struct{}

var _ Backend = DefaultsBackend{}

// Read implements Backend.
func (DefaultsBackend) Read(ctx context.Context, bundleID string) (map[string]string, error) {
	if err := checkBundleID(bundleID); err != nil {
		return nil, err
	}

	// `defaults export` succeeds with an empty dictionary for a domain that
	// does not exist, so a missing application is not an error here.
	out, err := run(ctx, "defaults", "export", bundleID, "-")
	if err != nil {
		return nil, err
	}

	entries, err := parseKeyEquivalents(bytes.NewReader(out))
	if err != nil {
		return nil, fmt.Errorf("reading %s of %s: %w", Key, bundleID, err)
	}
	return entries, nil
}

// Write implements Backend. It uses -dict-add so that menu titles absent from
// entries keep their current key equivalent.
func (DefaultsBackend) Write(ctx context.Context, bundleID string, entries map[string]string) error {
	if err := checkBundleID(bundleID); err != nil {
		return err
	}
	if len(entries) == 0 {
		return nil
	}

	args := []string{"write", bundleID, Key, "-dict-add"}
	for _, menu := range slices.Sorted(maps.Keys(entries)) {
		if menu == "" {
			return fmt.Errorf("menu title must not be empty")
		}
		args = append(args, menu, entries[menu])
	}

	_, err := run(ctx, "defaults", args...)
	return err
}

// checkBundleID rejects identifiers that defaults(1) would read as a flag.
func checkBundleID(bundleID string) error {
	if bundleID == "" {
		return fmt.Errorf("bundle identifier must not be empty")
	}
	if strings.HasPrefix(bundleID, "-") {
		return fmt.Errorf("invalid bundle identifier %q", bundleID)
	}
	return nil
}

func run(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	out, err := cmd.Output()
	if err != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return nil, fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, msg)
		}
		return nil, fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}
	return out, nil
}

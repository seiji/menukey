// Package plan compares the shortcuts declared in a configuration against the
// ones currently stored in macOS preferences.
package plan

import (
	"context"
	"fmt"
	"maps"
	"slices"

	"github.com/seiji/menukey/internal/config"
	"github.com/seiji/menukey/internal/keyspec"
	"github.com/seiji/menukey/internal/prefs"
)

// ChangeType is what applying a shortcut would do to the current preferences.
type ChangeType int

const (
	// Unchanged means the stored key equivalent already matches.
	Unchanged ChangeType = iota
	// Add means the menu item has no key equivalent set yet.
	Add
	// Update means a different key equivalent is currently set.
	Update
)

// Symbol is the diff marker used in CLI output.
func (t ChangeType) Symbol() string {
	switch t {
	case Add:
		return "+"
	case Update:
		return "~"
	default:
		return "="
	}
}

// Change is the planned outcome for a single menu item.
type Change struct {
	Menu string
	Type ChangeType
	// Before is the current value in friendly notation, empty for Add. A value
	// that cannot be decoded is carried through verbatim.
	Before string
	// After is the desired value in friendly notation.
	After string
	// Encoded is the value to store in NSUserKeyEquivalents.
	Encoded string
}

// AppPlan holds the planned changes for one application.
type AppPlan struct {
	App       config.App
	Changes   []Change
	Unmanaged []UnmanagedShortcut
}

// UnmanagedShortcut is a stored shortcut that the configuration does not
// declare. It is shown by status --unmanaged but is never changed by Apply.
type UnmanagedShortcut struct {
	Menu string
	Key  string
}

// Pending returns the changes that applying would actually write.
func (p AppPlan) Pending() []Change {
	pending := make([]Change, 0, len(p.Changes))
	for _, c := range p.Changes {
		if c.Type != Unchanged {
			pending = append(pending, c)
		}
	}
	return pending
}

// HasPending reports whether applying would write anything.
func (p AppPlan) HasPending() bool {
	for _, c := range p.Changes {
		if c.Type != Unchanged {
			return true
		}
	}
	return false
}

// Build reads the current preferences of every app in cfg and works out what
// synchronizing it would change. When bundleFilter is not empty, only that bundle
// identifier is considered.
func Build(ctx context.Context, backend prefs.Backend, cfg *config.Config, bundleFilter string) ([]AppPlan, error) {
	plans := make([]AppPlan, 0, len(cfg.Apps))
	for _, app := range cfg.Apps {
		if bundleFilter != "" && app.Bundle != bundleFilter {
			continue
		}

		current, err := backend.Read(ctx, app.Bundle)
		if err != nil {
			return nil, err
		}

		p := AppPlan{App: app, Changes: make([]Change, 0, len(app.Shortcuts))}
		declared := make(map[string]struct{}, len(app.Shortcuts))
		for _, s := range app.Shortcuts {
			p.Changes = append(p.Changes, buildChange(s, current))
			declared[s.Menu] = struct{}{}
		}
		for _, menu := range slices.Sorted(maps.Keys(current)) {
			if _, ok := declared[menu]; ok {
				continue
			}
			key := current[menu]
			if decoded, err := keyspec.Decode(key); err == nil {
				key = decoded.String()
			}
			p.Unmanaged = append(p.Unmanaged, UnmanagedShortcut{Menu: menu, Key: key})
		}
		plans = append(plans, p)
	}

	if len(plans) == 0 && bundleFilter != "" {
		return nil, fmt.Errorf("no app with bundle identifier %q in the configuration", bundleFilter)
	}
	return plans, nil
}

func buildChange(s config.Shortcut, current map[string]string) Change {
	c := Change{
		Menu:    s.Menu,
		After:   s.Parsed.String(),
		Encoded: s.Parsed.Encode(),
	}

	stored, ok := current[s.Menu]
	if !ok {
		c.Type = Add
		return c
	}

	// Compare the parsed forms: "@$t" and "$@t" are the same shortcut, and so
	// are "@$t" and "@T".
	if decoded, err := keyspec.Decode(stored); err == nil {
		c.Before = decoded.String()
		if decoded == s.Parsed {
			c.Type = Unchanged
			return c
		}
	} else {
		// Keep an undecodable value visible instead of hiding it behind the
		// planned overwrite.
		c.Before = stored
	}

	c.Type = Update
	return c
}

// Apply writes the pending changes of every plan. Menu items that are not
// declared keep whatever they are set to.
func Apply(ctx context.Context, backend prefs.Backend, plans []AppPlan) error {
	for _, p := range plans {
		pending := p.Pending()
		if len(pending) == 0 {
			continue
		}

		entries := make(map[string]string, len(pending))
		for _, c := range pending {
			entries[c.Menu] = c.Encoded
		}
		if err := backend.Write(ctx, p.App.Bundle, entries); err != nil {
			return fmt.Errorf("%s: %w", p.App.Bundle, err)
		}
	}
	return nil
}

// Package prefs reads and writes the NSUserKeyEquivalents entry of a macOS
// application's preference domain.
package prefs

import (
	"context"
	"maps"
)

// Key is the preference entry that holds per-application menu shortcuts.
const Key = "NSUserKeyEquivalents"

// Backend reads and writes key equivalents for an application bundle. The
// production implementation shells out to defaults(1); tests use MemoryBackend.
type Backend interface {
	// Read returns the current menu title to key equivalent mapping. A domain
	// or entry that does not exist yields an empty map and a nil error.
	Read(ctx context.Context, bundleID string) (map[string]string, error)
	// Write merges entries into the existing mapping, leaving menu titles that
	// are not mentioned untouched.
	Write(ctx context.Context, bundleID string, entries map[string]string) error
}

// MemoryBackend is an in-memory Backend for tests and for dry runs.
type MemoryBackend struct {
	Domains map[string]map[string]string
	// Writes records every Write call in order, so tests can assert that a dry
	// run wrote nothing.
	Writes []WriteCall
}

// WriteCall is one recorded Write on a MemoryBackend.
type WriteCall struct {
	BundleID string
	Entries  map[string]string
}

// NewMemoryBackend returns a MemoryBackend seeded with domains.
func NewMemoryBackend(domains map[string]map[string]string) *MemoryBackend {
	b := &MemoryBackend{Domains: map[string]map[string]string{}}
	for bundle, entries := range domains {
		b.Domains[bundle] = maps.Clone(entries)
	}
	return b
}

// Read implements Backend.
func (b *MemoryBackend) Read(_ context.Context, bundleID string) (map[string]string, error) {
	entries, ok := b.Domains[bundleID]
	if !ok {
		return map[string]string{}, nil
	}
	return maps.Clone(entries), nil
}

// Write implements Backend.
func (b *MemoryBackend) Write(_ context.Context, bundleID string, entries map[string]string) error {
	b.Writes = append(b.Writes, WriteCall{BundleID: bundleID, Entries: maps.Clone(entries)})
	if b.Domains == nil {
		b.Domains = map[string]map[string]string{}
	}
	if b.Domains[bundleID] == nil {
		b.Domains[bundleID] = map[string]string{}
	}
	maps.Copy(b.Domains[bundleID], entries)
	return nil
}

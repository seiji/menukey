// Package appinfo discovers macOS application bundle metadata.
package appinfo

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

// App identifies an installed macOS application bundle.
type App struct {
	Name   string
	Bundle string
	Path   string
}

var commandOutput = func(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		if message := strings.TrimSpace(stderr.String()); message != "" {
			return nil, fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, message)
		}
		return nil, fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}
	return out, nil
}

// Inspect reads an application's bundle identifier and display name from its
// Contents/Info.plist. plutil supports both XML and binary property lists.
func Inspect(ctx context.Context, path string) (App, error) {
	cleanPath := filepath.Clean(path)
	plist := filepath.Join(cleanPath, "Contents", "Info.plist")
	bundle, err := plistValue(ctx, plist, "CFBundleIdentifier")
	if err != nil {
		return App{}, fmt.Errorf("read application at %s: %w", path, err)
	}
	if bundle == "" {
		return App{}, fmt.Errorf("read application at %s: CFBundleIdentifier is empty", path)
	}

	name, err := plistValue(ctx, plist, "CFBundleDisplayName")
	if err != nil || name == "" {
		name, err = plistValue(ctx, plist, "CFBundleName")
	}
	if err != nil || name == "" {
		name = strings.TrimSuffix(filepath.Base(cleanPath), filepath.Ext(cleanPath))
	}
	return App{Name: name, Bundle: bundle, Path: cleanPath}, nil
}

func plistValue(ctx context.Context, plist, key string) (string, error) {
	out, err := commandOutput(ctx, "plutil", "-extract", key, "raw", "-o", "-", plist)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// Search finds application bundles whose Spotlight display name matches query.
func Search(ctx context.Context, query string) ([]App, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("search query must not be empty")
	}
	query = strings.NewReplacer("\\", "\\\\", "\"", "\\\"").Replace(query)
	spotlightQuery := "kMDItemContentType == \"com.apple.application-bundle\" && kMDItemDisplayName == \"*" + query + "*\"cd"
	out, err := commandOutput(ctx, "mdfind", spotlightQuery)
	if err != nil {
		return nil, err
	}

	apps := make([]App, 0)
	for _, path := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if path == "" {
			continue
		}
		app, err := Inspect(ctx, path)
		if err != nil {
			// Spotlight can return stale paths. Ignore them rather than making a
			// search result unusable because one application was removed.
			continue
		}
		apps = append(apps, app)
	}
	slices.SortFunc(apps, func(a, b App) int {
		if byName := strings.Compare(a.Name, b.Name); byName != 0 {
			return byName
		}
		return strings.Compare(a.Path, b.Path)
	})
	return apps, nil
}

package appinfo

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func TestInspect(t *testing.T) {
	old := commandOutput
	defer func() { commandOutput = old }()
	commandOutput = func(_ context.Context, name string, args ...string) ([]byte, error) {
		if name != "plutil" {
			t.Fatalf("command = %q, want plutil", name)
		}
		switch args[1] {
		case "CFBundleIdentifier":
			return []byte("com.google.Chrome\n"), nil
		case "CFBundleDisplayName":
			return []byte("Google Chrome\n"), nil
		default:
			t.Fatalf("key = %q", args[1])
			return nil, nil
		}
	}

	app, err := Inspect(context.Background(), "/Applications/Google Chrome.app")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := app.Bundle, "com.google.Chrome"; got != want {
		t.Errorf("Bundle = %q, want %q", got, want)
	}
	if got, want := app.Name, "Google Chrome"; got != want {
		t.Errorf("Name = %q, want %q", got, want)
	}
}

func TestSearchSortsResults(t *testing.T) {
	old := commandOutput
	defer func() { commandOutput = old }()
	commandOutput = func(_ context.Context, name string, args ...string) ([]byte, error) {
		if name == "mdfind" {
			return []byte("/Applications/Zeta.app\n/Applications/Alpha.app\n"), nil
		}
		plist := args[len(args)-1]
		path := filepath.Dir(filepath.Dir(plist))
		switch args[1] {
		case "CFBundleIdentifier":
			return []byte("com.example." + strings.TrimSuffix(filepath.Base(path), ".app")), nil
		case "CFBundleDisplayName":
			return []byte(strings.TrimSuffix(filepath.Base(path), ".app")), nil
		default:
			t.Fatalf("key = %q", args[1])
			return nil, nil
		}
	}

	apps, err := Search(context.Background(), "a")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(apps), 2; got != want {
		t.Fatalf("results = %d, want %d", got, want)
	}
	if got, want := apps[0].Name, "Alpha"; got != want {
		t.Errorf("first result = %q, want %q", got, want)
	}
}

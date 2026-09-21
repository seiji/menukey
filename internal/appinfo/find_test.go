package appinfo

import (
	"context"
	"testing"
)

func TestFindByBundle(t *testing.T) {
	old := commandOutput
	defer func() { commandOutput = old }()
	commandOutput = func(_ context.Context, name string, args ...string) ([]byte, error) {
		if name == "mdfind" {
			return []byte("/Applications/Google Chrome.app\n"), nil
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

	app, err := FindByBundle(context.Background(), "com.google.Chrome")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := app.Name, "Google Chrome"; got != want {
		t.Errorf("Name = %q, want %q", got, want)
	}
}

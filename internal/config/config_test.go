package config

import (
	"strings"
	"testing"
)

const validYAML = `
version: 1
apps:
  - bundle: com.google.Chrome
    name: Google Chrome
    shortcuts:
      - menu: "New Tab"
        key: "ctrl+t"
      - menu: "Close Tab"
        key: "ctrl+w"
`

func TestParse(t *testing.T) {
	cfg, err := Parse(strings.NewReader(validYAML))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(cfg.Apps) != 1 {
		t.Fatalf("got %d apps, want 1", len(cfg.Apps))
	}
	app := cfg.Apps[0]
	if got, want := app.Label(), "Google Chrome (com.google.Chrome)"; got != want {
		t.Errorf("Label() = %q, want %q", got, want)
	}
	if got, want := len(app.Shortcuts), 2; got != want {
		t.Fatalf("got %d shortcuts, want %d", got, want)
	}
	if got, want := app.Shortcuts[0].Parsed.Encode(), "^t"; got != want {
		t.Errorf("Parsed.Encode() = %q, want %q", got, want)
	}
}

func TestLabelWithoutName(t *testing.T) {
	app := App{Bundle: "com.google.Chrome"}
	if got, want := app.Label(), "com.google.Chrome"; got != want {
		t.Errorf("Label() = %q, want %q", got, want)
	}
}

func TestParseErrors(t *testing.T) {
	tests := []struct {
		name string
		yaml string
		want string
	}{
		{
			name: "empty",
			yaml: "",
			want: "empty",
		},
		{
			name: "unsupported version",
			yaml: "version: 2\napps:\n  - bundle: a\n",
			want: "unsupported version",
		},
		{
			name: "missing bundle",
			yaml: "version: 1\napps:\n  - name: Chrome\n",
			want: "bundle is required",
		},
		{
			name: "duplicate bundle",
			yaml: "version: 1\napps:\n  - bundle: a\n  - bundle: a\n",
			want: "already declared",
		},
		{
			name: "missing menu",
			yaml: "version: 1\napps:\n  - bundle: a\n    shortcuts:\n      - key: ctrl+t\n",
			want: "menu is required",
		},
		{
			name: "duplicate menu",
			yaml: "version: 1\napps:\n  - bundle: a\n    shortcuts:\n      - menu: New Tab\n        key: ctrl+t\n      - menu: New Tab\n        key: ctrl+n\n",
			want: "already declared",
		},
		{
			// Spelled differently, but the same key equivalent.
			name: "duplicate key",
			yaml: "version: 1\napps:\n  - bundle: a\n    shortcuts:\n      - menu: New Tab\n        key: ctrl+t\n      - menu: New Window\n        key: control+t\n",
			want: "already assigned",
		},
		{
			name: "bad key",
			yaml: "version: 1\napps:\n  - bundle: a\n    shortcuts:\n      - menu: New Tab\n        key: hyper+t\n",
			want: "unknown modifier",
		},
		{
			name: "unknown field",
			yaml: "version: 1\napps:\n  - bundle_id: a\n",
			want: "field bundle_id not found",
		},
		{
			name: "multiple documents",
			yaml: "version: 1\napps:\n  - bundle: a\n---\nversion: 1\napps:\n  - bundle: b\n",
			want: "only one YAML document",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(strings.NewReader(tt.yaml))
			if err == nil {
				t.Fatalf("Parse(%q) succeeded, want error", tt.yaml)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error = %q, want it to contain %q", err, tt.want)
			}
		})
	}
}

func TestEmptyConfigIsValid(t *testing.T) {
	cfg, err := Parse(strings.NewReader("version: 1\napps: []\n"))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(cfg.Apps) != 0 {
		t.Errorf("got %d apps, want none", len(cfg.Apps))
	}
}

func TestSaveRoundTrip(t *testing.T) {
	path := t.TempDir() + "/nested/config.yaml"
	cfg := &Config{Version: Version, Apps: []App{{
		Bundle:    "com.google.Chrome",
		Shortcuts: []Shortcut{{Menu: "New Tab", Key: "ctrl+t"}},
	}}}
	if err := cfg.Save(path); err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if got, want := loaded.Apps[0].Shortcuts[0].Key, "ctrl+t"; got != want {
		t.Errorf("Key = %q, want %q", got, want)
	}
}

func TestMarshalRoundTrip(t *testing.T) {
	cfg, err := Parse(strings.NewReader(validYAML))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	out, err := cfg.Marshal()
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	if strings.Contains(string(out), "parsed") {
		t.Errorf("Marshal leaked the parsed key into YAML:\n%s", out)
	}

	again, err := Parse(strings.NewReader(string(out)))
	if err != nil {
		t.Fatalf("re-parsing marshalled config failed: %v\n%s", err, out)
	}
	if got, want := again.Apps[0].Shortcuts[1].Key, "ctrl+w"; got != want {
		t.Errorf("Key = %q, want %q", got, want)
	}
}

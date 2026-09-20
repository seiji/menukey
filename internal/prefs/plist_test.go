package prefs

import (
	"strings"
	"testing"
)

const plistHeader = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
`

func plistDoc(body string) string {
	return plistHeader + body + "\n</plist>\n"
}

func TestParseKeyEquivalents(t *testing.T) {
	// Entries of other types surround the one we care about, the way they do in
	// a real application domain.
	body := `<dict>
	<key>AnArray</key>
	<array>
		<string>a</string>
		<dict><key>nested</key><string>b</string></dict>
	</array>
	<key>SomeData</key>
	<data>YWJj</data>
	<key>NSUserKeyEquivalents</key>
	<dict>
		<key>New Tab</key>
		<string>^t</string>
		<key>Reopen Closed Tab</key>
		<string>^$t</string>
	</dict>
	<key>LastSeen</key>
	<date>2026-08-23T00:00:00Z</date>
</dict>`

	entries, err := parseKeyEquivalents(strings.NewReader(plistDoc(body)))
	if err != nil {
		t.Fatalf("parseKeyEquivalents failed: %v", err)
	}

	want := map[string]string{"New Tab": "^t", "Reopen Closed Tab": "^$t"}
	if len(entries) != len(want) {
		t.Fatalf("got %d entries, want %d: %v", len(entries), len(want), entries)
	}
	for menu, value := range want {
		if entries[menu] != value {
			t.Errorf("entries[%q] = %q, want %q", menu, entries[menu], value)
		}
	}
}

func TestParseKeyEquivalentsAbsent(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"empty domain", `<dict/>`},
		{"other keys only", `<dict><key>Other</key><string>x</string></dict>`},
		{"empty entry", `<dict><key>NSUserKeyEquivalents</key><dict/></dict>`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entries, err := parseKeyEquivalents(strings.NewReader(plistDoc(tt.body)))
			if err != nil {
				t.Fatalf("parseKeyEquivalents failed: %v", err)
			}
			if len(entries) != 0 {
				t.Errorf("got %v, want no entries", entries)
			}
		})
	}
}

func TestParseKeyEquivalentsErrors(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "entry is not a dict",
			body: `<dict><key>NSUserKeyEquivalents</key><string>^t</string></dict>`,
			want: "want a <dict>",
		},
		{
			name: "value is not a string",
			body: `<dict><key>NSUserKeyEquivalents</key><dict><key>New Tab</key><integer>1</integer></dict></dict>`,
			want: "want a <string>",
		},
		{
			name: "key without a value",
			body: `<dict><key>NSUserKeyEquivalents</key></dict>`,
			want: "missing value",
		},
		{
			name: "truncated",
			body: `<dict><key>NSUserKeyEquivalents</key>`,
			want: "syntax error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entries, err := parseKeyEquivalents(strings.NewReader(plistDoc(tt.body)))
			if err == nil {
				t.Fatalf("parseKeyEquivalents = %v, want error", entries)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error = %q, want it to contain %q", err, tt.want)
			}
		})
	}
}

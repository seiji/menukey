package keyspec

import "testing"

func TestParseEncode(t *testing.T) {
	tests := []struct {
		notation string
		encoded  string
		// canonical is the notation Parse round-trips to; empty means it
		// matches the input.
		canonical string
	}{
		{notation: "t", encoded: "t"},
		{notation: "ctrl+t", encoded: "^t"},
		{notation: "cmd+t", encoded: "@t"},
		{notation: "ctrl+shift+t", encoded: "^$t"},
		{notation: "cmd+shift+t", encoded: "@$t"},
		{notation: "option+left", encoded: "~", canonical: "opt+left"},
		{notation: "cmd+opt+i", encoded: "@~i"},
		{notation: "cmd+ctrl+opt+shift+a", encoded: "@^~$a"},
		{notation: "alt+f5", encoded: "~", canonical: "opt+f5"},
		{notation: "cmd+f19", encoded: "@"},
		{notation: "shift+tab", encoded: "$\t"},
		{notation: "cmd+space", encoded: "@ "},
		{notation: "cmd+enter", encoded: "@\r", canonical: "cmd+return"},
		{notation: "ctrl+esc", encoded: "^\x1b", canonical: "ctrl+escape"},
		{notation: "cmd+pgdn", encoded: "@", canonical: "cmd+pagedown"},
		{notation: "cmd++", encoded: "@+"},
		{notation: "cmd+/", encoded: "@/"},

		// Modifier order in the input is irrelevant; the output is canonical.
		{notation: "shift+ctrl+t", encoded: "^$t", canonical: "ctrl+shift+t"},
		{notation: "Shift+CTRL+T", encoded: "^$t", canonical: "ctrl+shift+t"},

		// An uppercase letter implies Shift, the way Cocoa reads it.
		{notation: "ctrl+T", encoded: "^$t", canonical: "ctrl+shift+t"},
		{notation: "T", encoded: "$t", canonical: "shift+t"},

		// Aliases are accepted but never emitted.
		{notation: "command+w", encoded: "@w", canonical: "cmd+w"},
		{notation: "control+w", encoded: "^w", canonical: "ctrl+w"},
	}

	for _, tt := range tests {
		t.Run(tt.notation, func(t *testing.T) {
			k, err := Parse(tt.notation)
			if err != nil {
				t.Fatalf("Parse(%q) failed: %v", tt.notation, err)
			}
			if got := k.Encode(); got != tt.encoded {
				t.Errorf("Encode() = %q, want %q", got, tt.encoded)
			}

			want := tt.canonical
			if want == "" {
				want = tt.notation
			}
			if got := k.String(); got != want {
				t.Errorf("String() = %q, want %q", got, want)
			}
		})
	}
}

func TestParseErrors(t *testing.T) {
	tests := []string{
		"",
		"   ",
		"ctrl+",
		"ctrl+shift",
		"hyper+t",
		"ctrl+ctrl+t",
		"ctrl+nosuchkey",
		"cmd+f20",
		"cmd+f0",
	}

	for _, in := range tests {
		t.Run(in, func(t *testing.T) {
			if k, err := Parse(in); err == nil {
				t.Errorf("Parse(%q) = %v, want error", in, k)
			}
		})
	}
}

func TestDecode(t *testing.T) {
	tests := []struct {
		encoded  string
		notation string
	}{
		{"^t", "ctrl+t"},
		{"@w", "cmd+w"},
		{"^$t", "ctrl+shift+t"},
		{"~", "opt+left"},
		{"@", "cmd+f19"},

		// Symbol order in stored preferences is not guaranteed.
		{"$^t", "ctrl+shift+t"},
		{"$@~^a", "cmd+ctrl+opt+shift+a"},

		// A stored uppercase key means Shift.
		{"@T", "cmd+shift+t"},

		// The final rune is always the key, even when it is a modifier symbol.
		{"@$", "cmd+$"},
	}

	for _, tt := range tests {
		t.Run(tt.encoded, func(t *testing.T) {
			k, err := Decode(tt.encoded)
			if err != nil {
				t.Fatalf("Decode(%q) failed: %v", tt.encoded, err)
			}
			if got := k.String(); got != tt.notation {
				t.Errorf("String() = %q, want %q", got, tt.notation)
			}
		})
	}
}

func TestDecodeErrors(t *testing.T) {
	for _, in := range []string{"", "xt", "^^t"} {
		t.Run(in, func(t *testing.T) {
			if k, err := Decode(in); err == nil {
				t.Errorf("Decode(%q) = %v, want error", in, k)
			}
		})
	}
}

// A value read back from preferences must compare equal to the same shortcut
// declared in YAML, whatever spelling either side used.
func TestParseDecodeAgree(t *testing.T) {
	tests := []struct {
		notation string
		encoded  string
	}{
		{"ctrl+t", "^t"},
		{"ctrl+shift+t", "$^t"},
		{"cmd+shift+t", "@T"},
		{"opt+left", "~"},
	}

	for _, tt := range tests {
		t.Run(tt.notation, func(t *testing.T) {
			parsed, err := Parse(tt.notation)
			if err != nil {
				t.Fatalf("Parse(%q) failed: %v", tt.notation, err)
			}
			decoded, err := Decode(tt.encoded)
			if err != nil {
				t.Fatalf("Decode(%q) failed: %v", tt.encoded, err)
			}
			if parsed != decoded {
				t.Errorf("Parse(%q) = %v, Decode(%q) = %v, want equal",
					tt.notation, parsed, tt.encoded, decoded)
			}
		})
	}
}

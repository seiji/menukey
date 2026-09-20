// Package keyspec converts between the friendly key notation used in menukey
// YAML files ("ctrl+shift+t") and the key equivalent strings stored in the
// macOS NSUserKeyEquivalents preference ("^$t").
package keyspec

import (
	"fmt"
	"sort"
	"strings"
)

// Modifier is a bitmask of the modifier keys of a key equivalent.
type Modifier uint8

const (
	ModCommand Modifier = 1 << iota
	ModControl
	ModOption
	ModShift
)

// Key is a parsed key equivalent: a set of modifiers plus a single key.
type Key struct {
	Mods Modifier
	// Code is the base key. Latin letters are stored lowercase; special keys
	// use the Cocoa function key code points (NSUpArrowFunctionKey and friends).
	Code rune
}

// modifiers lists the modifiers in canonical output order. Encode and String
// both walk this slice, so a Key always renders deterministically.
var modifiers = []struct {
	mod    Modifier
	symbol rune
	name   string
	// aliases are additional names accepted by Parse.
	aliases []string
}{
	{ModCommand, '@', "cmd", []string{"command", "meta", "super"}},
	{ModControl, '^', "ctrl", []string{"control"}},
	{ModOption, '~', "opt", []string{"option", "alt"}},
	{ModShift, '$', "shift", nil},
}

// specialKeys maps friendly names to the code points Cocoa uses for keys that
// have no printable character. The function key range starts at
// NSUpArrowFunctionKey (0xF700).
var specialKeys = map[string]rune{
	"up":        0xF700, // NSUpArrowFunctionKey
	"down":      0xF701, // NSDownArrowFunctionKey
	"left":      0xF702, // NSLeftArrowFunctionKey
	"right":     0xF703, // NSRightArrowFunctionKey
	"delete":    0xF728, // NSDeleteFunctionKey (forward delete)
	"home":      0xF729, // NSHomeFunctionKey
	"end":       0xF72B, // NSEndFunctionKey
	"pageup":    0xF72C, // NSPageUpFunctionKey
	"pagedown":  0xF72D, // NSPageDownFunctionKey
	"help":      0xF746, // NSHelpFunctionKey
	"backspace": '\b',
	"tab":       '\t',
	"return":    '\r',
	"escape":    0x1B,
	"space":     ' ',
}

// specialKeyNames is the reverse of specialKeys, built once at init time.
var specialKeyNames = map[rune]string{}

// keyAliases are alternate spellings accepted by Parse but never emitted.
var keyAliases = map[string]string{
	"enter":  "return",
	"esc":    "escape",
	"pgup":   "pageup",
	"pgdn":   "pagedown",
	"pgdown": "pagedown",
	"del":    "delete",
	"bs":     "backspace",
	"spc":    "space",
}

const (
	fnKeyBase  = 0xF704 // NSF1FunctionKey
	fnKeyCount = 19     // f1 through f19
)

func init() {
	for name, code := range specialKeys {
		specialKeyNames[code] = name
	}
	for i := 1; i <= fnKeyCount; i++ {
		specialKeyNames[fnKeyBase+rune(i-1)] = fmt.Sprintf("f%d", i)
	}
}

// Parse reads the friendly notation used in YAML, such as "ctrl+shift+t" or
// "cmd+option+left". Modifiers may appear in any order. A bare uppercase Latin
// letter implies Shift, matching how Cocoa interprets key equivalents, so
// "ctrl+T" and "ctrl+shift+t" are the same key.
func Parse(s string) (Key, error) {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return Key{}, fmt.Errorf("empty key")
	}

	tokens := splitTokens(trimmed)
	var k Key
	for i, tok := range tokens {
		last := i == len(tokens)-1
		mod, ok := lookupModifier(strings.ToLower(tok))
		switch {
		case ok && last && len(tokens) > 1:
			// A trailing modifier name with nothing after it is a typo.
			return Key{}, fmt.Errorf("missing key after modifier %q in %q", tok, s)
		case ok && !last:
			if k.Mods&mod != 0 {
				return Key{}, fmt.Errorf("duplicate modifier %q in %q", tok, s)
			}
			k.Mods |= mod
			continue
		case last:
			code, mods, err := parseKeyToken(tok)
			if err != nil {
				return Key{}, fmt.Errorf("%w in %q", err, s)
			}
			k.Mods |= mods
			k.Code = code
		default:
			return Key{}, fmt.Errorf("unknown modifier %q in %q", tok, s)
		}
	}
	return k, nil
}

// splitTokens splits on "+" while keeping a literal "+" key intact, so that
// "cmd++" yields ["cmd", "+"]. A single trailing "+" ("ctrl+") is left as an
// empty token, which Parse rejects.
func splitTokens(s string) []string {
	if s == "+" {
		return []string{"+"}
	}
	if rest, ok := strings.CutSuffix(s, "++"); ok {
		return append(trimAll(strings.Split(rest, "+")), "+")
	}
	return trimAll(strings.Split(s, "+"))
}

func trimAll(parts []string) []string {
	tokens := make([]string, 0, len(parts)+1)
	for _, p := range parts {
		tokens = append(tokens, strings.TrimSpace(p))
	}
	return tokens
}

func lookupModifier(name string) (Modifier, bool) {
	for _, m := range modifiers {
		if name == m.name {
			return m.mod, true
		}
		for _, alias := range m.aliases {
			if name == alias {
				return m.mod, true
			}
		}
	}
	return 0, false
}

// parseKeyToken resolves the base key of a notation string. It returns any
// modifier implied by the key itself (an uppercase letter implies Shift).
func parseKeyToken(tok string) (rune, Modifier, error) {
	if tok == "" {
		return 0, 0, fmt.Errorf("empty key")
	}

	lower := strings.ToLower(tok)
	if canonical, ok := keyAliases[lower]; ok {
		lower = canonical
	}
	if code, ok := specialKeys[lower]; ok {
		return code, 0, nil
	}
	if code, ok := parseFunctionKey(lower); ok {
		return code, 0, nil
	}

	runes := []rune(tok)
	if len(runes) != 1 {
		return 0, 0, fmt.Errorf("unknown key %q", tok)
	}
	r := runes[0]
	if r >= 'A' && r <= 'Z' {
		return r + ('a' - 'A'), ModShift, nil
	}
	return r, 0, nil
}

func parseFunctionKey(name string) (rune, bool) {
	if len(name) < 2 || name[0] != 'f' {
		return 0, false
	}
	n := 0
	for _, c := range name[1:] {
		if c < '0' || c > '9' {
			return 0, false
		}
		n = n*10 + int(c-'0')
	}
	if n < 1 || n > fnKeyCount {
		return 0, false
	}
	return fnKeyBase + rune(n-1), true
}

// Decode reads a raw NSUserKeyEquivalents value such as "^$t". Modifier symbols
// may appear in any order; the final rune is always the base key, so a key
// equivalent whose key is itself a modifier symbol ("@$" meaning Command+$)
// round-trips correctly.
func Decode(s string) (Key, error) {
	runes := []rune(s)
	if len(runes) == 0 {
		return Key{}, fmt.Errorf("empty key equivalent")
	}

	var k Key
	for _, r := range runes[:len(runes)-1] {
		mod, ok := lookupSymbol(r)
		if !ok {
			return Key{}, fmt.Errorf("unknown modifier symbol %q in %q", string(r), s)
		}
		if k.Mods&mod != 0 {
			return Key{}, fmt.Errorf("duplicate modifier symbol %q in %q", string(r), s)
		}
		k.Mods |= mod
	}

	k.Code = runes[len(runes)-1]
	if k.Code >= 'A' && k.Code <= 'Z' {
		k.Code += 'a' - 'A'
		k.Mods |= ModShift
	}
	return k, nil
}

func lookupSymbol(r rune) (Modifier, bool) {
	for _, m := range modifiers {
		if r == m.symbol {
			return m.mod, true
		}
	}
	return 0, false
}

// Encode renders the key in the form stored in NSUserKeyEquivalents. Shift is
// always written as "$" rather than by uppercasing the key, and modifiers are
// emitted in a fixed order so that the output is stable across runs.
func (k Key) Encode() string {
	var b strings.Builder
	for _, m := range modifiers {
		if k.Mods&m.mod != 0 {
			b.WriteRune(m.symbol)
		}
	}
	b.WriteRune(k.Code)
	return b.String()
}

// String renders the canonical friendly notation, the form used in status output
// and in YAML written by import.
func (k Key) String() string {
	parts := make([]string, 0, 5)
	for _, m := range modifiers {
		if k.Mods&m.mod != 0 {
			parts = append(parts, m.name)
		}
	}
	parts = append(parts, k.keyName())
	return strings.Join(parts, "+")
}

func (k Key) keyName() string {
	if name, ok := specialKeyNames[k.Code]; ok {
		return name
	}
	return string(k.Code)
}

// SpecialKeyNames returns the accepted special key names in sorted order. It is
// used to build CLI help text.
func SpecialKeyNames() []string {
	names := make([]string, 0, len(specialKeys)+fnKeyCount)
	for name := range specialKeys {
		names = append(names, name)
	}
	for i := 1; i <= fnKeyCount; i++ {
		names = append(names, fmt.Sprintf("f%d", i))
	}
	sort.Strings(names)
	return names
}

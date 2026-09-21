# menukey — Design

## Overview

`menukey` is a CLI for declaratively managing macOS App Shortcuts.

The initial use case is Google Chrome:

- Replace `Command`-based shortcuts with `Control`-based shortcuts
- Manage shortcut mappings as YAML
- Apply the same mappings across multiple Macs
- Avoid OS-wide key remapping such as Karabiner-Elements

macOS application shortcuts are backed by the application's `NSUserKeyEquivalents` preference entries, so `menukey` manages those settings declaratively.

## Goals

- Manage per-application macOS menu shortcuts from YAML
- Keep shortcut configuration in Git
- Apply identical mappings to multiple Macs
- Support importing existing shortcut settings
- Show changes before applying them
- Start with Google Chrome and expand to arbitrary macOS applications later

## Non-Goals

For the initial version:

- No GUI
- No OS-wide key remapping
- No low-level event interception
- No Chrome extension
- No automatic menu discovery
- No preset library

## YAML Specification

```yaml
version: 1

apps:
  - bundle: com.google.Chrome
    name: Google Chrome
    shortcuts:
      - menu: New Tab
        key: ctrl+t
```

`bundle` was chosen over `bundle_id` and `app`: it is the shortest spelling that still says what the value is. `name` is optional and affects status output only.

Unknown fields are rejected, so a misspelled key is an error rather than a silently ignored shortcut. `set` and `import` sort shortcuts by `menu` so generated changes have a stable order.

## Key Encoding

`NSUserKeyEquivalents` values are a run of modifier symbols followed by a single key.

| Symbol | Modifier |
|---|---|
| `@` | Command |
| `^` | Control |
| `~` | Option |
| `$` | Shift |

Decisions:

- **Parsing is order independent, encoding is not.** Symbols are emitted in the fixed order `@ ^ ~ $` so that output is stable across runs.
- **Shift is encoded as `$`, never by uppercasing the key.** Both forms are accepted on the way in, because macOS writes either one depending on how the shortcut was set.
- **Comparison happens on the parsed form.** `@$t`, `$@t` and `@T` are one shortcut, not three, so `status` does not report noise.
- **Special keys use the Cocoa function key code points** (`NSUpArrowFunctionKey` = U+F700 and neighbours). Escape, Tab, Return, Backspace and Space use their control characters.

## CLI

The normal workflow uses `~/.config/menukey/config.yaml` (or
`$XDG_CONFIG_HOME/menukey/config.yaml`) as the source of truth. This makes
`git pull && menukey sync` sufficient to synchronize a Mac.

| Command | Behaviour |
|---|---|
| `sync [file]` | Print pending changes, then write additions and updates |
| `status [file]` | Print pending changes only; `--check` exits 1 when changes are pending |
| `add <bundle-id-or-app-path>` | Add an application to the configuration |
| `search <query>` | Find installed apps and print their name, bundle ID, and path |
| `set <bundle-id> <menu> <key>` | Add or update a menu shortcut |
| `unset <bundle-id> [menu]` | Remove an application declaration, or one menu shortcut declaration |
| `import <bundle-id-or-app-path>` | Read current entries into the configuration; `--replace` is required for an existing app |

`sync` and `status` accept either the optional file argument or `--config`; all
other configuration commands use `--config` when the default file is not
wanted. Any command that uses a missing configuration creates an empty file
automatically; this lets the CLI start with its default state without an
explicit initialization step.

`add /Applications/Example.app` is the recommended interactive workflow:
menukey reads the app bundle's `Info.plist` to obtain its bundle ID and display
name. `search <query>` uses Spotlight to find paths when needed; direct bundle
IDs remain supported for scripts.

`import` verifies that the app exists. A `.app` path is checked directly; a
bundle ID is resolved through Spotlight. It then reads every entry of the
domain, not only entries menukey wrote: there is no ownership tracking, so it
cannot tell them apart. An entry that cannot be decoded is reported on stderr
and skipped rather than being written out in a form `sync` would later reject.

## Implementation

`defaults(1)` is the shortcut backend, reached through `internal/prefs.Backend`.
Application discovery is isolated in `internal/appinfo`: `plutil` reads an app
bundle's `Info.plist`, and `mdfind` performs name searches.

Shortcut backend details:

- **Read** runs `defaults export <bundle-id> -` and pulls the `NSUserKeyEquivalents` entry out of the resulting XML property list. Converting the whole domain first was the original plan, but `plutil -convert json` fails outright on real domains — Chrome's holds values with no JSON equivalent. Parsing only the one entry sidesteps every other value's type.
- **Write** runs `defaults write <bundle-id> NSUserKeyEquivalents -dict-add <menu> <value>`. Passing menu titles as arguments avoids having to build and escape a property list, and `-dict-add` gives merge semantics for free.

Going through `defaults` rather than writing the plist file directly keeps `cfprefsd` in the loop, which is what makes the change visible to the application.

`Backend` is an interface so that a CoreFoundation implementation can replace this one later, and so that tests never touch machine state. `MemoryBackend` is the test double.

## Configuration Ownership

`sync` merges: it only adds and updates the shortcuts declared in YAML, and leaves everything else in the domain alone. Removed YAML entries therefore stay configured on the system.

The alternative, treating YAML as the source of truth and deleting anything else, needs ownership tracking to avoid destroying shortcuts the user set by hand. `unset` therefore changes only the configuration. Destructive reconciliation is deferred to a future `--reconcile` command.

## Locale Considerations

`NSUserKeyEquivalents` is keyed by menu item title, which differs between application locales. For v0.1, literal menu titles in YAML are sufficient.

A later version could introduce logical action names with a locale mapping:

```yaml
shortcuts:
  - action: new-tab
    key: ctrl+t
```

```yaml
new-tab:
  en: "New Tab"
  ja: "新しいタブ"
```

## Repository Structure

```text
menukey/
├── main.go
├── cmd/               cobra commands: sync, status, and configuration management
├── internal/
│   ├── config/        YAML loading and validation
│   ├── keyspec/       "ctrl+shift+t" ⇄ "^$t"
│   ├── plan/          desired vs current, shared by status and sync
│   └── prefs/         Backend interface, defaults(1) implementation, plist parsing
└── examples/chrome.yaml
```

## v0.1 Scope

Supported: macOS, `sync`, `status`, configuration management commands, import, and the three Chrome shortcuts (New Tab, Close Tab, Reopen Closed Tab). Nothing about the implementation is Chrome specific; Chrome is only what the example configuration targets.

Deferred: automatic menu scanning (`scan`), locale abstraction, `--reconcile`, presets, GUI, other platforms, global key remapping.

## Resolved Questions

- **What is Chrome's menu title for "Reopen Closed Tab"?** `Reopen Closed Tab`, confirmed against the English string table in Chrome 151's `locale.pak`.
- **Does Chrome reload `NSUserKeyEquivalents` immediately?** No. Menus are built when the application launches, so it has to be restarted. `sync` says so after writing.
- **How should modifiers be encoded?** See *Key Encoding* above.
- **Should `import` capture every entry, or only menukey-managed entries?** Every entry; there is no ownership tracking to filter by.
- **Should the YAML key be `bundle`, `bundle_id` or `app`?** `bundle`.
- **Should duplicate key assignments be rejected?** Yes, at validation time. Two menu items sharing a key equivalent means one of them silently never fires.

## Open Questions

- Should `scan` drive AppleScript/System Events to enumerate an application's menus, which needs Accessibility permission, or read the application's own string tables?
- Should `--reconcile` track ownership in a separate preference entry, or in a state file alongside the configuration?

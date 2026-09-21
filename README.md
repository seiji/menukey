# menukey

Declaratively manage macOS App Shortcuts.

macOS lets you override any application's menu shortcut through *System Settings > Keyboard > Keyboard Shortcuts > App Shortcuts*. Those overrides are stored in the application's `NSUserKeyEquivalents` preference. `menukey` manages that preference from a file you can keep in Git and apply on every Mac you use.

The initial use case is making Google Chrome use Control-based tab shortcuts, without remapping the whole operating system the way Karabiner-Elements would.

## Installation

```
go install github.com/seiji/menukey@latest
```

Or build from a checkout:

```
git clone https://github.com/seiji/menukey.git
cd menukey
make build
```

## Usage

The default configuration is `~/.config/menukey/config.yaml` (or
`$XDG_CONFIG_HOME/menukey/config.yaml`). It is created automatically on first
use. Keep that file in Git to use the same shortcuts on every Mac.

```bash
# Describe the desired state; this creates the default configuration.
# Pass the application bundle path so menukey finds its name and bundle ID.
menukey add "/Applications/Google Chrome.app"
menukey set com.google.Chrome "New Tab" ctrl+t
menukey set com.google.Chrome "Close Tab" ctrl+w

# Inspect and synchronize every configured application on this Mac
menukey status
menukey sync

# CI-friendly status check: exit 1 if synchronization is needed
menukey status --check

# Capture an application's current shortcuts into the configuration.
# --replace is needed because Chrome is already configured above.
menukey import "/Applications/Google Chrome.app" --replace

```

`sync` merges: menu items the configuration does not mention keep their current
shortcut. Applications read `NSUserKeyEquivalents` when they build their menus,
so restart affected applications after synchronizing.

Pass `--config <file>` to any command to use another configuration. `sync` and
`status` also accept the file as an optional positional argument, for example
`menukey sync examples/chrome.yaml`.

### Commands and flags

| Command | Purpose | Useful flags |
|---|---|---|
| `sync` | Apply all declared shortcuts to this Mac | `--dry-run`, `--app <bundle-id>` |
| `status` | Show pending changes without writing | `--check`, `--all`, `--unmanaged`, `--app <bundle-id>` |
| `add` | Add an application to the configuration | `add <path-to-app>`; `--name <name>` overrides the detected name |
| `search` | Find installed applications and display their bundle IDs | — |
| `set` | Add or update a configured shortcut | — |
| `unset` | Remove an application or shortcut declaration | `unset <bundle-id> [menu]` |
| `import` | Import an application's current shortcuts into the configuration | Pass a `.app` path; `--name <name>`, `--replace` |


### Adding applications

The recommended way to add an application is its `.app` path. menukey reads
its `Info.plist` to record the exact bundle ID and display name.

```bash
menukey add "/Applications/Google Chrome.app"
```

Use `menukey search chrome` when the path is unknown. Supplying a bundle ID
(`menukey add com.google.Chrome --name Chrome`) remains useful for scripts.

`import` also accepts an `.app` path and verifies that the application exists.
A bundle ID is accepted only when it resolves to an installed application.

## Configuration

```yaml
version: 1

apps:
  - bundle: com.google.Chrome
    name: Google Chrome        # optional, used in status output only
    shortcuts:
      - menu: New Tab
        key: ctrl+t

      - menu: Close Tab
        key: ctrl+w

      - menu: Reopen Closed Tab
        key: ctrl+shift+t
```

`menu` must be the menu item title exactly as the application displays it, because that title is what `NSUserKeyEquivalents` is keyed by. Titles are locale specific: an application running in Japanese needs the Japanese title.

Two menu items of the same application may not be given the same key, and a menu item may not be declared twice; both are rejected before anything is written.

### Key notation

Modifiers, in any order, joined with `+`:

| Modifier | Accepted spellings |
|---|---|
| Command | `cmd`, `command`, `meta`, `super` |
| Control | `ctrl`, `control` |
| Option | `opt`, `option`, `alt` |
| Shift | `shift` |

The key itself is either a single character (`t`, `/`, `+`) or one of `up`, `down`, `left`, `right`, `home`, `end`, `pageup`, `pagedown`, `delete`, `backspace`, `tab`, `return`, `escape`, `space`, `help`, `f1`–`f19`. An uppercase letter implies Shift, so `ctrl+T` and `ctrl+shift+t` are the same shortcut.

## How it works

`menukey` reads the preference domain with `defaults export <bundle-id> -` and writes with `defaults write <bundle-id> NSUserKeyEquivalents -dict-add ...`, so changes go through `cfprefsd` and stay consistent with what the preference system has cached.

Reading parses only the `NSUserKeyEquivalents` entry out of the exported property list. Converting the whole domain first is not viable: real application domains hold values that `plutil -convert json` refuses outright.

Shortcuts are compared in parsed form, so a stored `@$t` and a declared `cmd+shift+t` count as equal even though macOS may have written the modifiers in a different order, or spelled Shift by uppercasing the key.

## Limitations

- macOS only.
- `sync` never removes entries. `unset` only removes declarations from the configuration; it does not change macOS. Delete an individual shortcut through System Settings. `defaults delete <bundle-id> NSUserKeyEquivalents` removes **all** app-specific shortcuts for that bundle.
- Menu titles must be written out literally, and are locale specific.
- There is no menu discovery: you have to know the exact title of the menu item.

## Development

```bash
make test              # unit tests, no machine state touched
make test-integration  # exercises defaults(1) against a scratch preference domain
make build
```

## License

[MIT](LICENSE)

# Zsh Keybinding Design

`hi-shell` uses zsh ZLE widgets for the interactive shell flow. The keybinding design keeps the main shell keymap small and moves hi-shell-specific commands behind one configurable prefix key.

## Goals

- Keep Enter and Tab fast for the primary generation and accept flow.
- Avoid adding one global shortcut per feature.
- Avoid occupying zsh's built-in `^X...` prefix space.
- Let users choose the prefix key from `~/.hi-shell/config.toml`.
- Keep TOML parsing and defaulting in Go, not in zsh.

## Public Contract

The zsh plugin binds only these keys in the user's main keymap:

| Key | Behavior |
| --- | --- |
| Enter | Generate when the buffer starts with `hi `; otherwise delegate to the previous Enter widget. |
| Tab | Accept a visible suggestion; otherwise delegate to the previous Tab widget. |
| `keybindings.prefix` | Enter hi-shell's prefix keymap when a suggestion exists; otherwise delegate to the previous widget for that key. |

The default prefix is Ctrl-]:

```toml
[keybindings]
prefix = "^]"
```

Prefix key values use zsh `bindkey` notation. Examples:

```sh
hi-shell config set keybindings.prefix '^]'
hi-shell config set keybindings.prefix '^[;'
```

## Prefix Keymap

The plugin creates a dedicated `hi-shell-prefix` keymap. This keymap currently has two commands:

| Key | Behavior |
| --- | --- |
| `r` | Toggle revise mode for the current suggestion. |
| `q` | Exit prefix mode without changing the suggestion. |

Unknown prefix keys leave prefix mode and show a short message.

Canceling a shell input line remains a shell concern. The plugin does not bind Ctrl-C or Ctrl-G. zsh's normal line finish and prompt hooks clear hi-shell's transient state.

## Config Loading

The zsh plugin does not parse TOML. It calls the Go CLI:

```zsh
_hi_config_get keybindings.prefix '^]'
_hi_config_get history.fetch_limit 20
```

`_hi_config_get` is a generic wrapper around:

```sh
hi-shell config get <key>
```

The second argument is the shell-side fallback used only when the CLI is unavailable or returns no value. The authoritative default still lives in Go's `config.Default()`.

This keeps config loading in one place:

- Go owns TOML parsing.
- Go owns validation and defaults.
- zsh only consumes the final scalar value.

## Widget Compatibility

Before installing hi-shell widgets, the plugin records the user's previous Enter, line-feed, Tab, and prefix widgets:

```zsh
_HI_ENTER_WIDGET="$(_hi_bound_widget '^M')"
_HI_LINEFEED_WIDGET="$(_hi_bound_widget '^J')"
_HI_TAB_WIDGET="$(_hi_bound_widget '^I')"
_HI_PREFIX_WIDGET="$(_hi_bound_widget "$_HI_PREFIX_KEY")"
```

When hi-shell has no relevant state for a key, it delegates back through `_hi_call_widget`.

This matters for compatibility with completion frameworks and custom shell bindings. For example, Tab accepts hi-shell ghost text only when a suggestion is visible; otherwise the user's previous completion widget still runs.

## Revise Flow

Revise mode is entered with prefix then `r`. While revise mode is active:

- User text in `BUFFER` is treated as feedback.
- Enter sends a revision session to `hi-shell revise`.
- An empty feedback buffer shows a message instead of calling the LLM.
- Ctrl-C remains the normal zsh way to abandon the current line.

The plugin keeps the in-memory revision session as JSON strings in `_HI_TURNS`. The Go CLI validates and limits the session data before using it in prompts.

## Extension Rules

Future zsh-only actions should be added to `hi-shell-prefix` instead of the main keymap.

Add a new prefix action only when it does real work that cannot be handled by normal shell editing. Avoid aliases that only repeat existing messages or duplicate shell-native controls.

Good candidates:

- `e`: explain or inspect the current suggestion.
- `s`: show detailed risk information.
- `y`: copy the current suggestion.

Avoid:

- New global shortcuts for hi-shell-only actions.
- Shell-native cancel bindings such as Ctrl-C or Ctrl-G.
- Redundant help bindings that only repeat the prefix entry message.

## Tests

Current automated coverage:

- `zsh -n shell/hi.zsh` verifies plugin syntax.
- Config tests cover `keybindings.prefix` defaults, save/load, set, and get.
- CLI integration tests continue to cover generation, revision, and risk behavior.

Manual keymap smoke test:

```sh
tmp=$(mktemp -d)
PATH="$PWD:$PATH" HI_SHELL_HOME="$tmp" zsh -fc '
  source shell/hi.zsh
  bindkey "^]"
  bindkey -M hi-shell-prefix r
  bindkey -M hi-shell-prefix q
'
rm -rf "$tmp"
```

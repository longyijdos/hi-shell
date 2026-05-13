# hi-shell

Tiny AI command composition for zsh.

Type `hi ...`, press Enter, review the generated command, press Tab to accept it, then press Enter to run it yourself.

```zsh
$ hi list go files
# suggestion appears as ghost text:
$ find . -name "*.go"
```

`hi-shell` is not an agent, terminal emulator, or chat UI. It only turns natural language into a shell command and inserts that command into your current zsh input line for review.

## Status

Early MVP. The current build supports zsh on Linux/macOS-style environments and hosted OpenAI-compatible APIs. DeepSeek has a dedicated fast path because command generation is latency-sensitive.

## Features

- Natural-language command generation from your shell.
- zsh ghost-text flow using ZLE widgets.
- Tab accepts the suggestion; Enter runs only after you accept.
- `hi ...` prompts do not enter shell history.
- Safe/warn/blocked command risk scoring.
- Clearly catastrophic commands are blocked by default.
- Config lives under `~/.hi-shell/config.toml`.
- Secrets stay in environment variables, not config files.
- Clean install and uninstall with one managed `.zshrc` block.

## Install

```sh
curl -fsSL https://raw.githubusercontent.com/longyijdos/hi-shell/main/scripts/install.sh | sh
exec zsh
```

Make sure `~/.local/bin` is in your `PATH`.

## Install From Source

Prerequisites:

- Go 1.22+
- git
- zsh

```sh
git clone https://github.com/longyijdos/hi-shell.git
cd hi-shell
sh ./scripts/install.sh
exec zsh
```

The installer builds `hi-shell` from source into `~/.local/bin/hi-shell`, installs the zsh plugin under `~/.hi-shell/shell/hi.zsh`, creates a default config, and adds this managed block to `~/.zshrc`:

```zsh
# >>> hi-shell initialize >>>
source "$HOME/.hi-shell/shell/hi.zsh"
# <<< hi-shell initialize <<<
```

## Configure

OpenAI is the default provider:

```sh
export OPENAI_API_KEY="sk-..."

hi-shell config set provider openai
hi-shell config set openai.api_key_env OPENAI_API_KEY
hi-shell config set openai.model gpt-4.1-mini
```

DeepSeek is also supported and has a dedicated low-latency path:

```sh
export DEEPSEEK_API_KEY="sk-..."

hi-shell config set provider deepseek
hi-shell config set deepseek.api_key_env DEEPSEEK_API_KEY
hi-shell config set deepseek.model deepseek-v4-flash
```

View the active config:

```sh
hi-shell config get
```

Example config:

```toml
provider = "openai"
timeout_ms = 5000

[openai]
base_url = "https://api.openai.com/v1"
api_key_env = "OPENAI_API_KEY"
model = "gpt-4.1-mini"

[deepseek]
base_url = "https://api.deepseek.com/v1"
api_key_env = "DEEPSEEK_API_KEY"
model = "deepseek-v4-flash"
thinking = "disabled"
max_tokens = 256

[keybindings]
prefix = "^]"

[context]
pwd = true
os = true
shell = true
git = true
project_files = true
package_scripts = true
history = false

[history]
fetch_limit = 20
max_entries = 12
max_command_chars = 240
max_bytes = 2000

[session]
revise_turns = 8
ask_turns = 8
max_field_chars = 4000
max_json_bytes = 65536

[safety]
block_critical = true
warn_sudo = true
warn_destructive = true
```

Set `context.history = true` to include a filtered snapshot of recent shell commands in generation and revision prompts. The zsh plugin passes recent history to `hi-shell` through `HI_SHELL_HISTORY`; `hi-shell` ignores it unless this setting is enabled, then drops obvious secrets, `hi`/`hi-shell` commands, duplicates, and overly long entries. The `[history]` section controls how many commands the plugin fetches and how much filtered history Go keeps.

The `[session]` section controls how many revise and ask turns are kept in the current suggestion session, plus validation limits for session JSON passed by integrations.

Set `keybindings.prefix` to change the hi-shell prefix key used by the zsh plugin. The value uses zsh `bindkey` notation; the default `^]` means Ctrl-].

## Usage

Interactive zsh flow:

```zsh
hi list all go files
```

Press Enter to generate a suggestion. Press Tab to accept it into your input buffer. Press Enter again to run it.

`hi` is only the zsh natural-language prefix. Use `hi-shell` for management commands such as config, diagnostics, install, and uninstall.

Keyboard behavior:

| Key | State | Behavior |
| --- | --- | --- |
| Enter | buffer starts with `hi ` | Generate a command suggestion |
| Enter | revise mode | Revise the suggestion with session history |
| Enter | ask mode | Ask a question about the current suggestion |
| Enter | suggestion exists, edit mode | Run current `BUFFER` normally |
| Enter | normal shell input | Run normal zsh `accept-line` |
| Tab | suggestion visible | Accept suggestion into `BUFFER` |
| Tab | no suggestion | Use normal zsh completion |
| Text input | suggestion visible | Hide the ghost text and edit normally |
| Prefix, then `e` | suggestion exists | Switch to edit mode |
| Prefix, then `r` | suggestion exists | Switch to revise mode |
| Prefix, then `a` | suggestion exists | Switch to ask mode |
| Prefix, then `q` | prefix mode | Exit prefix mode |

CLI usage:

```sh
hi-shell generate --prompt "list go files" --format json
hi-shell revise --session-json - --format json
hi-shell ask --session-json - --format json
hi-shell risk --command 'rm -rf /' --format json
hi-shell doctor
hi-shell version
```

`hi-shell generate --format json` returns:

```json
{
  "command": "find . -name \"*.go\"",
  "risk": "safe",
  "warning": ""
}
```

Inspect local risk scoring without calling an LLM:

```sh
hi-shell risk --command 'rm -rf /'
hi-shell risk --command 'find . -name "node_modules" -type d -exec rm -rf {} +' --format json
```

`hi-shell revise` is for integrations that keep an in-memory command revision session. `--session-json` accepts inline JSON, `-` for stdin, or `@path/to/session.json`:

```json
{
  "initial_prompt": "list large files",
  "turns": [
    {
      "command": "find . -type f -size +100M",
      "risk": "safe",
      "warning": "",
      "feedback": "sort by size and show human readable sizes"
    }
  ]
}
```

`hi-shell ask` is for integrations that keep an in-memory command question session. It answers questions about the current suggestion and does not generate a replacement command:

```json
{
  "initial_prompt": "list large files",
  "turns": [
    {
      "command": "find . -type f -size +100M",
      "risk": "safe",
      "warning": "",
      "question": "will this modify files?"
    }
  ]
}
```

`hi-shell ask --format json` returns:

```json
{
  "answer": "No. This command only searches for matching files and prints their paths."
}
```

## Safety Model

`hi-shell` never runs generated commands automatically.

The shell plugin only inserts a suggestion into the current command line after you press Tab. You still review the final command and press Enter yourself.

Risk scoring is intentionally conservative:

- `safe`: no warning.
- `warn`: show a warning, but still allow the user to accept the suggestion.
- `blocked`: do not return an executable command by default.

Clearly catastrophic commands such as `rm -rf /` or `chmod -R 777 /` are blocked by default.

## Uninstall

```sh
hi-shell uninstall
exec zsh
```

This removes the managed `.zshrc` block and installed binary/plugin while preserving `~/.hi-shell/config.toml`.

To remove all hi-shell files:

```sh
hi-shell uninstall --purge
exec zsh
```

## Development

```sh
go test ./...
go vet ./...
go build -o hi-shell ./cmd/hi-shell
zsh -n shell/hi.zsh
sh -n scripts/install.sh
```

Test the plugin without touching `~/.zshrc`:

```sh
go build -o hi-shell ./cmd/hi-shell
export PATH="$PWD:$PATH"
zsh -f
source ./shell/hi.zsh
```

## Release

Create and push a version tag:

```sh
git tag vX.Y.Z
git push origin vX.Y.Z
```

The release workflow builds Linux/macOS archives for arm64 and x86_64, generates `checksums.txt`, and uploads them to GitHub Releases. The curl installer downloads the matching archive and verifies its checksum before installing.

## Scope

MVP scope:

- zsh only.
- Hosted providers only.
- No local model provider yet.
- No autonomous execution.
- No multi-step agent loop.

Local model support, bash/fish plugins, and richer safety UX can come after the zsh workflow is solid.

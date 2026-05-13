# Configuration

`hi-shell` stores config in:

```sh
~/.hi-shell/config.toml
```

Secrets should stay in environment variables. The config stores the environment variable name, not the secret value.

## Providers

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

## Example

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

## Shell History

Set `context.history = true` to include a filtered snapshot of recent shell commands in generation and revision prompts.

The zsh plugin passes recent history to `hi-shell` through `HI_SHELL_HISTORY`. `hi-shell` ignores it unless this setting is enabled, then drops obvious secrets, `hi`/`hi-shell` commands, duplicates, and overly long entries.

The `[history]` section controls how many commands the plugin fetches and how much filtered history Go keeps.

## Sessions

The `[session]` section controls how many revise and ask turns are kept in the current suggestion session, plus validation limits for session JSON passed by integrations.

## Keybindings

Set `keybindings.prefix` to change the hi-shell prefix key used by the zsh plugin. The value uses zsh `bindkey` notation; the default `^]` means Ctrl-].

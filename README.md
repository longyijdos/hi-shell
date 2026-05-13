<h1 align="center">hi-shell</h1>

<p align="center">
  <img src="assets/logo.png" alt="hi-shell logo" width="120">
</p>

<p align="center">
  AI-powered zsh command generation with reviewable, risk-scored shell suggestions.
</p>

![hi-shell demo](assets/demo.gif)

`hi-shell` turns natural language into shell commands and shows the result as ghost text in your current zsh line. Press Tab to accept the suggestion, edit it if needed, then press Enter yourself.

It is not an autonomous shell agent, terminal emulator, or chat UI. It keeps command generation inside your normal terminal workflow while leaving execution under your control.

## ✨ Why hi-shell

Shell commands are often easy to describe and annoying to spell out. `hi-shell` handles the translation without taking over the terminal.

- Stay in zsh instead of switching to a chat window.
- Review every generated command before it can run.
- Accept suggestions into the real command line, not a separate UI.
- Revise a suggestion or ask what it does before running it.
- Block clearly catastrophic commands by default.

## ⚡ Features

- Natural-language command generation from your shell.
- Ghost-text suggestions powered by zsh ZLE widgets.
- Tab-to-accept flow; Enter only runs after you accept.
- `safe`, `warn`, and `blocked` risk scoring.
- Built-in support for OpenAI and DeepSeek APIs.
- OpenAI-compatible provider configuration.
- Optional shell context and filtered history.
- Clean install and uninstall with one managed `.zshrc` block.

## 🚀 Install

```sh
curl -fsSL https://raw.githubusercontent.com/longyijdos/hi-shell/main/scripts/install.sh | sh
exec zsh
```

Make sure `~/.local/bin` is in your `PATH`.

From source:

```sh
git clone https://github.com/longyijdos/hi-shell.git
cd hi-shell
./scripts/install.sh
exec zsh
```

Prerequisites for source installs: Go 1.22+, git, and zsh.

## ⚙️ Configure

OpenAI is the default provider:

```sh
export OPENAI_API_KEY="sk-..."

hi-shell config set provider openai
hi-shell config set openai.api_key_env OPENAI_API_KEY
hi-shell config set openai.model gpt-4.1-mini
```

DeepSeek is also supported:

```sh
export DEEPSEEK_API_KEY="sk-..."

hi-shell config set provider deepseek
hi-shell config set deepseek.api_key_env DEEPSEEK_API_KEY
hi-shell config set deepseek.model deepseek-v4-flash
```

Secrets stay in environment variables. The config file stores environment variable names, not API keys.

See [Configuration](docs/configuration.md) for the full config reference.

## 🧭 Use

Type a natural-language request with the `hi` prefix:

```zsh
hi list go files
```

Press Enter to generate a suggestion. If the command looks right, press Tab to accept it into your shell input line. Press Enter again to run it.

Use the prefix key, Ctrl-] by default, to revise, ask about, edit, or dismiss the current suggestion. See [Usage](docs/usage.md) and [Keybindings](docs/zsh-keybindings.md) for details.

## 🛡️ Safety

`hi-shell` never runs generated commands automatically.

The shell plugin only inserts a suggestion after you accept it. You still review the final command and press Enter yourself. Local risk scoring classifies suggestions as `safe`, `warn`, or `blocked`; clearly catastrophic commands such as `rm -rf /` are blocked by default.

See [Risk Scoring](docs/risk-scoring.md) for the detailed model.

## 📚 Documentation

- [Configuration](docs/configuration.md)
- [Usage and CLI](docs/usage.md)
- [Keybindings](docs/zsh-keybindings.md)
- [Risk Scoring](docs/risk-scoring.md)
- [Development and Release](docs/development.md)

## 🧹 Uninstall

```sh
hi-shell uninstall
exec zsh
```

To remove all hi-shell files, including config:

```sh
hi-shell uninstall --purge
exec zsh
```

## 📄 License

MIT

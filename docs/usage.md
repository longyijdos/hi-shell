# Usage

## Interactive zsh Flow

Type a natural-language request:

```zsh
hi list all go files
```

Press Enter to generate a suggestion. Press Tab to accept it into your input buffer. Press Enter again to run it.

Before accepting a suggestion, type `hi? will this modify files?` and press Enter to ask about it, or type `hi: only search this directory` and press Enter to revise it. Both use the current suggestion and original request as context. Without a current suggestion, these prefixes show a message and leave the input available for editing. A new `hi ...` request starts a fresh suggestion without sending the previous request or suggestion to the model.

`hi` is only the zsh natural-language prefix. Use `hi-shell` for management commands such as config, diagnostics, install, and uninstall.

## Keyboard Behavior

| Key | State | Behavior |
| --- | --- | --- |
| Enter | buffer starts with `hi ` | Generate a new command suggestion |
| Enter | buffer starts with `hi?` | Ask about the current suggestion |
| Enter | buffer starts with `hi:` | Revise the current suggestion with feedback |
| Enter | normal shell input | Run normal zsh `accept-line` |
| Tab | suggestion visible | Accept suggestion into `BUFFER` |
| Tab | no suggestion | Use normal zsh completion |
| Text input | suggestion visible | Hide the ghost text; the suggestion remains available for `hi?` or `hi:` |
| Ctrl-C | current input line | Cancel the line and clear the suggestion session |

## CLI

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

## Revise

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

## Ask

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

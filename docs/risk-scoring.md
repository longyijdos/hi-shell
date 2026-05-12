# Risk Scoring Design

`hi-shell` suggests shell commands but never runs them automatically. Risk scoring is therefore a local warning and blocking layer, not an execution sandbox or approval engine.

The model only produces the command. Risk is decided locally by deterministic rules.

## Public Contract

`hi-shell generate --format json` and `hi-shell revise --format json` return:

```json
{
  "command": "find . -name \"*.go\"",
  "risk": "safe",
  "warning": ""
}
```

Risk values are intentionally three-state:

- `safe`: no warning.
- `warn`: show a warning, but still allow the user to accept the suggestion.
- `blocked`: do not return an executable command by default.

Blocked commands return an empty `command` field when `safety.block_critical = true`.

## User Experience

The zsh plugin shows any non-empty warning in the ZLE message area. A `warn` result can still be accepted with Tab. A `blocked` result is not inserted because the CLI returns an empty command.

Warning text should be short and specific:

```text
Risky command: uses sudo.
Risky command: recursively deletes files.
Risky command: may discard git working tree changes.
Blocked critical command: recursive delete targets /.
This command writes to files or changes project state.
```

Avoid generic warnings when a rule knows the reason.

## Parser

The scorer uses `mvdan.cc/sh/v3/syntax` to parse shell syntax into an AST. This is important because raw substring matching cannot distinguish an executable command from quoted text.

Examples:

```text
echo "rm -rf /"
=> safe

rm -rf /
=> blocked
```

The analyzer walks statements, simple commands, pipelines, redirects, subshells, shell wrappers, command substitutions, and process substitutions. For chains and pipelines, the strongest public result wins:

- any `blocked` part makes the whole command `blocked`
- otherwise any `warn` part makes the whole command `warn`
- otherwise the command is `safe`

## Rule Shape

Rules are implemented as command-specific scoring functions rather than a flat substring list. `Score(command, safety)` remains the public entry point.

Internally, each warning/block result carries a `RuleID` for tests and future explain/details UX. `RuleID` is not serialized in CLI JSON.

Unknown commands default to `safe` unless they use risky shell structure. This is intentional: hi-shell does not execute commands, and warning on every unknown tool would make warnings noisy.

## Blocked Rules

Blocked rules are narrow and reserved for obviously catastrophic commands.

Current coverage:

- Recursive delete of root or home:
  - `rm -rf /`
  - `rm -r -f /`
  - `rm --recursive --force /`
  - `rm -rf ~`
  - `rm -rf "$HOME"`
- Recursive permission or ownership changes under root or home:
  - `chmod -R 777 /`
  - `chmod -R 777 ~`
  - `chown -R user /`
  - `chown -R user ~`
- Disk/device destruction:
  - `dd ... of=/dev/...`
  - `mkfs.* /dev/...`
  - redirects directly to disk devices such as `/dev/sda`
- Fork bomb pattern:
  - `:(){ :|:& };:`

If `safety.block_critical = false`, these commands return `warn` instead of `blocked`, with a `Critical command: ...` warning.

## Warning Rules

Warning rules cover commands that may be intentional but deserve attention.

Current coverage includes:

- privilege wrappers: `sudo`, `doas`, `su -c`
- recursive or forceful deletion outside protected targets
- `find -delete`, `find -exec`, and `find` output-to-file options
- `git reset --hard`, `git clean`, `git checkout --`, `git restore`, branch deletion, force push, and common git state changes
- process or service disruption: `kill`, `pkill`, `systemctl stop|restart|disable`, `launchctl unload|bootout`
- remote or indirect shell execution: `curl ... | sh`, `cat script.sh | sh`, `eval`, `source`
- container and infrastructure changes: `docker`, `podman`, `kubectl`, `helm`, `terraform`, `pulumi`
- package and dependency changes: `apt`, `brew`, `npm`, `pnpm`, `yarn`, `pip`, `go`, `cargo`, `bundle`
- local file mutations: redirects, `tee`, `touch`, `mkdir`, `cp`, `mv`, `ln`, `sed -i`, file-writing `base64`, `curl`, `wget`, and `dd of=...`

Config toggles:

- `safety.warn_sudo = false` disables warnings caused only by privilege wrappers.
- `safety.warn_destructive = false` disables destructive warning rules where reasonable.
- Critical blocking is controlled only by `safety.block_critical`.

## Shell Wrappers

The scorer recursively analyzes simple shell wrappers:

```text
bash -lc "git status && git reset --hard"
=> warn

bash -lc "rm -rf /"
=> blocked
```

Command substitutions are also analyzed:

```text
echo "$(rm -rf /)"
=> blocked
```

Dynamic command names and complex shell syntax that cannot be analyzed confidently return `warn`.

## Tests

Risk tests are table-driven and live in `internal/risk/risk_test.go`. They cover:

- read-only safe commands
- quoted false positives
- file writes
- recursive delete variants
- root/home blocked targets
- chmod/chown blocked targets
- disk writes
- git destructive commands
- remote script execution
- shell wrappers and command substitutions
- safety config toggles

CLI tests also verify that blocked generated commands return:

```json
{
  "command": "",
  "risk": "blocked",
  "warning": "Blocked critical command: ..."
}
```

## References

- OpenAI Codex command safety: https://github.com/openai/codex/tree/main/codex-rs/shell-command/src/command_safety
- Gemini CLI shell restrictions: https://github.com/google-gemini/gemini-cli/blob/main/docs/tools/shell.md
- mvdan shell parser: https://pkg.go.dev/mvdan.cc/sh/v3/syntax

# ccinject

Deterministic context injection for Claude Code subagents. Coordinators write
one-line directives in an Agent dispatch prompt; a PreToolUse hook expands
them into the real content at spawn time — the subagent's first turn is real
work, and the content never passes through the coordinator's context.

```
@inject-file:docs/ui-scripting.md
@inject-cmd:`git show --stat abc123..def456`
@inject-cmd:`jq '.tasks[] | select(.id == "3")' docs/plans/plan.md.tasks.json`
```

Expanded content is appended to the prompt in delimited blocks:

```
<injected-context source="file:docs/ui-scripting.md">
…file content…
</injected-context>
```

Failures never block the dispatch: the directive gets a self-closing
`<injected-context source="…" error="…"/>` marker so the subagent knows to
fetch it manually, and you see a one-line `systemMessage` summary.

## Install

```
/plugin marketplace add Ismael/ccInject
/plugin install ccinject@ccinject
/ccinject:setup
```

`setup` downloads the checksum-verified release binary for your platform
(linux/darwin, amd64/arm64), or builds from source if you have Go. Hooks never
touch the network — only this explicit command does.

A SessionStart hook teaches the coordinator the directive convention
automatically; you don't need to change your CLAUDE.md.

## Directives

| Directive | Meaning |
|---|---|
| `@inject-file:<path>` | Inline the file (relative paths resolve against the session cwd). Binary files are rejected. |
| `` @inject-cmd:`<command>` `` | Inline the command's stdout. No shell: no pipes, redirects, or substitution — but quoted metacharacters are fine (see the jq example above). |

Directives must start a line. Idempotent: a directive whose block (or whose
identical content, ≥64 bytes) is already in the prompt is skipped.

## Allowlist

Built-in: `git show|diff|log` (write/exec flags blocked), `cat`, `jq`,
print-only `sed -n`, `wc`, `head`, `tail`. Extend with:

```json
{ "env": { "CCINJECT_ALLOW": "rg,ls,column" } }
```

Extras are matched by command name only — keeping them read-only is on you.

## Tuning

| Env var | Default | Meaning |
|---|---|---|
| `CCINJECT_DISABLE` | — | `1` disables all rewriting |
| `CCINJECT_NO_SESSION_CONTEXT` | — | `1` suppresses the SessionStart instruction block |
| `CCINJECT_ALLOW` | — | comma-separated extra allowed commands |
| `CCINJECT_CMD_TIMEOUT_MS` | 2000 | per-command timeout (then SIGKILL, whole process group) |
| `CCINJECT_BUDGET_MS` | 5000 | total wall budget per dispatch |
| `CCINJECT_MAX_INJECT_BYTES` | 32768 | per-injection cap (truncates) |
| `CCINJECT_MAX_TOTAL_BYTES` | 131072 | total cap per dispatch |
| `CCINJECT_MAX_DIRECTIVES` | 16 | directives per prompt |
| `CCINJECT_REPO` | git remote | `owner/repo` override for setup downloads |

## Caveats

- ccinject must be the **only** installed hook returning `updatedInput` for
  the Agent tool — Claude Code resolves multiple rewriters last-writer-wins,
  non-deterministically. Re-check when installing new plugins.
- Injected content is not escaped: a file containing a literal
  `</injected-context>` closes its block early.
- The rewritten prompt is visible in the subagent's transcript, not in the
  dispatch UI — the `systemMessage` summary is your confirmation.

## Development

```
make test   # go vet + go test
make build  # bin/ccinject for the host platform
```

Live smoke test: in a Claude Code session with the plugin installed, dispatch
any subagent with ``@inject-cmd:`wc -l go.mod` `` in the prompt and confirm the
systemMessage plus the block in the subagent's transcript.

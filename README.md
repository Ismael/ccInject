# ccinject

Lets your main agent inject stdout or files into subagent prompts directly.
Totally insecure, no guardrails or anything.

Why? I noticed many times the main agent dispatching agents with prompts like "read X file first" or for reviews "do a git diff", then the agent wastes their first turns doing read/running commands. With this plugin, the main agent can just @inject-file:path/something.md or @inject-cmd:`git diff`, so the subagent can get to work immediately.

Does it save tokens? I didn't measure. Hopefully yes, as it should save at least one tool calling turn.

Notes:
- if the subagent needs to edit a file, you can't inject it (because it needs to read the original file first)
- if the file is too big, it will give a warning (which is helpful to the subagent as well, it already knows it's dealing with a big file)

Use at your own risk. If it breaks something, you can keep all the pieces :)

Below is Claude's explanation.

## Introduction

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
| `@inject-file:<path>` | Inline the file (relative paths resolve against the session cwd). Binary files are rejected. Only for files the subagent *reads* — injected content is prompt text, not a file on disk, so a subagent that must edit the file should open it itself. |
| `` @inject-cmd:`<command>` `` | Inline the command's stdout. Runs through `sh -c`, so pipes, redirects, and substitutions all work. |

Directives must start a line. Idempotent: a directive whose block (or whose
identical content, ≥64 bytes) is already in the prompt is skipped.

There is no command allowlist: `@inject-cmd` runs whatever the coordinator
writes, with no permission prompt, at dispatch time. Vetting the command is the
coordinator's job — see Caveats.

An injection larger than `CCINJECT_MAX_INJECT_BYTES` (~0.4 MB, roughly 100k
tokens) is not truncated; it is rejected whole with an
`error="… is X MB, can't add fully"` marker so the subagent fetches it itself.

## Tuning

| Env var | Default | Meaning |
|---|---|---|
| `CCINJECT_DISABLE` | — | `1` disables all rewriting |
| `CCINJECT_DISABLE_CMD` | — | `1` rejects `@inject-cmd` directives (error marker); `@inject-file` still works |
| `CCINJECT_NO_SESSION_CONTEXT` | — | `1` suppresses the SessionStart instruction block |
| `CCINJECT_CMD_TIMEOUT_MS` | 2000 | per-command timeout (then SIGKILL, whole process group) |
| `CCINJECT_BUDGET_MS` | 5000 | total wall budget per dispatch |
| `CCINJECT_MAX_INJECT_BYTES` | 409600 | per-injection cap; larger content is rejected whole (not truncated) |
| `CCINJECT_MAX_DIRECTIVES` | 16 | directives per prompt |
| `CCINJECT_REPO` | git remote | `owner/repo` override for setup downloads |

## Caveats

- `@inject-cmd` executes arbitrary shell at dispatch time with **no permission
  prompt** and **no allowlist** — by design, the coordinator is trusted to only
  write safe commands. The directive is as powerful as whatever wrote the
  dispatch prompt; treat an untrusted prompt source accordingly.
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

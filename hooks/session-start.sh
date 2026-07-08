#!/usr/bin/env bash
# Teaches the coordinator the directive convention so users need no CLAUDE.md
# change. Kept short: this is paid on every session start.
cat > /dev/null
if [[ "${CCINJECT_NO_SESSION_CONTEXT:-0}" == "1" || "${CCINJECT_DISABLE:-0}" == "1" ]]; then
  exit 0
fi
cat <<'EOF'
{"hookSpecificOutput":{"hookEventName":"SessionStart","additionalContext":"ccinject is active. When dispatching subagents with the Agent tool, don't paste file contents or command output into the prompt. Instead write @inject-file:<path> or @inject-cmd:`<command>` on its own line; it is expanded into the prompt at spawn time and the subagent sees the result in an <injected-context> block. Only inject a file the subagent will read for context: if the subagent needs to edit the file, don't inject it — injected content is prompt text, not a file on disk, so let the subagent open the file itself."}}
EOF
exit 0

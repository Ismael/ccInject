#!/usr/bin/env bash
# Fail-open shim: the Go binary does the work; if it isn't built/installed we
# must never block the dispatch — drain stdin, hint once, exit 0.
BIN="${CLAUDE_PLUGIN_ROOT}/bin/ccinject"
if [[ -f "$BIN" && -x "$BIN" ]]; then
  exec "$BIN"
fi
cat > /dev/null
HINT="${TMPDIR:-/tmp}/ccinject-setup-hint-$(id -u)"
if [[ ! -e "$HINT" ]]; then
  : > "$HINT" 2>/dev/null || true
  echo '{"systemMessage":"ccinject: binary not installed — run /ccinject:setup"}'
fi
exit 0

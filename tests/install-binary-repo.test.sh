#!/usr/bin/env bash
# Fast, network-free regression test for repo-slug resolution in
# scripts/install-binary.sh.
#
# Guards the plugin-install defect: when ccinject runs from the extracted
# plugin cache (~/.claude/plugins/cache/…) there is no git remote and no
# CCINJECT_REPO, so the slug must come from the plugin manifest — otherwise the
# release-download path is dead and setup wrongly falls back to a source build.
#
# Uses CCINJECT_PRINT_REPO=1, a short-circuit that prints the resolved slug and
# exits 0 before any network/build, so this test never touches the network.
set -euo pipefail

SCRIPT_DIR=$(cd "$(dirname "$0")/.." && pwd)
INSTALL_SH="$SCRIPT_DIR/scripts/install-binary.sh"

fail=0
check() { # check <label> <expected> <actual>
  if [[ "$2" == "$3" ]]; then
    echo "ok: $1"
  else
    echo "FAIL: $1 — expected [$2], got [$3]" >&2
    fail=1
  fi
}

# --- Case 1: slug resolves from plugin.json when no git remote, no override ---
t1=$(mktemp -d); trap 'rm -rf "$t1" "${t2:-}" "${t3:-}"' EXIT
mkdir -p "$t1/.claude-plugin"
printf '%s\n' '{ "name":"ccinject", "repository":"Ismael/ccInject", "version":"0.1.1", "author":{"name":"Ismael"} }' \
  > "$t1/.claude-plugin/plugin.json"
got=$(env -u CCINJECT_REPO CLAUDE_PLUGIN_ROOT="$t1" CCINJECT_PRINT_REPO=1 bash "$INSTALL_SH")
check "plugin.json slug used when no git remote" "Ismael/ccInject" "$got"

# --- Case 2: explicit CCINJECT_REPO override wins over plugin.json ---
got=$(CCINJECT_REPO=foo/bar CLAUDE_PLUGIN_ROOT="$t1" CCINJECT_PRINT_REPO=1 bash "$INSTALL_SH")
check "CCINJECT_REPO override wins" "foo/bar" "$got"

# --- Case 3: git remote beats plugin.json ---
t2=$(mktemp -d)
mkdir -p "$t2/.claude-plugin"
printf '%s\n' '{ "name":"ccinject", "repository":"Ismael/ccInject", "version":"0.1.1" }' \
  > "$t2/.claude-plugin/plugin.json"
git -C "$t2" init -q
git -C "$t2" remote add origin git@github.com:someone/other.git
got=$(env -u CCINJECT_REPO CLAUDE_PLUGIN_ROOT="$t2" CCINJECT_PRINT_REPO=1 bash "$INSTALL_SH")
check "git remote beats plugin.json" "someone/other" "$got"

# --- Case 4: no source at all → empty slug, still exits 0 (fail-open) ---
t3=$(mktemp -d)
mkdir -p "$t3/.claude-plugin"
printf '%s\n' '{ "name":"ccinject", "version":"0.1.1" }' > "$t3/.claude-plugin/plugin.json"
got=$(env -u CCINJECT_REPO CLAUDE_PLUGIN_ROOT="$t3" CCINJECT_PRINT_REPO=1 bash "$INSTALL_SH")
check "no slug source yields empty string" "" "$got"

exit $fail

#!/usr/bin/env bash
# The ONLY component that touches the network, and only when the user
# explicitly runs /ccinject:setup.
set -euo pipefail

ROOT="${CLAUDE_PLUGIN_ROOT:-$(cd "$(dirname "$0")/.." && pwd)}"
BIN="$ROOT/bin/ccinject"

resolve_repo() { # prints the "owner/repo" slug, or empty if none can be found
  if [[ -n "${CCINJECT_REPO:-}" ]]; then
    printf '%s' "$CCINJECT_REPO"; return
  fi
  local slug
  slug=$(git -C "$ROOT" remote get-url origin 2>/dev/null \
    | sed -E 's#^(git@|https://)github\.com[:/]##; s#\.git$##' || true)
  if [[ -n "$slug" ]]; then
    printf '%s' "$slug"; return
  fi
  # Plugin installs run this script from ~/.claude/plugins/cache/… which is an
  # extracted plugin, not a git clone: no remote to read. So the manifest is the
  # slug's source of truth here. Fail open — a parse error must still yield "".
  sed -n 's/.*"repository"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' \
    "$ROOT/.claude-plugin/plugin.json" 2>/dev/null | head -1 || true
}

# Test/debug short-circuit: print the resolved slug and exit before any network
# or build. Keeps the resolution logic verifiable without touching the network.
if [[ -n "${CCINJECT_PRINT_REPO:-}" ]]; then
  resolve_repo
  echo
  exit 0
fi

mkdir -p "$ROOT/bin"

os=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$os" in linux|darwin) ;; *) echo "unsupported OS: $os" >&2; exit 1;; esac
arch=$(uname -m)
case "$arch" in
  x86_64) arch=amd64 ;;
  aarch64|arm64) arch=arm64 ;;
  *) echo "unsupported arch: $arch" >&2; exit 1 ;;
esac

version=$(sed -n 's/.*"version"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$ROOT/.claude-plugin/plugin.json" | head -1)
repo=$(resolve_repo)

sha_check() { # usage: sha_check <checksums-file> (in cwd, checks ./ccinject)
  if command -v sha256sum >/dev/null; then
    sha256sum -c "$1"
  else
    shasum -a 256 -c "$1"
  fi
}

asset="ccinject-$os-$arch"
if [[ -n "$repo" && -n "$version" ]]; then
  url="https://github.com/$repo/releases/download/v$version"
  tmp=$(mktemp -d); trap 'rm -rf "$tmp"' EXIT
  # Fold verification into the `if` condition: conditions are exempt from
  # `set -e`, so a missing checksum line (empty sum.txt) or a genuine mismatch
  # falls through to the source-build fallback with a message instead of
  # aborting the script silently. Still fail-closed — an unverified binary is
  # never installed, since `install` only runs when sha_check succeeds.
  if curl -fsSL --connect-timeout 10 --max-time 120 "$url/$asset" -o "$tmp/ccinject" &&
     curl -fsSL --connect-timeout 10 --max-time 120 "$url/checksums.txt" -o "$tmp/all.txt" &&
     grep " $asset\$" "$tmp/all.txt" | sed "s|$asset|ccinject|" > "$tmp/sum.txt" &&
     [[ -s "$tmp/sum.txt" ]] &&
     (cd "$tmp" && sha_check sum.txt); then
    install -m 0755 "$tmp/ccinject" "$BIN"
    rm -f "${TMPDIR:-/tmp}/ccinject-setup-hint-$(id -u)"
    echo "installed $BIN (v$version, $os/$arch, checksum verified)"
    exit 0
  fi
  echo "release download or checksum verification failed ($url/$asset) — trying local go build" >&2
fi

if command -v go >/dev/null; then
  (cd "$ROOT" && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o "$BIN" ./cmd/ccinject)
  rm -f "${TMPDIR:-/tmp}/ccinject-setup-hint-$(id -u)"
  echo "built $BIN from source"
else
  echo "no release binary available and no Go toolchain found." >&2
  echo "install Go (https://go.dev/dl/) or set CCINJECT_REPO=<owner>/<repo> and retry." >&2
  exit 1
fi

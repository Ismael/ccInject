---
description: Download (or build) the ccinject binary for this platform
allowed-tools: Bash
---

Run this exact command with the Bash tool and report its output to the user:

    bash "${CLAUDE_PLUGIN_ROOT}/scripts/install-binary.sh"

If it succeeds, tell the user ccinject is ready — subagent dispatch prompts
containing `@inject-file:` / `@inject-cmd:` directives will now be expanded
automatically. If it fails, relay the script's error message verbatim; do not
improvise alternative install methods.

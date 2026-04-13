#!/usr/bin/env bash
# Ensure canonical BMAD `_bmad/` tree exists (per BMAD-METHOD getting-started tutorial).
# Safe to run multiple times; uses non-interactive npx when possible.
#
# Usage: ./scripts/bmad-bootstrap-method.sh
# Requires: Node.js 20+, network for first install.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO="$(cd "${SCRIPT_DIR}/.." && pwd)"
MARKER="${REPO}/_bmad/_config/agent-manifest.csv"

if [[ -f "${MARKER}" ]]; then
	echo "BMAD _bmad/ tree already present (${MARKER})." >&2
	exit 0
fi

if ! command -v npx >/dev/null 2>&1; then
	echo "error: npx not found. Install Node.js 20+ (https://nodejs.org/)." >&2
	exit 1
fi

echo "Installing BMAD Method into ${REPO} (npx bmad-method install)…" >&2
cd "${REPO}"
# --yes accepts npx package install prompts; installer may still prompt for module selection in some versions.
export CI="${CI:-}" 
npx --yes bmad-method@latest install || {
	echo "error: bmad-method install failed. Run manually: cd \"${REPO}\" && npx bmad-method install" >&2
	exit 1
}

if [[ ! -f "${MARKER}" ]]; then
	echo "warning: install finished but ${MARKER} still missing; merge installer output or select BMad Method module." >&2
	exit 0
fi

echo "BMAD bootstrap complete." >&2

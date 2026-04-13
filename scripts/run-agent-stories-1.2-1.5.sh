#!/usr/bin/env bash
# Deprecated single-prompt runner — use the BMAD multi-phase workflow instead.
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
exec "${SCRIPT_DIR}/bmad-story-workflow.sh" --all "$@"

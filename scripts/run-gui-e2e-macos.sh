#!/usr/bin/env bash
# Optional macOS-only black-box GUI E2E (Robotgo + CGO). Safe to call from Linux: exits 0 immediately.
#
# Requires: macOS with a logged-in graphical session, Accessibility permission for the test runner
#           (Terminal, Cursor, or `go test`), Xcode CLT, CGO_ENABLED=1.
#
# Env:
#   PHOTO_TOOL_GUI_E2E_MACOS=1  — set by this script before go test (do not unset unless debugging).
#   BUNDLE                      — when set, tee test output to $BUNDLE/logs/gui-e2e-macos.txt and pass
#                                 PHOTO_TOOL_GUI_E2E_BUNDLE for optional failure PNGs under logs/.
set -euo pipefail

if [[ "$(uname -s)" != "Darwin" ]]; then
	printf '%s\n' "run-gui-e2e-macos: skipping (not macOS)" >&2
	exit 0
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO="$(cd "${SCRIPT_DIR}/.." && pwd)"
cd "${REPO}"

export CGO_ENABLED=1
export PHOTO_TOOL_GUI_E2E_MACOS=1

if [[ -n "${BUNDLE:-}" ]]; then
	mkdir -p "${BUNDLE}/logs"
	export PHOTO_TOOL_GUI_E2E_BUNDLE="${BUNDLE}"
	set -o pipefail
	rc=0
	go test ./tests/gui_macos/... -count=1 -timeout=15m 2>&1 | tee "${BUNDLE}/logs/gui-e2e-macos.txt" || rc=$?
	exit "${rc}"
fi

exec go test ./tests/gui_macos/... -count=1 -timeout=15m

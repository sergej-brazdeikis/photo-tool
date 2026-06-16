#!/usr/bin/env bash
# Validate toolchain for extended testing matrix (Go, CGO, DISPLAY/Xvfb, Cursor agent).
set -euo pipefail

log() { printf '%s\n' "$*" >&2; }

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
export REPO
# shellcheck source=extended-test-env.sh
source "${REPO}/scripts/extended-test-env.sh"
cd "$REPO"

need_go_version() {
	local want="${1:-1.26.2}"
	if [[ -z "${GO_BIN}" ]] || ! command -v go >/dev/null 2>&1; then
		log "error: go not on PATH"
		log "Tried: ~/.local/go/bin, /usr/local/go/bin"
		log "Install Go ${want}+ from https://go.dev/dl/ and re-run"
		exit 1
	fi
	local ver
	ver="$(go env GOVERSION | sed 's/^go//')"
	log "Go: ${ver} (${GO_BIN})"
}

check_cgo_deps() {
	if ! command -v gcc >/dev/null 2>&1; then
		log "error: gcc not found (required for Fyne CGO)"
		exit 1
	fi
	if ! command -v pkg-config >/dev/null 2>&1; then
		log "warning: pkg-config not found — install libgl1-mesa-dev xorg-dev (see docs/env.md)"
	fi
}

check_display() {
	if [[ -n "${DISPLAY:-}" || -n "${WAYLAND_DISPLAY:-}" ]]; then
		log "Display: ok (${DISPLAY:-}${WAYLAND_DISPLAY:+wayland})"
		return 0
	fi
	if command -v xvfb-run >/dev/null 2>&1; then
		log "Display: unset — will use xvfb-run for real GUI capture"
		export EXTENDED_USE_XVFB=1
		return 0
	fi
	log "error: no DISPLAY and xvfb-run not installed"
	log "Install: sudo apt-get install -y xvfb"
	exit 1
}

check_agent() {
	if command -v "${EXTENDED_AGENT_BIN}" >/dev/null 2>&1; then
		log "Agent CLI: ${EXTENDED_AGENT_BIN} ($(command -v "${EXTENDED_AGENT_BIN}"))"
		return 0
	fi
	log "error: Cursor agent CLI '${EXTENDED_AGENT_BIN}' not on PATH"
	log ""
	log "Install and authenticate:"
	log "  curl https://cursor.com/install -fsS | bash"
	log "  agent login   OR   export CURSOR_API_KEY=..."
	exit 1
}

check_build_smoke() {
	local tmp
	tmp="$(mktemp -d)"
	log "Build smoke: go build -o ${tmp}/photo-tool ."
	if ! go build -o "${tmp}/photo-tool" .; then
		rm -rf "${tmp}"
		log "error: go build failed — fix compile/CGO before extended run"
		exit 1
	fi
	if [[ ! -x "${tmp}/photo-tool" ]]; then
		rm -rf "${tmp}"
		log "error: build produced no executable"
		exit 1
	fi
	rm -rf "${tmp}"
	log "Build smoke: ok"
}

need_go_version "1.26.2"
check_cgo_deps
check_display
check_agent
check_build_smoke

ENV_EXAMPLE="${REPO}/scripts/extended-test.env.example"
cat >"${ENV_EXAMPLE}" <<'EOF'
# Source before extended-test runs (optional — scripts bootstrap PATH automatically)
export PATH="$HOME/.local/go/bin:/usr/local/go/bin:$HOME/.local/bin:$PATH"
export EXTENDED_LOOP_MAX=20
export EXTENDED_PARALLEL_FIX=4
export EXTENDED_STOP_ON_FAIL=1
export PHOTO_TOOL_GUI_E2E_LINUX=1
export PHOTO_TOOL_UX_CAPTURE_APP_MODE=real_binary
EOF

log "extended-test-setup: ok"
log "Optional: source ${ENV_EXAMPLE}"

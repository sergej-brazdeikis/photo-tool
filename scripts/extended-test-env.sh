#!/usr/bin/env bash
# Shared env for extended-test-*.sh (source, do not run alone).
# Ensures go/agent/CGO are on PATH even in subshells and fresh Cursor terminals.

if [[ -z "${REPO:-}" ]]; then
	REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
fi

for _go_dir in "${HOME}/.local/go/bin" "/usr/local/go/bin"; do
	if [[ -d "${_go_dir}" ]]; then
		case ":${PATH}:" in
		*":${_go_dir}:"*) ;;
		*) PATH="${_go_dir}:${PATH}" ;;
		esac
	fi
done
case ":${PATH}:" in
*":${HOME}/.local/bin:"*) ;;
*) PATH="${HOME}/.local/bin:${PATH}" ;;
esac
export PATH

if [[ -d "${HOME}/.local/debian-dev" ]]; then
	export PKG_CONFIG_PATH="${HOME}/.local/debian-dev/lib/x86_64-linux-gnu/pkgconfig:${PKG_CONFIG_PATH:-}"
	export CGO_CFLAGS="-I${HOME}/.local/debian-dev/include"
	export CGO_LDFLAGS="-L${HOME}/.local/debian-dev/lib/x86_64-linux-gnu"
fi

GO_BIN="$(command -v go 2>/dev/null || true)"
export GO_BIN

EXTENDED_AGENT_BIN="agent"
if [[ -n "${UX_AGENT:-}" ]]; then
	EXTENDED_AGENT_BIN="${UX_AGENT}"
elif [[ -n "${CURSOR_AGENT:-}" ]] && command -v "${CURSOR_AGENT}" >/dev/null 2>&1; then
	EXTENDED_AGENT_BIN="${CURSOR_AGENT}"
fi
export EXTENDED_AGENT_BIN

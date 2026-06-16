#!/usr/bin/env bash
# Generate extended testing matrix artifacts under RUN_DIR.
set -euo pipefail

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
export REPO
# shellcheck source=extended-test-env.sh
source "${REPO}/scripts/extended-test-env.sh"
OUT=""
while [[ $# -gt 0 ]]; do
	case "$1" in
	-out)
		OUT="$2"
		shift 2
		;;
	*)
		echo "usage: $0 -out RUN_DIR" >&2
		exit 2
		;;
	esac
done
if [[ -z "${OUT}" ]]; then
	echo "error: -out required" >&2
	exit 2
fi
mkdir -p "${OUT}"
cd "${REPO}"
go run ./tests/extended/cmd/matrix-gen -out "${OUT}"

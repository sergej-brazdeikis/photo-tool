#!/usr/bin/env bash
# Local-only: assemble judge bundle → vision judge (Cursor `agent`) → fix agent → post-fix BMAD-style code review → repeat.
# CI must never run this script (no LLM / no agent in GitHub Actions for this repo).
#
# Prerequisites: `agent` on PATH (`agent login` or CURSOR_API_KEY), python3 (for manifest in assemble script).
#
# Env:
#   UX_LOOP_MAX          — max rounds when UX_LOOP_NO_MAX is unset (default 5)
#   UX_LOOP_NO_MAX       — if 1, ignore UX_LOOP_MAX (run until pass, deadline, or Ctrl+C)
#   UX_LOOP_UNTIL_LOCAL  — stop before the next iteration once local wall clock is at or past HH:MM (24h, today).
#                          Example: UX_LOOP_UNTIL_LOCAL=18:00 runs until 6pm local time.
#   UX_AGENT             — agent binary name if not `agent` (default: agent)
#   CURSOR_AGENT         — alias for UX_AGENT; UX_AGENT wins if both set
#   UX_LOOP_POST_FIX_REVIEW — if 0, skip the post-fix code review agent (default: run review)
#
# Human kill switch: Ctrl+C between iterations.
#
# All-day example (until 6pm local, no iteration cap):
#   UX_LOOP_NO_MAX=1 UX_LOOP_UNTIL_LOCAL=18:00 ./scripts/ux-judge-loop.sh
#
# Same with a safety cap (e.g. at most 200 rounds):
#   UX_LOOP_MAX=200 UX_LOOP_UNTIL_LOCAL=18:00 ./scripts/ux-judge-loop.sh
#
# Usage (from repo root):
#   ./scripts/ux-judge-loop.sh
set -euo pipefail

log() { printf '%s\n' "$*" >&2; }

# Seconds since epoch for today's date + HH:MM in local timezone (macOS BSD date vs GNU date).
ux_deadline_epoch_today() {
	local hm="$1"
	local d
	d="$(date +%Y-%m-%d)"
	case "$(uname -s)" in
	Darwin)
		date -j -f "%Y-%m-%d %H:%M" "${d} ${hm}" +%s
		;;
	*)
		# GNU date
		date -d "${d} ${hm}:00" +%s 2>/dev/null || date -d "${d} ${hm}" +%s
		;;
	esac
}

ux_past_local_deadline() {
	local hm="${UX_LOOP_UNTIL_LOCAL:-}"
	[[ -z "${hm}" ]] && return 1
	if ! [[ "${hm}" =~ ^[0-9]{1,2}:[0-9]{2}$ ]]; then
		log "error: UX_LOOP_UNTIL_LOCAL must be HH:MM (24h), got: ${hm}"
		exit 2
	fi
	local now_s end_s
	now_s="$(date +%s)"
	end_s="$(ux_deadline_epoch_today "${hm}")"
	if [[ "${now_s}" -ge "${end_s}" ]]; then
		return 0
	fi
	return 1
}

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO="$(cd "${SCRIPT_DIR}/.." && pwd)"
cd "$REPO"

AGENT_BIN="${UX_AGENT:-${CURSOR_AGENT:-agent}}"
if ! command -v "${AGENT_BIN}" >/dev/null 2>&1; then
	log "error: '${AGENT_BIN}' not on PATH (set UX_AGENT if your binary has another name)"
	exit 1
fi

JUDGE_SPEC="${REPO}/_bmad-output/test-artifacts/judge-prompt-v2-screenshots.md"
FIX_SPEC="${REPO}/_bmad-output/test-artifacts/ux-fix-from-judge-prompt-v1.md"
POST_FIX_REVIEW_SPEC="${REPO}/_bmad-output/test-artifacts/ux-post-fix-code-review-loop.md"
for f in "${JUDGE_SPEC}" "${FIX_SPEC}" "${POST_FIX_REVIEW_SPEC}"; do
	if [[ ! -f "${f}" ]]; then
		log "error: missing ${f}"
		exit 1
	fi
done

if [[ "${UX_LOOP_NO_MAX:-}" == "1" ]]; then
	max_eff=2147483647
	max_label="no-max"
else
	max_eff="${UX_LOOP_MAX:-5}"
	max_label="${max_eff}"
fi

run_agent() {
	local prompt="$1"
	"${AGENT_BIN}" --print --trust --workspace "${REPO}" --yolo "${prompt}"
}

i=0
while true; do
	if ux_past_local_deadline; then
		log "ux-judge-loop: local time is past UX_LOOP_UNTIL_LOCAL=${UX_LOOP_UNTIL_LOCAL} — stopping (no pass required)"
		exit 0
	fi

	i=$((i + 1))
	if [[ "${i}" -gt "${max_eff}" ]]; then
		log "ux-judge-loop: exceeded UX_LOOP_MAX=${max_eff} without pass"
		exit 1
	fi
	log "ux-judge-loop: iteration ${i}/${max_label}"

	BUNDLE="$(UX_BUNDLE_EMIT_PATH_ONLY=1 "${SCRIPT_DIR}/assemble-judge-bundle.sh")"
	log "ux-judge-loop: bundle ${BUNDLE}"

	run_agent "photo-tool — **UX judge (v2, screenshots)**. Run locally only.

Read and follow the instructions in this file (open it and use as the contract):
${JUDGE_SPEC}

**Bundle directory (absolute):** ${BUNDLE}

- Use \`context/rubric.md\`, \`ui/steps.json\`, ordered \`ui/*.png\`, and \`logs/go-test.txt\` from that bundle.
- Write **only** under that bundle: \`verdict/judge-output.md\` with structure required by the spec.
- The **last line** of \`verdict/judge-output.md\` must be exactly \`UX_JUDGE_RESULT=pass\` or \`UX_JUDGE_RESULT=fail\` (no other text on that line)."

	verdict="${BUNDLE}/verdict/judge-output.md"
	if [[ ! -f "${verdict}" ]]; then
		log "ux-judge-loop: missing ${verdict}; treating as fail"
	elif [[ "$(tail -n 1 "${verdict}" | tr -d '\r')" == "UX_JUDGE_RESULT=pass" ]]; then
		log "ux-judge-loop: UX_JUDGE_RESULT=pass — done"
		exit 0
	else
		log "ux-judge-loop: judge did not end with UX_JUDGE_RESULT=pass (see ${verdict})"
	fi

	run_agent "photo-tool — **UX implementer (fix from judge)**.

Read and follow:
${FIX_SPEC}

**Repository root:** ${REPO}
**Bundle (read verdict here):** ${BUNDLE}

Implement minimal Fyne/layout changes to address gaps in \`verdict/judge-output.md\`. Run \`go test ./...\` and \`go build .\` from the repo root. Do **not** re-run the judge in this invocation; the outer loop will re-bundle and re-judge."

	if [[ "${UX_LOOP_POST_FIX_REVIEW:-1}" != "0" ]]; then
		log "ux-judge-loop: post-fix code review (BMAD-style, non-interactive)"
		post_fix_prompt="$(printf '%s\n\n%s\n\n%s\n%s\n%s\n%s\n' \
			"photo-tool — **Post-fix code review** (UX judge loop)." \
			"Read and follow this contract (single pass, no human checkpoints, no source edits):" \
			"${POST_FIX_REVIEW_SPEC}" \
			"**Repository root:** ${REPO}" \
			"**Bundle directory (absolute):** ${BUNDLE}" \
			"Under that bundle, read \`verdict/judge-output.md\` for context and write \`verdict/code-review-post-fix.md\` as specified in the contract.")"
		run_agent "${post_fix_prompt}"
		review_rc=$?
		if [[ "${review_rc}" -ne 0 ]]; then
			log "ux-judge-loop: warning: post-fix code review agent exited ${review_rc}; continuing loop"
		fi
	fi

	log "ux-judge-loop: iteration complete; next loop will re-capture"
done

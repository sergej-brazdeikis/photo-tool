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
#   CURSOR_AGENT         — optional CLI binary name if not `agent`; UX_AGENT wins if both set.
#                          Values 0 or 1 are ignored (Cursor IDE uses these as flags, not paths).
#   UX_LOOP_POST_FIX_REVIEW — if 0, skip the post-fix code review agent (default: run review)
#   UX_LOOP_SELF_IMPROVE   — if 0, skip the self-improve agent (default: run it). Writes bundle
#                            verdict/loop-self-improve.md; may patch judge prompt / rubric / capture test.
#   UX_LOOP_POST_IMPL_TEST — if 1, after the implementer step run `go test ./... ./tests/e2e/... -count=1`;
#                            on failure skip post-fix / QA / self-improve for this iteration (continue loop).
#   UX_LOOP_QA_AGENT       — if 1, run the QA agent after post-fix review (default: 0). Writes verdict/qa-loop-output.md.
#   UX_LOOP_QA_FIX         — if 0, skip the QA implementer after a failing QA pass (default: 1). Runs only when
#                            UX_LOOP_QA_AGENT=1 and verdict/qa-loop-output.md ends with QA_LOOP_RESULT=fail.
#   UX_LOOP_GUI_E2E_MACOS  — if 1 and host is Darwin, after the QA block run scripts/run-gui-e2e-macos.sh with
#                            BUNDLE set (tee logs/gui-e2e-macos.txt). Default 0. Requires local display + permissions.
#   UX_LOOP_GUI_E2E_FIX    — if 0, skip the GUI E2E implementer after a failing GUI step (default: 1). Only when
#                            UX_LOOP_GUI_E2E_MACOS=1 and the GUI script exits non-zero.
#   UX_BUNDLE_ALLOW_E2E_FAIL — passed through to assemble-judge-bundle.sh (see that script).
#
# Note: `assemble-judge-bundle.sh` exits non-zero if UX capture fails or PNG validation fails, so the loop
# stops before the judge (no wasted agent calls on empty ui/).
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
#
# Everything enabled (defaults — no extra env required):
#   Post-fix review, self-improve, and judge bundle with NFR-min (1024×768) PNGs from
#   TestUXJourneyCapture are all ON unless you set UX_LOOP_POST_FIX_REVIEW=0 or
#   UX_LOOP_SELF_IMPROVE=0. Explicit same-as-default:
#     UX_LOOP_POST_FIX_REVIEW=1 UX_LOOP_SELF_IMPROVE=1 ./scripts/ux-judge-loop.sh
#   Dark-theme PNGs in the bundle are not produced by assemble-judge-bundle.sh by default
#   (capture uses light theme unless you extend the assemble script to set PHOTO_TOOL_TEST_THEME=dark).
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

# Cursor may set CURSOR_AGENT=0|1 as an internal flag; that is not the CLI binary name.
AGENT_BIN="${UX_AGENT:-${CURSOR_AGENT:-agent}}"
if [[ "${AGENT_BIN}" == "0" || "${AGENT_BIN}" == "1" ]]; then
	AGENT_BIN=agent
fi
if ! command -v "${AGENT_BIN}" >/dev/null 2>&1; then
	log "error: '${AGENT_BIN}' not on PATH (set UX_AGENT if your binary has another name)"
	exit 1
fi

JUDGE_SPEC="${REPO}/_bmad-output/test-artifacts/judge-prompt-v2-screenshots.md"
FIX_SPEC="${REPO}/_bmad-output/test-artifacts/ux-fix-from-judge-prompt-v1.md"
POST_FIX_REVIEW_SPEC="${REPO}/_bmad-output/test-artifacts/ux-post-fix-code-review-loop.md"
SELF_IMPROVE_SPEC="${REPO}/_bmad-output/test-artifacts/ux-judge-loop-self-improve.md"
QA_LOOP_SPEC="${REPO}/_bmad-output/test-artifacts/ux-qa-post-fix-loop.md"
QA_FIX_SPEC="${REPO}/_bmad-output/test-artifacts/ux-fix-from-qa-prompt-v1.md"
GUI_E2E_FIX_SPEC="${REPO}/_bmad-output/test-artifacts/ux-fix-from-gui-e2e-macos-prompt-v1.md"
for f in "${JUDGE_SPEC}" "${FIX_SPEC}" "${POST_FIX_REVIEW_SPEC}" "${SELF_IMPROVE_SPEC}" "${QA_LOOP_SPEC}" "${QA_FIX_SPEC}" "${GUI_E2E_FIX_SPEC}"; do
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
	if [[ -z "${BUNDLE}" || ! -d "${BUNDLE}" ]]; then
		log "error: ux-judge-loop: assemble-judge-bundle did not produce a bundle path"
		exit 1
	fi
	log "ux-judge-loop: bundle ${BUNDLE}"
	if [[ -f "${BUNDLE}/manifest.json" ]] && command -v python3 >/dev/null 2>&1; then
		gtc="$(python3 -c "import json,sys;print(json.load(open(sys.argv[1]))['exit_codes'].get('go_test_ci',0))" "${BUNDLE}/manifest.json" 2>/dev/null || echo "")"
		if [[ -n "${gtc}" && "${gtc}" != "0" ]]; then
			log "ux-judge-loop: warning: manifest go_test_ci exit code is ${gtc} (see ${BUNDLE}/logs/go-test-ci.txt) — judge still runs; fix CI-tagged tests if you rely on them"
		fi
		gte="$(python3 -c "import json,sys;print(json.load(open(sys.argv[1]))['exit_codes'].get('go_test_e2e',0))" "${BUNDLE}/manifest.json" 2>/dev/null || echo "")"
		if [[ -n "${gte}" && "${gte}" != "0" ]]; then
			log "ux-judge-loop: warning: manifest go_test_e2e exit code is ${gte} (see ${BUNDLE}/logs/go-test-e2e.txt) — bundle used UX_BUNDLE_ALLOW_E2E_FAIL=1 or stale; fix E2E if you rely on strict gates"
		fi
	fi

	run_agent "photo-tool — **UX judge (v2, screenshots)**. Run locally only.

Read and follow the instructions in this file (open it and use as the contract):
${JUDGE_SPEC}

**Bundle directory (absolute):** ${BUNDLE}

- Use \`context/rubric.md\`, \`ui/capture-files.txt\`, \`ui/steps.json\`, each \`ui/<steps[].file>\` image (do not rely on directory listing alone), \`logs/go-test.txt\`, and \`logs/go-test-e2e.txt\` from that bundle.
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

	if [[ "${UX_LOOP_POST_IMPL_TEST:-0}" == "1" ]]; then
		log "ux-judge-loop: post-implementer regression tests (go test ./... ./tests/e2e/...)"
		set +e
		go test ./... ./tests/e2e/... -count=1 >"${BUNDLE}/logs/post-implementer-go-test.txt" 2>&1
		pit_rc=$?
		set -e
		if [[ "${pit_rc}" -ne 0 ]]; then
			log "ux-judge-loop: post-implementer tests failed (exit ${pit_rc}); see ${BUNDLE}/logs/post-implementer-go-test.txt — skipping post-fix review, QA agent, QA fix, and self-improve this iteration"
			log "ux-judge-loop: iteration complete; next loop will re-capture"
			continue
		fi
	fi

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

	if [[ "${UX_LOOP_QA_AGENT:-0}" == "1" ]]; then
		log "ux-judge-loop: QA loop agent (test traceability + log evidence)"
		qa_prompt="$(printf '%s\n\n%s\n\n%s\n%s\n%s\n%s\n' \
			"photo-tool — **UX judge loop — QA pass** (one pass)." \
			"Read and follow this contract (no application source edits, no go test from this role):" \
			"${QA_LOOP_SPEC}" \
			"**Repository root:** ${REPO}" \
			"**Bundle directory (absolute):** ${BUNDLE}" \
			"Write verdict/qa-loop-output.md as specified; last line QA_LOOP_RESULT=pass or QA_LOOP_RESULT=fail.")"
		run_agent "${qa_prompt}"
		qa_rc=$?
		if [[ "${qa_rc}" -ne 0 ]]; then
			log "ux-judge-loop: warning: QA loop agent exited ${qa_rc}; continuing loop"
		fi
		qa_out="${BUNDLE}/verdict/qa-loop-output.md"
		qa_fail=false
		if [[ -f "${qa_out}" ]] && [[ "$(tail -n 1 "${qa_out}" | tr -d '\r')" == "QA_LOOP_RESULT=fail" ]]; then
			qa_fail=true
			log "ux-judge-loop: ${qa_out} ends with QA_LOOP_RESULT=fail"
		fi
		if [[ "${qa_fail}" == true ]] && [[ "${UX_LOOP_QA_FIX:-1}" != "0" ]]; then
			log "ux-judge-loop: QA implementer (fix from QA pass)"
			run_agent "photo-tool — **QA implementer (fix from QA)**.

Read and follow:
${QA_FIX_SPEC}

**Repository root:** ${REPO}
**Bundle (read QA verdict and logs here):** ${BUNDLE}

Implement minimal changes so automated tests and traceability gaps from \`verdict/qa-loop-output.md\` are addressed. Run \`go test ./...\`, \`go test ./tests/e2e/... -count=1\`, and \`go build .\` from the repo root. Do **not** re-run the vision judge or QA agent; the next loop iteration will re-assemble."
			qfix_rc=$?
			if [[ "${qfix_rc}" -ne 0 ]]; then
				log "ux-judge-loop: warning: QA implementer agent exited ${qfix_rc}; continuing loop"
			fi
		elif [[ "${qa_fail}" == true ]]; then
			log "ux-judge-loop: warning: QA_LOOP_RESULT=fail but UX_LOOP_QA_FIX=0 — skipping QA implementer (UX_JUDGE_RESULT still drives loop exit)"
		fi
	fi

	if [[ "$(uname -s)" == "Darwin" && "${UX_LOOP_GUI_E2E_MACOS:-0}" == "1" ]]; then
		log "ux-judge-loop: macOS GUI E2E (optional black-box)"
		set +e
		BUNDLE="${BUNDLE}" "${SCRIPT_DIR}/run-gui-e2e-macos.sh"
		gui_rc=$?
		set -e
		if [[ "${gui_rc}" -ne 0 ]]; then
			log "ux-judge-loop: GUI E2E failed (exit ${gui_rc}); see ${BUNDLE}/logs/gui-e2e-macos.txt"
			if [[ "${UX_LOOP_GUI_E2E_FIX:-1}" != "0" ]]; then
				log "ux-judge-loop: GUI E2E implementer (fix from macOS GUI log)"
				run_agent "$(printf '%s\n\n%s\n\n%s\n%s\n%s\n%s\n' \
					"photo-tool — **GUI E2E implementer (macOS)**." \
					"Read and follow this contract (single pass):" \
					"${GUI_E2E_FIX_SPEC}" \
					"**Repository root:** ${REPO}" \
					"**Bundle directory (absolute):** ${BUNDLE}" \
					"Read logs/gui-e2e-macos.txt and optional logs/gui-e2e-failure-*.png; implement minimal fixes. On macOS run CGO_ENABLED=1 PHOTO_TOOL_GUI_E2E_MACOS=1 go test ./tests/gui_macos/... -count=1; from repo root also run go test ./... and go test ./tests/e2e/... -count=1 and go build . Do not re-run the vision judge, QA agent, or GUI harness in this invocation.")"
				gfix_rc=$?
				if [[ "${gfix_rc}" -ne 0 ]]; then
					log "ux-judge-loop: warning: GUI E2E implementer agent exited ${gfix_rc}; continuing loop"
				fi
			else
				log "ux-judge-loop: warning: GUI E2E failed but UX_LOOP_GUI_E2E_FIX=0 — skipping GUI implementer"
			fi
		fi
	fi

	if [[ "${UX_LOOP_SELF_IMPROVE:-1}" != "0" ]]; then
		log "ux-judge-loop: self-improve (prompt/rubric/capture contract)"
		self_improve_prompt="$(printf '%s\n\n%s\n\n%s\n%s\n%s\n%s\n' \
			"photo-tool — **UX judge loop — self-improve** (one pass)." \
			"Read and follow this contract (you may patch only the allowed artifact paths listed there):" \
			"${SELF_IMPROVE_SPEC}" \
			"**Repository root:** ${REPO}" \
			"**Bundle directory (absolute):** ${BUNDLE}" \
			"Read verdict + optional code-review under the bundle; write verdict/loop-self-improve.md as specified.")"
		run_agent "${self_improve_prompt}"
		si_rc=$?
		if [[ "${si_rc}" -ne 0 ]]; then
			log "ux-judge-loop: warning: self-improve agent exited ${si_rc}; continuing loop"
		fi
	fi

	log "ux-judge-loop: iteration complete; next loop will re-capture"
done

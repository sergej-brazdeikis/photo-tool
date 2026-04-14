#!/usr/bin/env bash
# BMAD story automation aligned with BMAD-METHOD getting-started build cycle:
#   [batch] bmad-check-implementation-readiness → per story: create-story → dev-story →
#   bmad-code-review → AC verify → [batch] bmad-retrospective.
#
# Prerequisites: `agent` on PATH, `agent login` or CURSOR_API_KEY.
# Optional: Node 20+ + `npx` for `scripts/bmad-bootstrap-method.sh` (--bootstrap).
#
# Usage:
#   ./scripts/bmad-story-workflow.sh 1.2
#   ./scripts/bmad-story-workflow.sh --all
#   ./scripts/bmad-story-workflow.sh --full-parity 1.2
#   ./scripts/bmad-story-workflow.sh --phase=review 1.2
#   ./scripts/bmad-story-workflow.sh --phase=readiness
#   ./scripts/bmad-story-workflow.sh --phase=retro
#   ./scripts/bmad-story-workflow.sh --bootstrap --all
#   ./scripts/bmad-story-workflow.sh --skip-readiness --skip-retro --all
#   ./scripts/bmad-story-workflow.sh --with-readiness 1.2
#   ./scripts/bmad-story-workflow.sh --with-retro --retro-epic=1
#   ./scripts/bmad-story-workflow.sh --sprint-sync --all   # regenerate sprint-status.yaml first
#   ./scripts/bmad-story-workflow.sh --phase=sprint
#   ./scripts/bmad-story-workflow.sh --skip-go-fix-loop 1.2   # do not run workspace gate / agent repair
#   ./scripts/bmad-story-workflow.sh --phase=gate             # only automated Epic 1 / module gate (no story phases)
#   make gate                                                 # same as --phase=gate (see Makefile)
#   ./scripts/bmad-story-workflow.sh --skip-verify-autofix --phase=verify 1.5   # verify only, no AC-driven repair loop
#   VERIFY_AC_FIX_MAX=8 ./scripts/bmad-story-workflow.sh --phase=verify 1.5
#
# Story ids match _bmad-output/planning-artifacts/epics.md (### Story X.Y: …).
# Shorthands (expand to story lists): 1.2-1.5, 1.1-1.8, epic-1, epic-2, epic-3, epic-4, all-epics
#   ./scripts/bmad-story-workflow.sh epic-1
#   ./scripts/bmad-story-workflow.sh --all-epics --skip-readiness --skip-retro
# `--all` = first delivery slice (1.2–1.5). `--all-epics` = every story 1.1 through 4.1.
# Shorthand `1.6-thru-4.1` = 1.6–1.8, Epic 2 (2.1–2.12), Epic 3 (3.1–3.5), 4.1 (create + implement backlog after 1.5).
#
# Hands-free (Cursor agent CLI): pass `--hands-free` → adds `--yolo` (Run Everything). MCP servers are **not** auto-approved.
# The script always passes `--trust` with `--print`. You can add e.g. `--model sonnet-4` after `--hands-free`.
#
# Party mode in automation: interactive Party Mode uses IDE subagents; in `--print` mode the agent follows
# `.claude/skills/bmad-party-mode/SKILL.md` and applies file edits without waiting for you.
#   --party                  → after each selected hook, run PARTY_ROUNDS headless party sessions (default 2)
#   --party-rounds=N        → override round count (same sessions repeated with "deepen / challenge prior")
#   --party-after=create,dev → default; use e.g. `dev` only
# Env: PARTY_ROUNDS=2 PARTY_AFTER=create,dev (if set before --party, --party still wins unless you only use env)
#
# Example (your backlog, minimal prompts, party after create+dev):
#   ./scripts/bmad-story-workflow.sh --hands-free --party --skip-readiness --skip-retro 1.6-thru-4.1
#
# Workspace gate (after --phase=all|dev|verify|review for each run, once all stories finish):
#   go mod tidy → go fmt ./... → go mod verify → go vet ./... → go test ./... → go build .
#   Optional: staticcheck (if on PATH; set SKIP_STATICCHECK=1 to skip), golangci-lint (GOLANGCI_LINT=1),
#   go test -race (GO_TEST_RACE=1, slow).
# On failure, invokes `agent` with the log (tail), up to GO_FIX_MAX_ITERATIONS (default 5, --go-fix-max=N).
# This does not replace manual Fyne UX QA — it enforces compile/test/static checks.
#
# `--phase=gate --skip-go-fix-loop` runs checks only (no agent); `agent` not required for that combo.
#
# Verify autofix (--phase=verify or all/review paths that run verify):
#   The verify agent must end with a single line: VERIFY_ANY_FAIL=no or VERIFY_ANY_FAIL=yes.
#   If yes (and not --skip-verify-autofix), a repair agent runs (tail of verify log), then verify re-runs,
#   up to VERIFY_AC_FIX_MAX times (default 5, env or --verify-ac-fix-max=N). This closes AC gaps that
#   still pass `go test` (e.g. UX copy / guards) without parsing markdown tables heuristically.
#
# Batch defaults: more than one story (e.g. --all) runs readiness once and retrospective at end.
# Forward agent flags: --yolo, --force, etc. (passed through to `agent`).

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO="$(cd "${SCRIPT_DIR}/.." && pwd)"
# #region agent log
_dbg_log="${REPO}/.cursor/debug-8dfdc8.log"
_dbg_ts="$(($(date +%s) * 1000))"
_dbg_lines="$(wc -l <"${BASH_SOURCE[0]}" | tr -d ' ')"
{
	printf '%s\n' "{\"sessionId\":\"8dfdc8\",\"hypothesisId\":\"H-entry\",\"location\":\"bmad-story-workflow.sh:boot\",\"message\":\"script started after parse\",\"data\":{\"bashVersion\":\"${BASH_VERSION}\",\"scriptLines\":${_dbg_lines},\"pid\":$$},\"timestamp\":${_dbg_ts}}"
} >>"${_dbg_log}" 2>/dev/null || true
# #endregion
CONFIG="${REPO}/_bmad/bmm/config.yaml"
IMPL="${REPO}/_bmad-output/implementation-artifacts"
PLAN="${REPO}/_bmad-output/planning-artifacts"
SKILLS="${REPO}/.claude/skills"

CREATE_STORY_WF="${SKILLS}/bmad-create-story/workflow.md"
DEV_STORY_WF="${SKILLS}/bmad-dev-story/workflow.md"
READINESS_WF="${SKILLS}/bmad-check-implementation-readiness/workflow.md"
CODE_REVIEW_WF="${SKILLS}/bmad-code-review/workflow.md"
RETRO_WF="${SKILLS}/bmad-retrospective/workflow.md"
SPRINT_WF="${SKILLS}/bmad-sprint-planning/workflow.md"

PHASE="all"
EXTRA_AGENT_ARGS=()
STORIES=()
DO_BOOTSTRAP=0
FULL_PARITY=0
SKIP_READINESS=0
SKIP_CODE_REVIEW=0
SKIP_RETRO=0
FORCE_READINESS=0
FORCE_RETRO=0
RETRO_EPIC="${RETRO_EPIC:-1}"
SPRINT_SYNC=0
SKIP_GO_FIX_LOOP=0
GO_FIX_MAX_ITERATIONS="${GO_FIX_MAX_ITERATIONS:-5}"
SKIP_VERIFY_AUTOFIX=0
VERIFY_AC_FIX_MAX="${VERIFY_AC_FIX_MAX:-5}"
HANDS_FREE=0
PARTY_ROUNDS="${PARTY_ROUNDS:-0}"
PARTY_AFTER="${PARTY_AFTER:-create,dev}"
PARTY_SKILL="${SKILLS}/bmad-party-mode/SKILL.md"

usage() {
	sed -n '2,64p' "$0" | sed 's/^# \{0,1\}//'
	exit 0
}

while [[ $# -gt 0 ]]; do
	case "$1" in
	-h | --help) usage ;;
	--phase=*)
		PHASE="${1#--phase=}"
		shift
		;;
	--all)
		STORIES=(1.2 1.3 1.4 1.5)
		shift
		;;
	--all-epics)
		STORIES=(all-epics)
		shift
		;;
	--full-parity)
		FULL_PARITY=1
		shift
		;;
	--bootstrap)
		DO_BOOTSTRAP=1
		shift
		;;
	--skip-readiness)
		SKIP_READINESS=1
		shift
		;;
	--skip-code-review)
		SKIP_CODE_REVIEW=1
		shift
		;;
	--skip-retro)
		SKIP_RETRO=1
		shift
		;;
	--skip-go-fix-loop)
		SKIP_GO_FIX_LOOP=1
		shift
		;;
	--go-fix-max=*)
		GO_FIX_MAX_ITERATIONS="${1#--go-fix-max=}"
		shift
		;;
	--skip-verify-autofix)
		SKIP_VERIFY_AUTOFIX=1
		shift
		;;
	--verify-ac-fix-max=*)
		VERIFY_AC_FIX_MAX="${1#--verify-ac-fix-max=}"
		shift
		;;
	--with-readiness)
		FORCE_READINESS=1
		shift
		;;
	--with-retro)
		FORCE_RETRO=1
		shift
		;;
	--retro-epic=*)
		RETRO_EPIC="${1#--retro-epic=}"
		FORCE_RETRO=1
		shift
		;;
	--sprint-sync)
		SPRINT_SYNC=1
		shift
		;;
	--hands-free)
		HANDS_FREE=1
		shift
		;;
	--party)
		PARTY_ROUNDS=2
		shift
		;;
	--party-rounds=*)
		PARTY_ROUNDS="${1#--party-rounds=}"
		shift
		;;
	--party-after=*)
		PARTY_AFTER="${1#--party-after=}"
		shift
		;;
	--yolo | --force)
		EXTRA_AGENT_ARGS+=("$1")
		shift
		;;
	-*)
		EXTRA_AGENT_ARGS+=("$1")
		shift
		;;
	*)
		STORIES+=("$1")
		shift
		;;
	esac
done

if [[ ${HANDS_FREE} -eq 1 ]]; then
	has_yolo=0
	if ((${#EXTRA_AGENT_ARGS[@]} > 0)); then
		for a in "${EXTRA_AGENT_ARGS[@]}"; do
			[[ "${a}" == --yolo || "${a}" == --force ]] && has_yolo=1
		done
	fi
	if ((has_yolo == 0)); then
		if ((${#EXTRA_AGENT_ARGS[@]} > 0)); then
			EXTRA_AGENT_ARGS=(--yolo "${EXTRA_AGENT_ARGS[@]}")
		else
			EXTRA_AGENT_ARGS=(--yolo)
		fi
	fi
fi

if ! [[ "${PARTY_ROUNDS}" =~ ^[0-9]+$ ]]; then
	echo "error: PARTY_ROUNDS must be a non-negative integer (got ${PARTY_ROUNDS})" >&2
	exit 1
fi

if [[ ${DO_BOOTSTRAP} -eq 1 ]]; then
	if [[ -x "${SCRIPT_DIR}/bmad-bootstrap-method.sh" ]]; then
		"${SCRIPT_DIR}/bmad-bootstrap-method.sh"
	else
		bash "${SCRIPT_DIR}/bmad-bootstrap-method.sh"
	fi
fi

if ! command -v agent >/dev/null 2>&1; then
	if [[ "${PHASE}" == "gate" && ${SKIP_GO_FIX_LOOP} -eq 1 ]]; then
		:
	else
		echo "error: 'agent' not found on PATH." >&2
		exit 1
	fi
fi

if [[ "${PHASE}" == "gate" ]]; then
	:
elif [[ ! -f "${CREATE_STORY_WF}" || ! -f "${DEV_STORY_WF}" ]]; then
	echo "error: BMAD skill workflows missing under .claude/skills/." >&2
	exit 1
fi

if [[ ! -f "${CONFIG}" ]]; then
	echo "error: BMAD config missing: ${CONFIG}" >&2
	exit 1
fi

story_paths() {
	case "$1" in
	1.1) echo "${IMPL}/1-1-library-foundation.md" ;;
	1.2) echo "${IMPL}/1-2-capture-time-hash.md" ;;
	1.3) echo "${IMPL}/1-3-core-ingest.md" ;;
	1.4) echo "${IMPL}/1-4-collections-schema.md" ;;
	1.5) echo "${IMPL}/1-5-upload-confirm-receipt.md" ;;
	1.6) echo "${IMPL}/1-6-scan-cli.md" ;;
	1.7) echo "${IMPL}/1-7-import-cli.md" ;;
	1.8) echo "${IMPL}/1-8-drag-drop-upload.md" ;;
	2.1) echo "${IMPL}/2-1-app-shell-navigation-themes.md" ;;
	2.2) echo "${IMPL}/2-2-filter-strip.md" ;;
	2.3) echo "${IMPL}/2-3-thumbnail-grid-rating-badges.md" ;;
	2.4) echo "${IMPL}/2-4-review-loupe-keyboard-rating.md" ;;
	2.5) echo "${IMPL}/2-5-tags-bulk-review.md" ;;
	2.6) echo "${IMPL}/2-6-reject-undo-hidden-restore.md" ;;
	2.7) echo "${IMPL}/2-7-delete-quarantine.md" ;;
	2.8) echo "${IMPL}/2-8-collections-list-detail.md" ;;
	2.9) echo "${IMPL}/2-9-collection-crud-multi-assign.md" ;;
	2.10) echo "${IMPL}/2-10-quick-collection-assign.md" ;;
	2.11) echo "${IMPL}/2-11-layout-display-scaling-gate.md" ;;
	2.12) echo "${IMPL}/2-12-empty-states-error-tone.md" ;;
	3.1) echo "${IMPL}/3-1-share-preview-snapshot-mint.md" ;;
	3.2) echo "${IMPL}/3-2-loopback-http-token.md" ;;
	3.3) echo "${IMPL}/3-3-share-html-readonly.md" ;;
	3.4) echo "${IMPL}/3-4-share-privacy-wcag.md" ;;
	3.5) echo "${IMPL}/3-5-share-performance-abuse.md" ;;
	4.1) echo "${IMPL}/4-1-multi-asset-snapshot-packages.md" ;;
	*)
		echo "error: unknown story id '$1' (see epics.md; shorthands: 1.2-1.5, epic-1, all-epics, --help)" >&2
		exit 1
		;;
	esac
}

run_agent() {
	local prompt="$1"
	# With `set -u`, an empty EXTRA_AGENT_ARGS can make "${EXTRA_AGENT_ARGS[@]}"
	# error on some Bash builds; branch explicitly.
	if ((${#EXTRA_AGENT_ARGS[@]} > 0)); then
		agent --print --trust --workspace "${REPO}" "${EXTRA_AGENT_ARGS[@]}" "${prompt}"
	else
		agent --print --trust --workspace "${REPO}" "${prompt}"
	fi
}

# Run full module gate; on failure call agent with logs, repeat until green or max iterations.
ensure_workspace_green() {
	local max="${GO_FIX_MAX_ITERATIONS:-5}"
	local use_agent=1
	# With --skip-go-fix-loop: story phases skip the gate entirely (see maybe_ensure_workspace_green).
	# For --phase=gate --skip-go-fix-loop: one check pass, no agent (CI / no Cursor CLI).
	if [[ ${SKIP_GO_FIX_LOOP} -eq 1 ]]; then
		max=1
		use_agent=0
	fi
	if ! [[ "${max}" =~ ^[1-9][0-9]*$ ]]; then
		echo "error: GO_FIX_MAX_ITERATIONS must be a positive integer (got ${max})" >&2
		return 1
	fi
	local iter=0
	local logf
	while ((iter < max)); do
		logf="$(mktemp "${TMPDIR:-/tmp}/phototool-gate.XXXXXX")"
		: >"${logf}"
		local ok=1

		echo ">>> [workspace-gate] pass $((iter + 1))/${max}: tidy, fmt, verify, vet, test, build ..." >&2

		# Best-effort auto-fix (does not alone flip ok)
		{ echo "=== go mod tidy ===" && cd "${REPO}" && go mod tidy; } >>"${logf}" 2>&1 || true
		if ! (
			set -o pipefail
			{ echo "=== go fmt ./... ===" && cd "${REPO}" && go fmt ./...; } 2>&1 | tee -a "${logf}"
		); then
			ok=0
		fi
		if [[ ${ok} -eq 1 ]]; then
			if ! (
				set -o pipefail
				{ echo "=== go mod verify ===" && cd "${REPO}" && go mod verify; } 2>&1 | tee -a "${logf}"
			); then
				ok=0
			fi
		fi
		if [[ ${ok} -eq 1 ]]; then
			if ! (
				set -o pipefail
				{ echo "=== go vet ./... ===" && cd "${REPO}" && go vet ./...; } 2>&1 | tee -a "${logf}"
			); then
				ok=0
			fi
		fi
		if [[ ${ok} -eq 1 ]]; then
			if [[ "${GO_TEST_RACE:-0}" == "1" ]]; then
				if ! (
					set -o pipefail
					{ echo "=== go test -race ./... ===" && cd "${REPO}" && go test -race ./...; } 2>&1 | tee -a "${logf}"
				); then
					ok=0
				fi
			else
				if ! (
					set -o pipefail
					{ echo "=== go test ./... ===" && cd "${REPO}" && go test ./...; } 2>&1 | tee -a "${logf}"
				); then
					ok=0
				fi
			fi
		fi
		if [[ ${ok} -eq 1 ]]; then
			if ! (
				set -o pipefail
				{ echo "=== go build . ===" && cd "${REPO}" && go build .; } 2>&1 | tee -a "${logf}"
			); then
				ok=0
			fi
		fi
		if [[ ${ok} -eq 1 ]] && command -v staticcheck >/dev/null 2>&1 && [[ "${SKIP_STATICCHECK:-0}" != "1" ]]; then
			if ! (
				set -o pipefail
				{ echo "=== staticcheck ./... ===" && cd "${REPO}" && staticcheck ./...; } 2>&1 | tee -a "${logf}"
			); then
				ok=0
			fi
		fi
		if [[ ${ok} -eq 1 ]] && command -v golangci-lint >/dev/null 2>&1 && [[ "${GOLANGCI_LINT:-0}" == "1" ]]; then
			if ! (
				set -o pipefail
				{ echo "=== golangci-lint run ===" && cd "${REPO}" && golangci-lint run ./...; } 2>&1 | tee -a "${logf}"
			); then
				ok=0
			fi
		fi

		if [[ ${ok} -eq 1 ]]; then
			rm -f "${logf}"
			echo ">>> [workspace-gate] OK — all checks passed (${REPO})" >&2
			return 0
		fi
		iter=$((iter + 1))
		local snippet_file
		snippet_file="$(mktemp "${TMPDIR:-/tmp}/phototool-gate-snippet.XXXXXX")"
		tail -n 400 "${logf}" >"${snippet_file}" 2>/dev/null || true
		rm -f "${logf}"

		if [[ ${use_agent} -eq 0 ]]; then
			rm -f "${snippet_file}"
			echo "error: [workspace-gate] checks failed (agent repair disabled for this run)" >&2
			return 1
		fi
		if ! command -v agent >/dev/null 2>&1; then
			echo "error: [workspace-gate] checks failed and agent not on PATH" >&2
			rm -f "${snippet_file}"
			return 1
		fi

		echo ">>> [workspace-gate] failure ${iter}/${max}; invoking agent to repair..." >&2
		run_agent "photo-tool — workspace repair loop (${iter} of ${max}): make the repo pass the full gate.

Repository root: ${REPO}  (cd there for all commands)

The gate runs, in order:
1. go mod tidy
2. go fmt ./...
3. go mod verify
4. go vet ./...
5. go test ./...  (or go test -race ./... if race mode is enabled in CI — not required unless requested)
6. go build .
7. staticcheck ./...  (if staticcheck is installed; else skip)
8. golangci-lint run ./...  (only if GOLANGCI_LINT=1)

Requirements:
- Apply minimal fixes; match existing code style under internal/ and root.
- Do not delete or weaken tests to force green unless a test is wrong.
- Do not modify scripts/bmad-story-workflow.sh unless the failure is only fixable in automation itself.

Read the last ~400 lines of gate output from this file (plain text, not shell — it may contain quotes or backticks):
  ${snippet_file}

Summarize the failure, fix the repo, then delete that temp file if you no longer need it.

After edits, run the same commands locally if you can. The script will re-run the gate on the next iteration."
		rm -f "${snippet_file}"
	done
	echo "error: [workspace-gate] still failing after ${max} agent repair iteration(s)" >&2
	return 1
}

maybe_ensure_workspace_green() {
	command -v go >/dev/null 2>&1 || {
		echo "warning: [workspace-gate] skipped — go not on PATH" >&2
		return 0
	}
	# Skip gate after story work when user opted out — except dedicated gate phase.
	if [[ ${SKIP_GO_FIX_LOOP} -eq 1 && "${PHASE}" != "gate" ]]; then
		return 0
	fi
	ensure_workspace_green
}

phase_sprint_sync() {
	[[ -f "${SPRINT_WF}" ]] || {
		echo "warning: skip sprint-sync — missing ${SPRINT_WF}" >&2
		return 0
	}
	echo ">>> [bmad-sprint-planning] regenerate ${IMPL}/sprint-status.yaml" >&2
	run_agent "You are running **bmad-sprint-planning** for photo-tool.

Read and follow the workflow:
  ${SPRINT_WF}

Config: ${CONFIG}
Epics: ${PLAN}/epics.md
Output: refresh \`${IMPL}/sprint-status.yaml\` so every epic/story from epics.md appears with correct **development_status** keys.

**Headless mode:** at Continue menus, choose **Continue**. Preserve existing \`done\` statuses where they match epics; do not downgrade completed work."
}

phase_readiness() {
	[[ -f "${READINESS_WF}" ]] || {
		echo "warning: skip readiness — missing ${READINESS_WF}" >&2
		return 0
	}
	local out="${PLAN}/implementation-readiness-report.md"
	echo ">>> [check-implementation-readiness] → ${out}" >&2
	run_agent "You are running **bmad-check-implementation-readiness** for photo-tool.

Read and follow the workflow (all steps in order). At each menu that offers **[C] Continue**, choose **Continue** without waiting for a human — complete the workflow in this single headless run.

Workflow file:
  ${READINESS_WF}

Config: ${CONFIG}
Planning artifacts: ${PLAN}
Implementation artifacts: ${IMPL}

Write or update the consolidated report at:
  ${out}

Include: PRD + UX + architecture + epics alignment, gaps, and a clear **Go / No-Go** for starting Phase 4 implementation."
}

phase_create() {
	local ver="$1"
	local story_file
	story_file="$(story_paths "${ver}")"
	echo ">>> [create-story] Story ${ver} → ${story_file}" >&2
	run_agent "You are running the **BMAD create-story** workflow for photo-tool.

Read and follow the full workflow (all steps in order):
  ${CREATE_STORY_WF}

Configuration:
- Project root: ${REPO}
- Config: ${CONFIG}
- planning_artifacts: ${PLAN}
- implementation_artifacts: ${IMPL}
- Epics: ${PLAN}/epics.md

Target: **Epic ${ver%%.*} Story ${ver}** (see epics.md section '### Story ${ver}:').

Output: update or create the story spec at:
  ${story_file}

Use the story template from:
  ${REPO}/.claude/skills/bmad-create-story/template.md

If the file already exists, **merge**: preserve Status; refresh Acceptance Criteria and Tasks from epics if needed; add missing Dev Notes and References.

Update \`${IMPL}/sprint-status.yaml\` so the matching key (e.g. 1-2-capture-time-hash) is **ready-for-dev** after the spec is complete.

Do not implement application code in this phase—story specification only."
}

phase_dev() {
	local ver="$1"
	local story_file
	story_file="$(story_paths "${ver}")"
	if [[ ! -f "${story_file}" ]]; then
		echo "error: story file missing: ${story_file} (run --phase=create first)" >&2
		exit 1
	fi
	echo ">>> [dev-story] Story ${ver} → ${story_file}" >&2
	run_agent "You are running the **BMAD dev-story** workflow for photo-tool.

Read and follow the full workflow (all steps in order):
  ${DEV_STORY_WF}

Story file (source of truth):
  ${story_file}

Project root: ${REPO}
Config: ${CONFIG}

Rules:
- Implement until **all acceptance criteria** and **tasks** in the story are satisfied.
- Only modify the story file in allowed sections (tasks checkboxes, Dev Agent Record, status) per dev-story workflow.
- Match existing Go patterns under internal/.
- Run \`go test ./...\` and \`go build .\`; fix failures.

HALT only on true blockers (e.g. missing prerequisite story); otherwise complete the story in this run."
}

phase_code_review() {
	local ver="$1"
	local story_file
	story_file="$(story_paths "${ver}")"
	[[ -f "${CODE_REVIEW_WF}" ]] || {
		echo "warning: skip code-review — missing ${CODE_REVIEW_WF}" >&2
		return 0
	}
	echo ">>> [bmad-code-review] Story ${ver}" >&2
	run_agent "You are running **bmad-code-review** for photo-tool.

Read and follow the code-review workflow (all step files under the skill, in order):
  ${CODE_REVIEW_WF}

Config: ${CONFIG}
Sprint status: ${IMPL}/sprint-status.yaml
Story context: ${story_file}

Scope the review to changes and code paths introduced for **Epic ${ver%%.*} Story ${ver}**. Use adversarial layers (Blind Hunter, Edge Case Hunter, Acceptance Auditor) and produce triaged, actionable findings.

If the workflow expects human checkpoints, proceed with **Continue** autonomously for this headless run.

End with a short summary table: Severity × Category × Recommended action."
}

# Prompt text for AC verification; agent must end with VERIFY_ANY_FAIL=yes|no (see header).
phase_verify_prompt() {
	local ver="$1"
	local story_file="$2"
	cat <<PVHDR
Verification pass for photo-tool **Story ${ver}**.

Read the story file:
  ${story_file}

For **each** acceptance criterion, respond with: AC#, PASS or FAIL, evidence (test name, manual step, or file:line).
Run \`go test ./...\` and \`go build .\` from ${REPO} and include exit status.

## Machine-readable outcome (required)
On the **last line** of your reply, output **exactly** one of these two lines and nothing else on that line:
VERIFY_ANY_FAIL=no
VERIFY_ANY_FAIL=yes
Use \`yes\` if **any** acceptance criterion is FAIL. Use \`no\` only if **all** are PASS.

If any AC is FAIL, you may still fix in this same reply before printing the VERIFY_ANY_FAIL line; the script may also run a dedicated repair pass when you output \`yes\`.
PVHDR
}

phase_verify_repair_prompt() {
	local ver="$1"
	local story_file="$2"
	local snippet="$3"
	{
		echo "photo-tool — **verification repair** for Story ${ver}."
		echo ""
		echo "Story file:"
		echo "  ${story_file}"
		echo ""
		echo "Project root: ${REPO}"
		echo ""
		echo "Previous verification output (last 200 lines):"
		echo '```'
		printf '%s\n' "${snippet}"
		echo '```'
		echo ""
		echo "Apply **minimal** code + test changes so **every** AC in the story file would PASS (re-read the story)."
		echo "Prefer guards and accurate user-visible copy over weakening requirements."
		echo "Run \`go test ./...\` and \`go build .\` when possible."
		echo "Do not change scripts/bmad-story-workflow.sh unless the defect is specifically in automation."
	}
}

run_verify_agent_tee() {
	local ver="$1"
	local story_file="$2"
	local logf="$3"
	echo ">>> [verify AC] Story ${ver} → ${story_file}" >&2
	run_agent "$(phase_verify_prompt "${ver}" "${story_file}")" 2>&1 | tee "${logf}"
}

phase_verify() {
	local ver="$1"
	local story_file
	story_file="$(story_paths "${ver}")"
	local max="${VERIFY_AC_FIX_MAX:-5}"
	if ! [[ "${max}" =~ ^[1-9][0-9]*$ ]]; then
		echo "error: VERIFY_AC_FIX_MAX must be a positive integer (got ${max})" >&2
		exit 1
	fi

	local logf
	local status_line
	local round
	local snippet

	logf="$(mktemp "${TMPDIR:-/tmp}/phototool-verify-${ver}.XXXXXX")"
	run_verify_agent_tee "${ver}" "${story_file}" "${logf}"
	status_line="$(grep -E '^VERIFY_ANY_FAIL=(yes|no)$' "${logf}" 2>/dev/null | tail -n 1 || true)"

	if [[ "${status_line}" == "VERIFY_ANY_FAIL=no" ]]; then
		rm -f "${logf}"
		return 0
	fi

	if [[ ${SKIP_VERIFY_AUTOFIX} -eq 1 ]]; then
		if [[ "${status_line}" != "VERIFY_ANY_FAIL=yes" && "${status_line}" != "VERIFY_ANY_FAIL=no" ]]; then
			echo "warning: [verify-autofix] missing VERIFY_ANY_FAIL line at end of agent output; enable autofix or add the line for automation." >&2
		fi
		rm -f "${logf}"
		return 0
	fi

	if [[ "${status_line}" != "VERIFY_ANY_FAIL=yes" ]]; then
		echo "warning: [verify-autofix] expected VERIFY_ANY_FAIL=yes|no on last line; got '${status_line}' — skipping repair loop." >&2
		rm -f "${logf}"
		return 0
	fi

	round=0
	while ((round < max)); do
		round=$((round + 1))
		snippet="$(tail -n 200 "${logf}" 2>/dev/null || true)"
		echo ">>> [verify-autofix] VERIFY_ANY_FAIL=yes — repair round ${round}/${max}" >&2
		rm -f "${logf}"
		run_agent "$(phase_verify_repair_prompt "${ver}" "${story_file}" "${snippet}")"

		logf="$(mktemp "${TMPDIR:-/tmp}/phototool-verify-${ver}.XXXXXX")"
		run_verify_agent_tee "${ver}" "${story_file}" "${logf}"
		status_line="$(grep -E '^VERIFY_ANY_FAIL=(yes|no)$' "${logf}" 2>/dev/null | tail -n 1 || true)"

		if [[ "${status_line}" == "VERIFY_ANY_FAIL=no" ]]; then
			rm -f "${logf}"
			return 0
		fi
		if [[ "${status_line}" != "VERIFY_ANY_FAIL=yes" ]]; then
			echo "warning: [verify-autofix] after repair, expected VERIFY_ANY_FAIL line; got '${status_line}'" >&2
			rm -f "${logf}"
			return 0
		fi
	done

	rm -f "${logf}"
	echo "error: [verify-autofix] VERIFY_ANY_FAIL still yes after ${max} repair round(s)" >&2
	return 1
}

phase_retrospective() {
	[[ -f "${RETRO_WF}" ]] || {
		echo "warning: skip retrospective — missing ${RETRO_WF}" >&2
		return 0
	}
	local manifest="${REPO}/_bmad/_config/agent-manifest.csv"
	if [[ ! -f "${manifest}" ]]; then
		echo "warning: retrospective workflow expects ${manifest}. Run ./scripts/bmad-bootstrap-method.sh or \`npx bmad-method install\`. Running simplified retro prompt anyway." >&2
	fi
	local out="${IMPL}/epic-${RETRO_EPIC}-retrospective-$(date +%Y%m%d).md"
	echo ">>> [bmad-retrospective] Epic ${RETRO_EPIC} → ${out}" >&2
	run_agent "You are running **bmad-retrospective** for photo-tool **Epic ${RETRO_EPIC}**.

Read and follow:
  ${RETRO_WF}

Config: ${CONFIG}
Sprint status: ${IMPL}/sprint-status.yaml
Epics: ${PLAN}/epics.md (Epic ${RETRO_EPIC} section)
Architecture / PRD under ${PLAN}/ as needed.

**Headless mode:** Where the workflow asks for user facilitation, synthesize a concise retrospective from sprint-status + epics + recent story files instead of waiting for replies. No blame; systems focus; actionable improvements.

Write the retrospective document to:
  ${out}

Include: what went well, what to improve, risks for the next epic, and 3–5 concrete action items."
}

party_runs_on_hook() {
	local want="$1"
	local _h _IFS="${IFS}"
	IFS=',' read -ra _hooks <<< "${PARTY_AFTER}"
	IFS="${_IFS}"
	for _h in "${_hooks[@]}"; do
		_h="${_h#"${_h%%[![:space:]]*}"}"
		_h="${_h%"${_h##*[![:space:]]}"}"
		_h="${_h// /}"
		[[ "${_h}" == "${want}" ]] && return 0
	done
	return 1
}

party_mode_prompt() {
	local ver="$1" story_file="$2" hook="$3" pr="$4" total="$5"
	local manifest="${REPO}/_bmad/_config/agent-manifest.csv"
	cat <<EOF
photo-tool — **BMAD party mode** (automated headless), hook **${hook}**, Epic ${ver%%.*} Story ${ver}, session **${pr}/${total}**.

Read and follow:
  ${PARTY_SKILL}
Agent roster (if present):
  ${manifest}
Story file:
  ${story_file}
Epics:
  ${PLAN}/epics.md

**Automation instructions**
- Run one **party round** per the skill: pick 3–4 agents whose roles fit **${hook}** (\`create\` → spec/AC/tasks/risks; \`dev\` → code, tests, UX, edge cases).
- If independent subagent processes are not available, **simulate** the roundtable with clearly separated agent voices (icon + name), genuine disagreement, then a short **Orchestrator synthesis** with **concrete next edits**.
- **Apply** those edits immediately (story markdown, Go, tests, \`sprint-status.yaml\` if needed) using tools. Do not wait for human approval.
- Session **${pr}/${total}**: if ${pr} is greater than 1, **deepen or challenge** prior conclusions; do not repeat the same analysis verbatim.
- Where BMAD workflows expect human checkpoints, proceed with **Continue** autonomously.
EOF
}

phase_party_rounds() {
	local ver="$1" story_file="$2" hook="$3"
	local pr
	[[ -f "${PARTY_SKILL}" ]] || {
		echo "warning: party mode skipped — missing ${PARTY_SKILL}" >&2
		return 0
	}
	((PARTY_ROUNDS > 0)) || return 0
	for ((pr = 1; pr <= PARTY_ROUNDS; pr++)); do
		echo ">>> [party-mode] Story ${ver} hook=${hook} round ${pr}/${PARTY_ROUNDS}" >&2
		run_agent "$(party_mode_prompt "${ver}" "${story_file}" "${hook}" "${pr}" "${PARTY_ROUNDS}")"
	done
}

run_one_story() {
	local ver="$1"
	local story_file
	story_file="$(story_paths "${ver}")"
	case "${PHASE}" in
	all)
		phase_create "${ver}"
		if ((PARTY_ROUNDS > 0)) && party_runs_on_hook create; then
			phase_party_rounds "${ver}" "${story_file}" create
		fi
		phase_dev "${ver}"
		if ((PARTY_ROUNDS > 0)) && party_runs_on_hook dev; then
			phase_party_rounds "${ver}" "${story_file}" dev
		fi
		if [[ ${SKIP_CODE_REVIEW} -eq 0 ]]; then
			phase_code_review "${ver}"
		fi
		phase_verify "${ver}"
		;;
	create)
		phase_create "${ver}"
		if ((PARTY_ROUNDS > 0)) && party_runs_on_hook create; then
			phase_party_rounds "${ver}" "${story_file}" create
		fi
		;;
	dev)
		phase_dev "${ver}"
		if ((PARTY_ROUNDS > 0)) && party_runs_on_hook dev; then
			phase_party_rounds "${ver}" "${story_file}" dev
		fi
		;;
	review) phase_code_review "${ver}" ;;
	verify) phase_verify "${ver}" ;;
	readiness) phase_readiness ;;
	retro) phase_retrospective ;;
	*)
		echo "error: unknown --phase=${PHASE} (use all|create|dev|review|verify|readiness|retro|sprint|gate)" >&2
		exit 1
		;;
	esac
}

if [[ ${#STORIES[@]} -eq 0 ]]; then
	if [[ "${PHASE}" == "readiness" ]]; then
		phase_readiness
		echo "Done." >&2
		exit 0
	fi
	if [[ "${PHASE}" == "retro" ]]; then
		phase_retrospective
		echo "Done." >&2
		exit 0
	fi
	if [[ "${PHASE}" == "sprint" ]]; then
		phase_sprint_sync
		echo "Done." >&2
		exit 0
	fi
	if [[ "${PHASE}" == "gate" ]]; then
		maybe_ensure_workspace_green
		echo "Done." >&2
		exit 0
	fi
	echo "error: specify story id(s), e.g. 1.6, epic-2, --all, or --all-epics" >&2
	echo "Try: $0 --help" >&2
	exit 1
fi

EXPANDED=()
for s in "${STORIES[@]}"; do
	case "${s}" in
	1.2-1.5) EXPANDED+=(1.2 1.3 1.4 1.5) ;;
	1.1-1.8) EXPANDED+=(1.1 1.2 1.3 1.4 1.5 1.6 1.7 1.8) ;;
	epic-1) EXPANDED+=(1.1 1.2 1.3 1.4 1.5 1.6 1.7 1.8) ;;
	2.1-2.12) EXPANDED+=(2.1 2.2 2.3 2.4 2.5 2.6 2.7 2.8 2.9 2.10 2.11 2.12) ;;
	epic-2) EXPANDED+=(2.1 2.2 2.3 2.4 2.5 2.6 2.7 2.8 2.9 2.10 2.11 2.12) ;;
	3.1-3.5) EXPANDED+=(3.1 3.2 3.3 3.4 3.5) ;;
	epic-3) EXPANDED+=(3.1 3.2 3.3 3.4 3.5) ;;
	epic-4) EXPANDED+=(4.1) ;;
	all-epics)
		EXPANDED+=(
			1.1 1.2 1.3 1.4 1.5 1.6 1.7 1.8
			2.1 2.2 2.3 2.4 2.5 2.6 2.7 2.8 2.9 2.10 2.11 2.12
			3.1 3.2 3.3 3.4 3.5
			4.1
		)
		;;
	1.6-thru-4.1)
		EXPANDED+=(
			1.6 1.7 1.8
			2.1 2.2 2.3 2.4 2.5 2.6 2.7 2.8 2.9 2.10 2.11 2.12
			3.1 3.2 3.3 3.4 3.5
			4.1
		)
		;;
	*) EXPANDED+=("${s}") ;;
	esac
done
STORIES=("${EXPANDED[@]}")

# Dedicated gate phase ignores extra story ids (full module check only).
if [[ "${PHASE}" == "gate" ]]; then
	STORIES=()
fi

N="${#STORIES[@]}"
RUN_READINESS=0
RUN_RETRO=0

if [[ ${FULL_PARITY} -eq 1 ]]; then
	RUN_READINESS=1
	RUN_RETRO=1
fi
if [[ ${N} -gt 1 ]]; then
	RUN_READINESS=1
	RUN_RETRO=1
fi
[[ ${FORCE_READINESS} -eq 1 ]] && RUN_READINESS=1
[[ ${FORCE_RETRO} -eq 1 ]] && RUN_RETRO=1
[[ ${SKIP_READINESS} -eq 1 ]] && RUN_READINESS=0
[[ ${SKIP_RETRO} -eq 1 ]] && RUN_RETRO=0

if [[ "${PHASE}" == "all" && ${SPRINT_SYNC} -eq 1 ]]; then
	phase_sprint_sync
fi

if [[ "${PHASE}" == "all" && ${RUN_READINESS} -eq 1 ]]; then
	phase_readiness
fi

for ver in "${STORIES[@]}"; do
	run_one_story "${ver}"
	echo "=== Story ${ver} phases (${PHASE}) complete ===" >&2
done

if [[ "${PHASE}" == "all" && ${RUN_RETRO} -eq 1 ]]; then
	phase_retrospective
fi

# One full workspace gate after story work (or --phase=gate with stray story ids cleared above).
if [[ "${PHASE}" == "all" || "${PHASE}" == "dev" || "${PHASE}" == "verify" || "${PHASE}" == "review" || "${PHASE}" == "gate" ]]; then
	maybe_ensure_workspace_green
fi

# #region agent log
_dbg_ts2="$(($(date +%s) * 1000))"
{
	printf '%s\n' "{\"sessionId\":\"8dfdc8\",\"hypothesisId\":\"H-tail\",\"location\":\"bmad-story-workflow.sh:pre-done\",\"message\":\"reached end of main path before Done echo\",\"data\":{\"phase\":\"${PHASE}\"},\"timestamp\":${_dbg_ts2}}"
} >>"${REPO}/.cursor/debug-8dfdc8.log" 2>/dev/null || true
# #endregion
echo "Done." >&2

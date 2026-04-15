#!/usr/bin/env bash
# YOLO driver for _bmad-output/planning-artifacts/initiative-fyne-image-first-bmad.md
#
# - Defaults: accept everything headless (--yolo), Party Mode 2 rounds after create + dev (per story).
# - Instructs agent to apply party / skill outputs (no interactive approval).
# - Story batch: 1.5 → 2.1 → 2.2 → 2.3 → 2.11 (Upload, shell, filters, grid, NFR evidence).
#
# Prerequisites: `agent` on PATH; BMAD skills under .claude/skills/; `_bmad/bmm/config.yaml`.
#
# Optional env (all default OFF / sensible):
#   INITIATIVE_SKIP_PHASE1=1     Skip UX/Arch party pre-pass (planning artifacts only)
#   INITIATIVE_SKIP_MAIN=1       Skip bmad-story-workflow story batch (create+dev+review+verify each)
#   INITIATIVE_SKIP_TEA=1        Skip Murat / test-architect notes file
#   INITIATIVE_SKIP_CHECKPOINT=1 Skip checkpoint markdown summary
#   INITIATIVE_SKIP_SPRINT=1     Skip bmad-sprint-planning sync
#   INITIATIVE_SKIP_RETRO=1    Skip retrospective
#   INITIATIVE_RETRO_EPIC=2    Epic number for retro (default 2)
#   PARTY_ROUNDS=2             Passed through to bmad-story-workflow (--party-rounds)
#
# Usage:
#   ./scripts/initiative-fyne-image-first-yolo.sh
#   INITIATIVE_SKIP_PHASE1=1 ./scripts/initiative-fyne-image-first-yolo.sh
#
# Shell: line continuations — backslash must be the last character on the line.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO="$(cd "${SCRIPT_DIR}/.." && pwd)"
WF="${SCRIPT_DIR}/bmad-story-workflow.sh"
INIT="${REPO}/_bmad-output/planning-artifacts/initiative-fyne-image-first-bmad.md"
EPICS_V2="${REPO}/_bmad-output/planning-artifacts/epics-v2-ux-aligned-2026-04-14.md"
CONFIG="${REPO}/_bmad/bmm/config.yaml"
SKILLS="${REPO}/.claude/skills"
PARTY_SKILL="${SKILLS}/bmad-party-mode/SKILL.md"
TEA_SKILL="${SKILLS}/bmad-tea/SKILL.md"
CHECKPOINT_SKILL="${SKILLS}/bmad-checkpoint-preview/SKILL.md"

# Initiative slice order (same as initiative doc)
STORIES=(1.5 2.1 2.2 2.3 2.11)

PARTY_ROUNDS="${PARTY_ROUNDS:-2}"
INITIATIVE_RETRO_EPIC="${INITIATIVE_RETRO_EPIC:-2}"

log() { printf '%s\n' "$*" >&2; }

require_agent() {
	if ! command -v agent >/dev/null 2>&1; then
		log "error: 'agent' not found on PATH (Cursor CLI)."
		exit 1
	fi
}

require_wf() {
	if [[ ! -f "${WF}" ]]; then
		log "error: missing ${WF}"
		exit 1
	fi
	bash -n "${WF}" || {
		log "error: bash -n failed for ${WF}"
		exit 1
	}
}

run_agent_yolo() {
	local prompt="$1"
	agent --print --trust --workspace "${REPO}" --yolo "${prompt}"
}

phase1_spec_party() {
	[[ -f "${INIT}" ]] || {
		log "warning: skip phase 1 — missing ${INIT}"
		return 0
	}
	[[ -f "${PARTY_SKILL}" ]] || {
		log "warning: skip phase 1 — missing ${PARTY_SKILL}"
		return 0
	}
	log ">>> [initiative] Phase 1 — UX + Arch party (2 rounds), planning artifacts only"
	run_agent_yolo "photo-tool — **Initiative Fyne image-first**, Phase 1 (headless YOLO).

Read and follow:
  ${PARTY_SKILL}

Also read:
  ${INIT}
  ${EPICS_V2} (if present — use for FR/UX-DR anchors)

**Automation rules**
- Run **two** party rounds (session 1/2 then 2/2). Round 2 must **challenge or deepen** round 1, not repeat it.
- Voices: at minimum **UX designer** + **architect** (simulate subagents if needed).
- **Apply** synthesized outcomes immediately: edits **only** under \`_bmad-output/planning-artifacts/\` (markdown notes, optional small clarifications to initiative or epics). **No** production Go changes in this phase.
- Where BMAD workflows ask for human input, choose **Continue** / default without waiting.
- End with a 5-bullet summary of what you changed."
}

phase_main_stories() {
	log ">>> [initiative] Main batch — per story: create → party×${PARTY_ROUNDS} → dev → party×${PARTY_ROUNDS} → code-review → verify (hands-free)"
	# Single workflow invocation: sequential stories, one workspace gate at end.
	# Do NOT skip code review (initiative Phase 4). Readiness/retro handled separately.
	PARTY_ROUNDS="${PARTY_ROUNDS}" bash "${WF}" \
		--hands-free \
		--party \
		--party-rounds="${PARTY_ROUNDS}" \
		--skip-readiness \
		--skip-retro \
		"${STORIES[@]}"
}

phase_tea_notes() {
	[[ -f "${TEA_SKILL}" ]] || {
		log "warning: skip TEA phase — missing ${TEA_SKILL}"
		return 0
	}
	local out="${REPO}/_bmad-output/planning-artifacts/initiative-fyne-image-first-tea-notes.md"
	log ">>> [initiative] TEA / test strategy notes → ${out}"
	run_agent_yolo "photo-tool — **bmad-tea** (Murat / test architect), headless YOLO.

Read and follow:
  ${TEA_SKILL}

Context:
  ${INIT}
  Stories touched: ${STORIES[*]}

**Apply**
- Create or update \`${out}\` with **test strategy** for the Fyne image-first initiative: risks, AC-level checks, \`go test\` packages to stress, headless Fyne vs manual matrix (NFR-01), and \`-race\` suggestions for ingest/grid if relevant (**UX-DR17**).
- Do not remove existing initiative doc content; add a linked section or new file only.
- Use **Continue** for any menu prompts. **Apply** edits without asking for confirmation."
}

phase_checkpoint() {
	[[ -f "${CHECKPOINT_SKILL}" ]] || {
		log "warning: skip checkpoint — missing ${CHECKPOINT_SKILL}"
		return 0
	}
	local out="${REPO}/_bmad-output/planning-artifacts/initiative-fyne-image-first-checkpoint.md"
	log ">>> [initiative] Checkpoint preview → ${out}"
	run_agent_yolo "photo-tool — **bmad-checkpoint-preview**, headless YOLO.

Read and follow:
  ${CHECKPOINT_SKILL}

Context:
  ${INIT}
  Recent story IDs: ${STORIES[*]}
  Optional TEA notes: _bmad-output/planning-artifacts/initiative-fyne-image-first-tea-notes.md (if exists)

**Apply**
- Write \`${out}\`: what changed in repo, **where to look** (files), **manual smoke** steps for Upload + Review, and **open risks**.
- Use **Continue** for checkpoints. No chat questions to the user."
}

phase_sprint_sync() {
	log ">>> [initiative] Sprint planning sync (sprint-status.yaml)"
	PARTY_ROUNDS=0 bash "${WF}" --hands-free --phase=sprint
}

phase_retro() {
	log ">>> [initiative] Retrospective (Epic ${INITIATIVE_RETRO_EPIC})"
	bash "${WF}" --hands-free --phase=retro --retro-epic="${INITIATIVE_RETRO_EPIC}"
}

[[ -f "${CONFIG}" ]] || {
	log "error: BMAD config missing: ${CONFIG}"
	exit 1
}

require_agent
require_wf

log "=== Fyne image-first initiative (YOLO) — repo ${REPO} ==="
log "Stories: ${STORIES[*]} | PARTY_ROUNDS=${PARTY_ROUNDS}"

if [[ "${INITIATIVE_SKIP_PHASE1:-0}" != "1" ]]; then
	phase1_spec_party
else
	log ">>> [initiative] Skipping Phase 1 (INITIATIVE_SKIP_PHASE1=1)"
fi

if [[ "${INITIATIVE_SKIP_MAIN:-0}" != "1" ]]; then
	phase_main_stories
else
	log ">>> [initiative] Skipping main story batch (INITIATIVE_SKIP_MAIN=1)"
fi

if [[ "${INITIATIVE_SKIP_TEA:-0}" != "1" ]]; then
	phase_tea_notes
fi

if [[ "${INITIATIVE_SKIP_CHECKPOINT:-0}" != "1" ]]; then
	phase_checkpoint
fi

if [[ "${INITIATIVE_SKIP_SPRINT:-0}" != "1" ]]; then
	phase_sprint_sync
fi

if [[ "${INITIATIVE_SKIP_RETRO:-0}" != "1" ]]; then
	phase_retro
fi

log "=== Initiative YOLO run finished ==="

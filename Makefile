.PHONY: all build run test test-e2e test-ci judge-bundle ux-judge-loop extended-test extended-test-loop scale-test extended-test-scale extended-test-scale-ux scale-seed ux-scale-capture scale-report-open tidy clean fmt vet gate scripts-check

BIN_DIR := bin
BINARY := $(BIN_DIR)/photo-tool

all: build

build:
	@mkdir -p $(BIN_DIR)
	go build -o $(BINARY) .

run:
	go run .

test:
	go test ./...

# Black-box CLI: builds real binary (or PHOTO_TOOL_E2E_BIN) and runs phototool scan/import against a temp library.
test-e2e:
	go test ./tests/e2e/... -count=1

# Fyne software driver; includes main_fyne_ci_test.go (NewWithID / preferences regression).
test-ci:
	go test -tags ci ./...

# Optional: assemble _bmad-output/test-artifacts/judge-bundles/<stamp>/ (logs, rubric, UI PNGs, manifest).
judge-bundle:
	./scripts/assemble-judge-bundle.sh

# Local-only: bundle → vision judge agent → fix agent loop (requires Cursor `agent` on PATH; not for CI).
ux-judge-loop:
	./scripts/ux-judge-loop.sh

# Extended matrix: functional + real-app UX capture (single pass; requires DISPLAY or xvfb).
extended-test:
	./scripts/extended-test-run.sh --all

# Extended matrix hands-free loop until EXTENDED_RESULT=pass (requires agent CLI; not for CI).
extended-test-loop:
	./scripts/extended-test-loop.sh

# Scale unit tests (heavy rows skip under go test -short).
scale-test:
	go test ./tests/fixture/... ./tests/e2e/... -run 'TestScale_' -count=1 -short

# Real-binary scale/edge/layout UX capture (requires DISPLAY or xvfb).
extended-test-scale-ux:
	./scripts/extended-test-run.sh --layer=scale_ux --judges

# Functional scale tests only (no PNG capture).
extended-test-scale:
	./scripts/extended-test-run.sh --layer=scale

# Seed a tiered library: make scale-seed TIER=S5 OUT=/tmp/lib-500
TIER ?= S5
OUT_LIB ?= /tmp/photo-tool-scale-lib
scale-seed:
	go run ./tests/extended/cmd/seed-library -out $(OUT_LIB) -tier $(TIER)

# Real-binary scale spot capture only.
ux-scale-capture:
	PHOTO_TOOL_UX_FIXTURE_SCALE=S5 PHOTO_TOOL_UX_CAPTURE_FLOWS=scale_spot PHOTO_TOOL_UX_CAPTURE_APP_MODE=real_binary \
		PHOTO_TOOL_GUI_E2E_LINUX=1 ./bin/photo-tool

# Open interactive scale report (RUN=path to extended run dir).
RUN ?=
scale-report-open:
	@test -n "$(RUN)" || (echo "usage: make scale-report-open RUN=path/to/extended-run" && exit 1)
	xdg-open "$(RUN)/scale-report.html" 2>/dev/null || open "$(RUN)/scale-report.html" 2>/dev/null || echo "Open $(RUN)/scale-report.html in a browser"

tidy:
	go mod tidy

fmt:
	go fmt ./...

vet:
	go vet ./...

# Full module gate (tidy, fmt, verify, vet, test, build; optional staticcheck / golangci-lint). See script header.
gate:
	./scripts/bmad-story-workflow.sh --phase=gate

# Catch unclosed quotes / truncated edits before long BMAD runs (see scripts/bmad-story-workflow.sh header).
scripts-check:
	bash -n scripts/bmad-story-workflow.sh scripts/assemble-judge-bundle.sh scripts/ux-judge-loop.sh scripts/extended-test-setup.sh scripts/extended-test-run.sh scripts/extended-test-loop.sh scripts/extended-test-matrix-gen.sh

clean:
	rm -rf $(BIN_DIR)

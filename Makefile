.PHONY: all build run test test-e2e test-ci judge-bundle ux-judge-loop tidy clean fmt vet gate scripts-check

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
	bash -n scripts/bmad-story-workflow.sh scripts/assemble-judge-bundle.sh scripts/ux-judge-loop.sh

clean:
	rm -rf $(BIN_DIR)

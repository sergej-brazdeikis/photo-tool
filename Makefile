.PHONY: all build run test test-ci tidy clean fmt vet gate

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

# Fyne software driver; includes main_fyne_ci_test.go (NewWithID / preferences regression).
test-ci:
	go test -tags ci ./...

tidy:
	go mod tidy

fmt:
	go fmt ./...

vet:
	go vet ./...

# Full module gate (tidy, fmt, verify, vet, test, build; optional staticcheck / golangci-lint). See script header.
gate:
	./scripts/bmad-story-workflow.sh --phase=gate

clean:
	rm -rf $(BIN_DIR)

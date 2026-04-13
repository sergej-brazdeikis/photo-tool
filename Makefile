.PHONY: all build run test tidy clean fmt vet

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

tidy:
	go mod tidy

fmt:
	go fmt ./...

vet:
	go vet ./...

clean:
	rm -rf $(BIN_DIR)

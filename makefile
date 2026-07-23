BINARY   := lazytmux
VERSION  ?= v0.1.8
COMMIT   := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS  := -X main.version=$(VERSION) -X main.gitCommit=$(COMMIT)

.PHONY: run build test test-verbose lint fmt quality build-all

run:
	go run -ldflags "$(LDFLAGS)" ./cmd

build: quality
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) ./cmd

test:
	go test -race -cover ./...

test-verbose:
	go test -race -v -cover ./...

fmt:
	gofumpt -w -l . 2>/dev/null || true
	go fmt ./...

lint:
	golangci-lint run ./...

quality: fmt
	go vet ./...

build-all:
	goreleaser build --snapshot --clean

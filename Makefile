VERSION ?= $(shell cat VERSION 2>/dev/null | tr -d '\n')
LDFLAGS := -X github.com/hrodrig/kzero/internal/cli.Version=$(VERSION)

.PHONY: build lint lint-fix test
build:
	go build -trimpath -ldflags "$(LDFLAGS)" -o bin/kzero ./cmd/kzero

lint:
	test -z "$$(gofmt -s -l . 2>/dev/null)" || (echo "gofmt needed"; exit 1)
	go vet ./...

lint-fix:
	gofmt -s -w .
	go vet ./...

test:
	go test ./...

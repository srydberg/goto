BINARY  := goto
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: all build test vet fmt cover install clean

all: fmt vet test build

build:
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) ./cmd/goto

test:
	go test -race ./...

vet:
	go vet ./...

fmt:
	gofmt -s -w .

cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

install:
	go install -ldflags "$(LDFLAGS)" ./cmd/goto

clean:
	rm -rf bin coverage.out

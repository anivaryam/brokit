BINARY_NAME=brokit
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-s -w -X main.version=$(VERSION)"

build:
	go build $(LDFLAGS) -o bin/$(BINARY_NAME) ./cmd/brokit

install: build
	cp bin/$(BINARY_NAME) ~/.local/bin/$(BINARY_NAME)

clean:
	rm -rf bin/

test:
	go test ./... -race -count=1 -coverprofile=coverage.out -covermode=atomic

lint:
	golangci-lint run

.PHONY: build install clean test lint

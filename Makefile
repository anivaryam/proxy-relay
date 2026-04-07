.PHONY: build-server build-client install test clean

build-server:
	go build -o bin/proxy-relay-server ./cmd/server

VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-s -w -X main.version=$(VERSION)"

build-client:
	go build $(LDFLAGS) -o bin/proxy-relay ./cmd/proxy-relay

install: build-client
	cp bin/proxy-relay ~/.local/bin/proxy-relay

test:
	go test ./... -race -count=1

clean:
	rm -rf bin/

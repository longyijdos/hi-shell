.PHONY: build test install-local

build:
	go build -o hi-shell ./cmd/hi-shell

test:
	go test ./...

install-local:
	sh ./scripts/install.sh

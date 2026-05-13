.PHONY: build test check install-hooks install-local

build:
	go build -o hi-shell ./cmd/hi-shell

test:
	go test ./...

check:
	./scripts/check.sh

install-hooks:
	./scripts/install-hooks.sh

install-local:
	./scripts/install.sh

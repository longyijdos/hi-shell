.PHONY: build test install-local

build:
	go build -o hi ./cmd/hi

test:
	go test ./...

install-local:
	sh ./scripts/install.sh

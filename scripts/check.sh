#!/bin/sh
set -eu

go test ./...
go vet ./...
zsh -n shell/hi.zsh
sh -n scripts/install.sh

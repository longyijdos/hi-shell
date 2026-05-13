#!/bin/sh
set -eu

go test ./...
go vet ./...
zsh -n shell/hi.zsh
for script in scripts/*.sh; do
  sh -n "$script"
done

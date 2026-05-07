#!/bin/sh
set -eu

repo="longyijdos/hi-shell"
bin_dir="${HI_BIN_DIR:-"$HOME/.local/bin"}"
bin_path="$bin_dir/hi"

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
repo_root=$(CDPATH= cd -- "$script_dir/.." && pwd)

mkdir -p "$bin_dir"

build_from_source() {
  if ! command -v go >/dev/null 2>&1; then
    echo "go is required to install from source" >&2
    exit 1
  fi

  (
    cd "$repo_root"
    go build -ldflags "-X main.version=dev" -o "$bin_path" ./cmd/hi
  )
}

download_release() {
  os=$(uname -s)
  arch=$(uname -m)

  case "$os" in
    Darwin|Linux) ;;
    *)
      echo "unsupported OS: $os" >&2
      exit 1
      ;;
  esac

  case "$arch" in
    x86_64|amd64) arch="x86_64" ;;
    arm64|aarch64) arch="arm64" ;;
    *)
      echo "unsupported architecture: $arch" >&2
      exit 1
      ;;
  esac

  if ! command -v curl >/dev/null 2>&1; then
    echo "curl is required to download a release" >&2
    exit 1
  fi
  if ! command -v tar >/dev/null 2>&1; then
    echo "tar is required to unpack a release" >&2
    exit 1
  fi

  tmp_dir=$(mktemp -d)
  trap 'rm -rf "$tmp_dir"' EXIT HUP INT TERM

  version="${HI_SHELL_VERSION:-latest}"
  if [ "$version" = "latest" ]; then
    url="https://github.com/$repo/releases/latest/download/hi_${os}_${arch}.tar.gz"
  else
    url="https://github.com/$repo/releases/download/$version/hi_${os}_${arch}.tar.gz"
  fi

  curl -fsSL "$url" | tar -xz -C "$tmp_dir"
  cp "$tmp_dir/hi" "$bin_path"
  chmod 0755 "$bin_path"
}

if [ -f "$repo_root/go.mod" ] && [ -d "$repo_root/cmd/hi" ]; then
  build_from_source
else
  download_release
fi

"$bin_path" install zsh

echo
echo "hi-shell installed at $bin_path"
echo "Run 'exec zsh' or open a new terminal."

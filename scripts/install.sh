#!/bin/sh
set -eu

repo="longyijdos/hi-shell"
bin_dir="${HI_BIN_DIR:-"$HOME/.local/bin"}"
bin_path="$bin_dir/hi-shell"
repo_root=""

case "$0" in
  */*)
    script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
    repo_root=$(CDPATH= cd -- "$script_dir/.." && pwd)
    ;;
esac

mkdir -p "$bin_dir"

build_from_source() {
  if ! command -v go >/dev/null 2>&1; then
    echo "go is required to install from source" >&2
    exit 1
  fi

  (
    cd "$repo_root"
    go build -ldflags "-X main.version=dev" -o "$bin_path" ./cmd/hi-shell
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
  asset="hi-shell_${os}_${arch}.tar.gz"
  if [ "$version" = "latest" ]; then
    release_url="https://github.com/$repo/releases/latest/download"
  else
    release_url="https://github.com/$repo/releases/download/$version"
  fi

  archive="$tmp_dir/$asset"
  checksums="$tmp_dir/checksums.txt"

  curl -fsSL "$release_url/$asset" -o "$archive"
  curl -fsSL "$release_url/checksums.txt" -o "$checksums"
  verify_checksum "$archive" "$asset" "$checksums"

  tar -xzf "$archive" -C "$tmp_dir"
  if [ ! -f "$tmp_dir/hi-shell" ]; then
    echo "release archive did not contain hi-shell binary" >&2
    exit 1
  fi

  cp "$tmp_dir/hi-shell" "$bin_path"
  chmod 0755 "$bin_path"
}

verify_checksum() {
  archive="$1"
  asset="$2"
  checksums="$3"

  expected=$(awk -v asset="$asset" '$2 == asset { print $1 }' "$checksums")
  if [ -z "$expected" ]; then
    echo "checksum for $asset was not found" >&2
    exit 1
  fi

  if command -v sha256sum >/dev/null 2>&1; then
    actual=$(sha256sum "$archive" | awk '{ print $1 }')
  elif command -v shasum >/dev/null 2>&1; then
    actual=$(shasum -a 256 "$archive" | awk '{ print $1 }')
  else
    echo "sha256sum or shasum is required to verify downloads" >&2
    exit 1
  fi

  if [ "$actual" != "$expected" ]; then
    echo "checksum verification failed for $asset" >&2
    exit 1
  fi
}

if [ -n "$repo_root" ] && [ -f "$repo_root/go.mod" ] && [ -d "$repo_root/cmd/hi-shell" ]; then
  build_from_source
else
  download_release
fi

"$bin_path" install zsh

echo
echo "hi-shell installed at $bin_path"
echo "Run 'exec zsh' or open a new terminal."

#!/bin/sh
set -eu

if [ "$#" -lt 1 ] || [ "$#" -gt 2 ]; then
  echo "usage: $0 <version-tag> [output-file]" >&2
  exit 2
fi

version="$1"
output="${2:-dist/release-notes.md}"

if ! git rev-parse -q --verify "refs/tags/$version" >/dev/null; then
  echo "tag not found: $version" >&2
  exit 1
fi

current_commit="$(git rev-list -n 1 "$version")"
previous_tag="$(
  git tag --merged "$current_commit" --list 'v*' --sort=-v:refname |
    grep -Fvx "$version" |
    head -n 1 || true
)"

if [ -n "$previous_tag" ]; then
  range="$previous_tag..$version"
  heading="Changes since $previous_tag"
else
  range="$version"
  heading="Initial release"
fi

output_dir="$(dirname "$output")"
mkdir -p "$output_dir"

tmp="${output}.tmp"
trap 'rm -f "$tmp"' EXIT

git log --no-merges --pretty=format:'- %s (%h)' "$range" > "$tmp"

{
  printf '# %s\n\n' "$version"
  printf '%s:\n\n' "$heading"

  if [ -s "$tmp" ]; then
    cat "$tmp"
    printf '\n'
  else
    printf '%s\n' '- No changes recorded.'
  fi

  if [ -n "${GITHUB_REPOSITORY:-}" ] && [ -n "$previous_tag" ]; then
    printf '\n**Full Changelog**: https://github.com/%s/compare/%s...%s\n' \
      "$GITHUB_REPOSITORY" "$previous_tag" "$version"
  fi
} > "$output"

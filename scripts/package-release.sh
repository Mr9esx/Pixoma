#!/usr/bin/env bash
set -Eeuo pipefail

usage() {
  printf 'Usage: scripts/package-release.sh <GOOS> <GOARCH> <VERSION>\n' >&2
  exit 1
}

[[ $# -eq 3 ]] || usage

target_os="$1"
target_arch="$2"
version="$3"
release_number="${version#v}"
output_dir="release"
archive_name="pixoma_${release_number}_${target_os}_${target_arch}.tar.gz"
stage_dir="${output_dir}/stage/${target_os}-${target_arch}"
binary_extension=""

case "$target_os" in
  linux|darwin)
    ;;
  windows)
    binary_extension=".exe"
    ;;
  *)
    echo "unsupported GOOS: $target_os" >&2
    exit 1
    ;;
esac

mkdir -p "$stage_dir"
GOOS="$target_os" GOARCH="$target_arch" CGO_ENABLED=0 go build \
  -trimpath \
  -o "${stage_dir}/pixoma${binary_extension}" \
  ./apps/pixoma/cmd/pixoma
GOOS="$target_os" GOARCH="$target_arch" CGO_ENABLED=0 go build \
  -trimpath \
  -o "${stage_dir}/pixoma-edge-agent${binary_extension}" \
  ./apps/edge-agent/cmd/edge-agent

tar -czf "${output_dir}/${archive_name}" -C "$stage_dir" \
  "pixoma${binary_extension}" "pixoma-edge-agent${binary_extension}"
rm -rf "$stage_dir"

if command -v sha256sum >/dev/null 2>&1; then
  (
    cd "$output_dir"
    sha256sum "$archive_name" >> checksums.txt
  )
elif command -v shasum >/dev/null 2>&1; then
  (
    cd "$output_dir"
    shasum -a 256 "$archive_name" >> checksums.txt
  )
else
  echo "sha256sum or shasum is required" >&2
  exit 1
fi

printf 'Created %s/%s\n' "$output_dir" "$archive_name"

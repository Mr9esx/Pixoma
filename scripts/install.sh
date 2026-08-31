#!/bin/sh
set -eu

GITHUB_OWNER="Mr9esx"
GITHUB_REPO="Pixoma"

usage() {
  cat <<'EOF'
Usage: sh install.sh [--edge]

Installs the latest Pixoma release. Set PIXOMA_VERSION to pin a tag.

Examples:
  curl -fsSL https://pixoma.miaoplus.com/install.sh | sh
  curl -fsSL https://pixoma.miaoplus.com/install.sh | \
    PIXOMA_VERSION=v0.1.0 sh

Edge agents should use PIXOMA_INSTALL=edge and provide:
  CONTROL_PLANE_URL, AGENT_TOKEN, EDGE_ID

Example:
  curl -fsSL https://pixoma.miaoplus.com/install.sh | \
    PIXOMA_INSTALL=edge CONTROL_PLANE_URL=https://pixoma.example.com \
    AGENT_TOKEN=... EDGE_ID=... sh
EOF
}

fail() {
  printf 'pixoma installer: %s\n' "$1" >&2
  exit 1
}

install_binary() {
  source_file="$1"
  destination_file="$2"

  if command -v install >/dev/null 2>&1; then
    install -m 0755 "$source_file" "$destination_file"
  else
    cp "$source_file" "$destination_file"
    chmod 0755 "$destination_file"
  fi
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --edge)
      PIXOMA_INSTALL=edge
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      usage >&2
      fail "unknown option: $1"
      ;;
  esac
done

command -v curl >/dev/null 2>&1 || fail "curl is required"
command -v tar >/dev/null 2>&1 || fail "tar is required"

binary_extension=""
case "$(uname -s)" in
  Darwin)
    operating_system="darwin"
    ;;
  Linux)
    operating_system="linux"
    ;;
  MINGW*|MSYS*|CYGWIN*)
    operating_system="windows"
    binary_extension=".exe"
    ;;
  *)
    fail "supported systems are macOS, Linux, and Windows through Git Bash"
    ;;
esac

if [ "$operating_system" = "windows" ]; then
  detected_architecture="${PROCESSOR_ARCHITEW6432:-${PROCESSOR_ARCHITECTURE:-$(uname -m)}}"
else
  detected_architecture="$(uname -m)"
fi

case "$detected_architecture" in
  x86_64|amd64|AMD64) architecture="amd64" ;;
  arm64|aarch64|ARM64) architecture="arm64" ;;
  *) fail "unsupported architecture: $detected_architecture" ;;
esac

PIXOMA_BINARY="pixoma${binary_extension}"
EDGE_BINARY="pixoma-edge-agent${binary_extension}"

version="${PIXOMA_VERSION:-}"
if [ -z "$version" ]; then
  api_url="https://api.github.com/repos/${GITHUB_OWNER}/${GITHUB_REPO}/releases/latest"
  release_json="$(curl -fsSL "$api_url")" || fail "unable to resolve the latest release"
  version="$(printf '%s\n' "$release_json" | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' | head -n 1)"
  [ -n "$version" ] || fail "the release metadata did not contain a tag name"
fi

release_number="${version#v}"
archive_name="pixoma_${release_number}_${operating_system}_${architecture}.tar.gz"
download_url="https://github.com/${GITHUB_OWNER}/${GITHUB_REPO}/releases/download/${version}/${archive_name}"

temporary_dir="$(mktemp -d "${TMPDIR:-${TEMP:-${TMP:-/tmp}}}/pixoma-install.XXXXXX")"
trap 'rm -rf "$temporary_dir"' EXIT INT TERM

printf 'Downloading Pixoma %s for %s/%s...\n' "$version" "$operating_system" "$architecture"
curl -fL --progress-bar "$download_url" -o "$temporary_dir/$archive_name" || fail "unable to download $download_url"

checksum_url="https://github.com/${GITHUB_OWNER}/${GITHUB_REPO}/releases/download/${version}/checksums.txt"
checksum_file="$temporary_dir/checksums.txt"
curl -fsSL "$checksum_url" -o "$checksum_file" || fail "unable to download $checksum_url"
expected_checksum="$(awk -v file="$archive_name" '$2 == file {print $1}' "$checksum_file" | head -n 1)"
[ -n "$expected_checksum" ] || fail "checksum file does not contain $archive_name"

if command -v sha256sum >/dev/null 2>&1; then
  actual_checksum="$(sha256sum "$temporary_dir/$archive_name" | awk '{print $1}')"
elif command -v shasum >/dev/null 2>&1; then
  actual_checksum="$(shasum -a 256 "$temporary_dir/$archive_name" | awk '{print $1}')"
elif [ "$operating_system" = "windows" ] && command -v certutil >/dev/null 2>&1; then
  windows_path="$(cygpath -w "$temporary_dir/$archive_name" 2>/dev/null || printf '%s' "$temporary_dir/$archive_name")"
  actual_checksum="$(certutil -hashfile "$windows_path" SHA256 | sed -n '2s/^[[:space:]]*//p')"
else
  fail "sha256sum, shasum, or certutil is required to verify downloads"
fi

[ "$actual_checksum" = "$expected_checksum" ] || fail "checksum verification failed for $archive_name"

extract_dir="$temporary_dir/extract"
mkdir -p "$extract_dir"
tar -xzf "$temporary_dir/$archive_name" -C "$extract_dir"

if [ "${PIXOMA_INSTALL:-control-plane}" = "edge" ]; then
  : "${CONTROL_PLANE_URL:?CONTROL_PLANE_URL is required for edge installation}"
  : "${AGENT_TOKEN:?AGENT_TOKEN is required for edge installation}"
  : "${EDGE_ID:?EDGE_ID is required for edge installation}"

  [ -f "$extract_dir/$EDGE_BINARY" ] || fail "$EDGE_BINARY is missing from the release archive"
else
  [ -f "$extract_dir/$PIXOMA_BINARY" ] || fail "$PIXOMA_BINARY is missing from the release archive"
  [ -f "$extract_dir/$EDGE_BINARY" ] || fail "$EDGE_BINARY is missing from the release archive"
fi

if [ -n "${PIXOMA_INSTALL_DIR:-}" ]; then
  install_dir="$PIXOMA_INSTALL_DIR"
elif [ "$operating_system" = "windows" ]; then
  windows_home="${HOME:-${USERPROFILE:-}}"
  [ -n "$windows_home" ] || fail "unable to determine the Windows home directory"
  install_dir="$windows_home/.pixoma/bin"
elif [ "$(id -u)" -eq 0 ]; then
  install_dir="/usr/local/bin"
elif [ -w /usr/local/bin ]; then
  install_dir="/usr/local/bin"
else
  install_dir="${HOME}/.pixoma/bin"
fi

mkdir -p "$install_dir"
install_binary "$extract_dir/$EDGE_BINARY" "$install_dir/$EDGE_BINARY"
if [ "${PIXOMA_INSTALL:-control-plane}" != "edge" ]; then
  install_binary "$extract_dir/$PIXOMA_BINARY" "$install_dir/$PIXOMA_BINARY"
fi

case ":$PATH:" in
  *":$install_dir:"*) ;;
  *) printf '\nNote: add this directory to PATH if needed:\n  export PATH="%s:$PATH"\n' "$install_dir" ;;
esac

if [ "${PIXOMA_INSTALL:-control-plane}" = "edge" ]; then
  printf '\nPixoma edge agent installed: %s/%s\n' "$install_dir" "$EDGE_BINARY"
  printf 'Run it with:\n  CONTROL_PLANE_URL=%s AGENT_TOKEN=*** EDGE_ID=%s %s/%s\n' \
    "$CONTROL_PLANE_URL" "$EDGE_ID" "$install_dir" "$EDGE_BINARY"
  printf 'Use systemd or launchd to run it as a persistent service.\n'
else
  printf '\nPixoma control plane installed: %s/%s\n' "$install_dir" "$PIXOMA_BINARY"
  printf 'Start it with:\n  %s/%s\n' "$install_dir" "$PIXOMA_BINARY"
  printf 'Then open the admin URL printed by pixoma and complete the setup wizard.\n'
fi

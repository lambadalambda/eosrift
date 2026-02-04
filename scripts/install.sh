#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Install the eosrift client from GitHub Releases.

Usage:
  ./scripts/install.sh [--version <tag>] [--repo <owner/repo>] [--dir <install-dir>]

Examples:
  ./scripts/install.sh --version v0.1.1
  ./scripts/install.sh --repo your-org/eosrift --version v0.1.1 --dir ~/.local/bin

Notes:
  - If --version is omitted, this script installs the latest GitHub release.
  - Release artifacts must match the naming produced by .github/workflows/release.yml.
EOF
}

repo="${EOSRIFT_INSTALL_REPO:-lambadalambda/eosrift}"
version="${EOSRIFT_INSTALL_VERSION:-}"
install_dir="${EOSRIFT_INSTALL_DIR:-$HOME/.local/bin}"

while [ $# -gt 0 ]; do
  case "$1" in
    --repo)
      repo="$2"
      shift 2
      ;;
    --version)
      version="$2"
      shift 2
      ;;
    --dir|--install-dir|--prefix)
      install_dir="$2"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "unknown argument: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

os="$(uname -s)"
case "$os" in
  Linux) os="linux" ;;
  Darwin) os="darwin" ;;
  *)
    echo "unsupported OS: ${os}" >&2
    exit 1
    ;;
esac

arch="$(uname -m)"
case "$arch" in
  x86_64) arch="amd64" ;;
  aarch64|arm64) arch="arm64" ;;
  *)
    echo "unsupported arch: ${arch}" >&2
    exit 1
    ;;
esac

sha256_file() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
    return
  fi
  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$1" | awk '{print $1}'
    return
  fi
  if command -v openssl >/dev/null 2>&1; then
    openssl dgst -sha256 "$1" | awk '{print $NF}'
    return
  fi
  echo "no sha256 tool found (need sha256sum, shasum, or openssl)" >&2
  exit 1
}

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "missing required command: $1" >&2
    exit 1
  fi
}

require_cmd curl
require_cmd tar
require_cmd awk

if [ -z "$version" ]; then
  version="$(
    curl -fsSL "https://api.github.com/repos/${repo}/releases/latest" \
      | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' \
      | head -n 1
  )"
  if [ -z "$version" ]; then
    echo "failed to determine latest version for ${repo}" >&2
    exit 1
  fi
fi

tmp="$(mktemp -d)"
cleanup() { rm -rf "${tmp}"; }
trap cleanup EXIT

tar_name="eosrift_${version}_${os}_${arch}.tar.gz"
base_url="https://github.com/${repo}/releases/download/${version}"

echo "repo:    ${repo}"
echo "version: ${version}"
echo "target:  ${os}/${arch}"
echo "install: ${install_dir}/eosrift"

curl -fsSL "${base_url}/checksums.txt" -o "${tmp}/checksums.txt"
curl -fsSL "${base_url}/${tar_name}" -o "${tmp}/${tar_name}"

expected="$(
  grep "  ${tar_name}\$" "${tmp}/checksums.txt" \
    | awk '{print $1}' \
    | head -n 1
)"
if [ -z "$expected" ]; then
  echo "no checksum entry for ${tar_name} in checksums.txt" >&2
  exit 1
fi

actual="$(sha256_file "${tmp}/${tar_name}")"
if [ "$expected" != "$actual" ]; then
  echo "checksum mismatch for ${tar_name}" >&2
  echo "expected: ${expected}" >&2
  echo "actual:   ${actual}" >&2
  exit 1
fi

(cd "${tmp}" && tar -xzf "${tar_name}")

dir="eosrift_${version}_${os}_${arch}"
bin="${tmp}/${dir}/eosrift"
if [ ! -f "${bin}" ]; then
  echo "extracted binary not found at ${bin}" >&2
  exit 1
fi

mkdir -p "${install_dir}"
cp "${bin}" "${install_dir}/eosrift"
chmod +x "${install_dir}/eosrift"

echo "installed: ${install_dir}/eosrift"

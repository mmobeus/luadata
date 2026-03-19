#!/usr/bin/env bash
#
# Prepares the release branch by staging shared libraries and generating
# platform-specific Go embed files.
#
# Usage:
#   scripts/prepare-release.sh <release-tag> <artifacts-dir>
#
# Example (CI):
#   scripts/prepare-release.sh v0.5.0 clib-artifacts
#
# Example (local dry-run, skips git):
#   DRY_RUN=1 scripts/prepare-release.sh v0.5.0 clib-artifacts
#
# The artifacts directory should contain subdirectories like:
#   clib-linux_amd64/libluadata_clib.so
#   clib-darwin_arm64/libluadata_clib.dylib
#   etc.

set -euo pipefail

RELEASE_TAG="${1:?Usage: prepare-release.sh <release-tag> <artifacts-dir>}"
ARTIFACTS_DIR="${2:?Usage: prepare-release.sh <release-tag> <artifacts-dir>}"
DRY_RUN="${DRY_RUN:-}"

FFI_DIR="go/internal/ffi"
LIB_DIR="$FFI_DIR/lib"

# ── Set workspace version from tag ───────────────────────────────

VERSION="${RELEASE_TAG#v}"
echo "Setting workspace version to ${VERSION}..."
sed -i.bak "s/^version = \".*\"/version = \"${VERSION}\"/" Cargo.toml
rm -f Cargo.toml.bak
echo "  updated: Cargo.toml"

# ── Stage shared libraries ────────────────────────────────────────

stage_lib() {
  local platform="$1" src_name="$2" dst_name="$3"
  local src="$ARTIFACTS_DIR/clib-${platform}/${src_name}"
  local dst="$LIB_DIR/${platform}/${dst_name}"

  if [[ ! -f "$src" ]]; then
    echo "ERROR: missing artifact: $src" >&2
    exit 1
  fi

  mkdir -p "$LIB_DIR/${platform}"
  cp "$src" "$dst"
  echo "  staged: $dst"
}

echo "Staging shared libraries..."
stage_lib darwin_amd64  libluadata_clib.dylib  libluadata.dylib
stage_lib darwin_arm64  libluadata_clib.dylib  libluadata.dylib
stage_lib linux_amd64   libluadata_clib.so     libluadata.so
stage_lib linux_arm64   libluadata_clib.so     libluadata.so
stage_lib windows_amd64 luadata_clib.dll       luadata.dll

# ── Generate platform embed files ─────────────────────────────────

write_embed() {
  local file="$1" constraint="$2" embed_path="$3" lib_name="$4"
  cat > "$file" <<EOF
//go:build ${constraint}

package ffi

import _ "embed"

//go:embed ${embed_path}
var EmbeddedLib []byte

const LibName = "${lib_name}"
EOF
  echo "  wrote: $file"
}

echo "Replacing embed_dev.go with platform embed files..."
rm -f "$FFI_DIR/embed_dev.go"

write_embed "$FFI_DIR/embed_darwin_amd64.go"  "darwin && amd64"  "lib/darwin_amd64/libluadata.dylib"  "libluadata.dylib"
write_embed "$FFI_DIR/embed_darwin_arm64.go"  "darwin && arm64"  "lib/darwin_arm64/libluadata.dylib"  "libluadata.dylib"
write_embed "$FFI_DIR/embed_linux_amd64.go"   "linux && amd64"   "lib/linux_amd64/libluadata.so"      "libluadata.so"
write_embed "$FFI_DIR/embed_linux_arm64.go"   "linux && arm64"   "lib/linux_arm64/libluadata.so"      "libluadata.so"
write_embed "$FFI_DIR/embed_windows_amd64.go" "windows && amd64" "lib/windows_amd64/luadata.dll"      "luadata.dll"

# ── Update .gitignore ─────────────────────────────────────────────

echo "Updating go/.gitignore to allow lib/ on release branch..."
sed -i.bak '/internal\/ffi\/lib\//d' go/.gitignore
rm -f go/.gitignore.bak

# ── Git operations ────────────────────────────────────────────────

if [[ -n "$DRY_RUN" ]]; then
  echo ""
  echo "DRY_RUN: skipping git operations. Files staged at:"
  echo "  $FFI_DIR/embed_*.go"
  echo "  $LIB_DIR/*/"
  exit 0
fi

echo "Creating release branch commit..."
git config user.name "github-actions[bot]"
git config user.email "github-actions[bot]@users.noreply.github.com"

git checkout -B release
git add Cargo.toml
git add "$FFI_DIR/"
git add go/.gitignore
git commit -m "Release ${RELEASE_TAG}: embed shared libraries and set version"
git tag "$RELEASE_TAG"
git push origin release --force
git push origin "$RELEASE_TAG"

echo "Done. Tagged ${RELEASE_TAG} on release branch."

#!/usr/bin/env bash
#
# Smoke-test a published luadata release across all package targets.
#
# Usage:
#   bash scripts/smoke-test-release.sh [version]
#
# If no version is given, reads from Cargo.toml.

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

VERSION="${1:-$(grep '^version = ' "$REPO_ROOT/Cargo.toml" | head -1 | sed 's/.*"\(.*\)"/\1/')}"

INPUT='playerName = "Thrall"'
EXPECTED_KEY='"playerName"'
EXPECTED_VAL='"Thrall"'

PASS=0
FAIL=0
SKIP=0
RESULTS=()

TMPDIRS=()
cleanup() { for d in "${TMPDIRS[@]+"${TMPDIRS[@]}"}"; do rm -rf "$d"; done; }
trap cleanup EXIT

make_tmpdir() {
  local d
  d="$(mktemp -d)"
  TMPDIRS+=("$d")
  echo "$d"
}

pass() { RESULTS+=("  PASS  $1"); ((PASS++)); }
fail() { RESULTS+=("  FAIL  $1 -- $2"); ((FAIL++)); }
skip() { RESULTS+=("  SKIP  $1 -- $2"); ((SKIP++)); }

check_output() {
  local name="$1" output="$2"
  if echo "$output" | grep -q "$EXPECTED_KEY" && echo "$output" | grep -q "$EXPECTED_VAL"; then
    pass "$name"
  else
    fail "$name" "unexpected output: $output"
  fi
}

has_cmd() { command -v "$1" &>/dev/null; }

# ── Rust (crates.io) ────────────────────────────────────────────

test_rust() {
  local name="rust (crates.io)"
  if ! has_cmd cargo; then skip "$name" "cargo not found"; return; fi

  local dir
  dir="$(make_tmpdir)"
  echo "  Testing $name..."

  cargo init --name smoke_rust "$dir" --quiet 2>/dev/null

  sed -i.bak "s/\[dependencies\]/[dependencies]\nluadata = \"=$VERSION\"/" "$dir/Cargo.toml"
  rm -f "$dir/Cargo.toml.bak"

  cat > "$dir/src/main.rs" <<'RUST'
use luadata::{text_to_json, ParseConfig};
fn main() {
    let json = text_to_json("input", r#"playerName = "Thrall""#, ParseConfig::new()).unwrap();
    print!("{}", json);
}
RUST

  local output
  if output="$(cargo run --manifest-path "$dir/Cargo.toml" --quiet 2>&1)"; then
    check_output "$name" "$output"
  else
    fail "$name" "cargo run failed: $output"
  fi
}

# ── Python (PyPI) ───────────────────────────────────────────────

test_python() {
  local name="python (pypi)"
  if ! has_cmd python3; then skip "$name" "python3 not found"; return; fi

  local dir
  dir="$(make_tmpdir)"
  echo "  Testing $name..."

  python3 -m venv "$dir/venv"
  "$dir/venv/bin/pip" install "mmobeus-luadata==$VERSION" --quiet 2>/dev/null

  local output
  if output="$("$dir/venv/bin/python" -c "
from luadata import lua_to_json
print(lua_to_json('playerName = \"Thrall\"'))
" 2>&1)"; then
    check_output "$name" "$output"
  else
    fail "$name" "python run failed: $output"
  fi
}

# ── npm (WebAssembly) ───────────────────────────────────────────

test_npm() {
  local name="npm"
  if ! has_cmd npm; then skip "$name" "npm not found"; return; fi

  local dir
  dir="$(make_tmpdir)"
  echo "  Testing $name..."

  cat > "$dir/package.json" <<EOF
{"dependencies": {"@mmobeus/luadata-wasm": "$VERSION"}}
EOF

  local output
  if ! output="$(npm install --prefix "$dir" 2>&1)"; then
    fail "$name" "npm install failed: $output"
    return
  fi

  # Verify expected files exist (WASM package requires a bundler to run,
  # so we check structure rather than executing)
  local pkg="$dir/node_modules/@mmobeus/luadata-wasm"
  local missing=()
  for f in index.js index.d.ts wasm/luadata_wasm.js wasm/luadata_wasm_bg.wasm; do
    [[ -f "$pkg/$f" ]] || missing+=("$f")
  done

  if [[ ${#missing[@]} -eq 0 ]]; then
    pass "$name"
  else
    fail "$name" "missing files: ${missing[*]}"
  fi
}

# ── Go ──────────────────────────────────────────────────────────

test_go() {
  local name="go"
  if ! has_cmd go; then skip "$name" "go not found"; return; fi

  local dir
  dir="$(make_tmpdir)"
  echo "  Testing $name..."

  # Go resolves subdirectory modules via go/v0.1.x git tags automatically
  local get_output
  if ! get_output="$(cd "$dir" && go mod init smoketest 2>&1 && go get "github.com/mmobeus/luadata/go@v$VERSION" 2>&1)"; then
    fail "$name" "go get failed: $get_output"
    return
  fi

  cat > "$dir/main.go" <<'GO'
package main

import (
	"fmt"
	"io"
	luadata "github.com/mmobeus/luadata/go"
)

func main() {
	reader, err := luadata.TextToJSON("input", `playerName = "Thrall"`)
	if err != nil {
		panic(err)
	}
	b, _ := io.ReadAll(reader)
	fmt.Print(string(b))
}
GO

  local output
  if output="$(cd "$dir" && go run . 2>&1)"; then
    check_output "$name" "$output"
  else
    fail "$name" "go run failed: $output"
  fi
}

# ── npm native (napi-rs) ────────────────────────────────────────

test_napi() {
  local name="npm-native (napi)"
  if ! has_cmd node || ! has_cmd npm; then skip "$name" "node/npm not found"; return; fi

  local dir
  dir="$(make_tmpdir)"
  echo "  Testing $name..."

  cat > "$dir/package.json" <<EOF
{"dependencies": {"@mmobeus/luadata": "$VERSION"}}
EOF

  local output
  if ! output="$(npm install --prefix "$dir" 2>&1)"; then
    fail "$name" "npm install failed: $output"
    return
  fi

  cat > "$dir/test.mjs" <<'JS'
import { convertLuaToJson } from "@mmobeus/luadata";
console.log(convertLuaToJson('playerName = "Thrall"'));
JS

  if output="$(node "$dir/test.mjs" 2>&1)"; then
    check_output "$name" "$output"
  else
    fail "$name" "node run failed: $output"
  fi
}

# ── Homebrew (CLI) ──────────────────────────────────────────────

test_homebrew() {
  local name="homebrew (cli)"
  if ! has_cmd brew; then skip "$name" "brew not found"; return; fi

  echo "  Testing $name..."

  # Install or upgrade
  if brew list mmobeus/tap/luadata &>/dev/null; then
    brew upgrade mmobeus/tap/luadata 2>/dev/null || true
  else
    brew install mmobeus/tap/luadata 2>/dev/null
  fi

  if ! has_cmd luadata; then
    fail "$name" "luadata binary not found after install"
    return
  fi

  local output
  if output="$(echo "$INPUT" | luadata tojson - 2>&1)"; then
    check_output "$name" "$output"
  else
    fail "$name" "luadata tojson failed: $output"
  fi
}

# ── Run ─────────────────────────────────────────────────────────

echo "==========================================="
echo "  Smoke Test: luadata v$VERSION"
echo "==========================================="
echo ""

test_rust
test_python
test_npm
test_napi
test_go
test_homebrew

echo ""
echo "==========================================="
echo "  Results: luadata v$VERSION"
echo "==========================================="
for r in "${RESULTS[@]}"; do echo "$r"; done
echo "==========================================="
echo "  $PASS passed, $FAIL failed, $SKIP skipped"
echo "==========================================="

if (( FAIL > 0 )); then exit 1; fi

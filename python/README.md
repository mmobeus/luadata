# Python Package: mmobeus-luadata

This document explains how the Python package works, how it calls into the Go
library, how the wheels are built, and the decisions behind the design.

## Architecture overview

The core Lua-to-JSON conversion logic is written in Go (the `luadata` module at
the repo root). The Python package doesn't reimplement any parsing — it calls
the Go code directly through a shared library (`.so` on Linux, `.dylib` on
macOS, `.dll` on Windows).

The chain looks like this:

```
Python (luadata/__init__.py)
  → ctypes FFI (luadata/_binding.py)
    → Go shared library (libluadata.so/.dylib/.dll)
      → Go luadata package (the real parser)
```

## The C shared library adapter (`clib/`)

Go can compile packages as C-compatible shared libraries using
`-buildmode=c-shared`. The `clib/` directory contains a thin Go adapter that
bridges between C calling conventions and the Go `luadata` package.

### Why it must be `package main`

Go's `c-shared` build mode requires:

1. The package must be declared as `package main`
2. A `func main() {}` must exist (even though it's empty and never called)
3. Exported functions use `//export` directives and accept/return C types

This is a hard requirement of the Go toolchain, not a design choice.

### Why it's not Python-specific

The shared library exposes a generic C interface — two functions:

- `LuaDataToJSON(input *C.char, options *C.char) *C.char` — takes Lua text and
  JSON-encoded options, returns a JSON response envelope
- `LuaDataFree(p *C.char)` — frees memory allocated by the Go side

Any language that can call C functions (Python, Ruby, Node via FFI, etc.) can
use this library. The `clib/` directory lives outside `python/` for this reason.

### The response envelope

Rather than dealing with C error handling conventions, the adapter wraps all
responses in a JSON envelope:

```json
{"result": "..."}   // on success
{"error": "..."}    // on failure
```

The Python side parses this envelope and raises `LuaDataError` for error
responses.

### CGO and cross-compilation

Because the Go code uses `import "C"` (cgo), it requires a C compiler on the
build machine. Go's usual easy cross-compilation (`GOOS=linux GOARCH=amd64`)
does not work for `c-shared` builds — you need an actual C toolchain for each
target platform. This is why we build on native runners in CI rather than
cross-compiling from a single machine.

## The Python binding (`luadata/_binding.py`)

The binding uses Python's built-in `ctypes` module (no external dependencies
like cffi or Cython). At import time, it:

1. Finds the shared library (see search order below)
2. Loads it with `ctypes.CDLL()`
3. Declares the function signatures (`argtypes` and `restype`)

### Library search order

1. `LUADATA_LIB_PATH` environment variable — explicit path override
2. Next to `_binding.py` itself — this is where the wheel bundles it
3. `bin/clib/` relative to the project root — for local development after
   `make build-clib`

### Memory management

The Go library allocates C strings (via `C.CString()`) for return values. The
Python side is responsible for freeing them. `_binding.py` does this in a
`try/finally` block:

```python
ptr = _lib.LuaDataToJSON(input, options)
try:
    raw = ctypes.cast(ptr, ctypes.c_char_p).value
    return raw.decode("utf-8")
finally:
    _lib.LuaDataFree(ptr)
```

### Async compatibility

The ctypes call releases the GIL (this is the default behavior for
`ctypes.CDLL` foreign function calls), so other Python threads and async tasks
can make progress while Go is parsing. The Go library performs no I/O — it's
purely CPU-bound in-memory parsing.

For async code, the simplest approach is:

```python
result = await asyncio.to_thread(lua_to_json, "x = 42")
```

File I/O (when passing a `Path`) happens on the Python side before the C call,
so `asyncio.to_thread` covers both the file read and the parsing.

## Wheel building and distribution

### Why platform-specific wheels?

A normal Python package produces a "pure" wheel tagged `py3-none-any` (works
everywhere). Since our package bundles a compiled shared library, each wheel
only works on one OS + architecture combination. The wheel's platform tag tells
pip which one to download.

### Platform tags explained

| Tag | Meaning |
|-----|---------|
| `manylinux_2_17_x86_64` | Linux, glibc >= 2.17, Intel 64-bit |
| `manylinux_2_17_aarch64` | Linux, glibc >= 2.17, ARM 64-bit |
| `macosx_11_0_x86_64` | macOS >= 11 (Big Sur), Intel |
| `macosx_11_0_arm64` | macOS >= 11 (Big Sur), Apple Silicon |
| `win_amd64` | Windows, 64-bit |

**manylinux** is not a Linux distro — it's a compatibility standard. The
`2_17` means "requires glibc 2.17 or newer," which covers RHEL 7+, Ubuntu
14.04+, Debian 8+, and basically every Linux distribution from 2013 onward. Go
shared libraries are statically linked (the entire Go runtime is compiled in),
so the only external dependency is libc itself.

**macOS** tags include a minimum OS version. We set `MACOSX_DEPLOYMENT_TARGET=11.0`
during the Go build, which tells the compiler to produce a binary compatible
back to macOS 11. Without this, the binary would target the build machine's
OS version and pip would reject it on older macOS.

**Windows** tags don't include a version — all modern 64-bit Windows is treated
as compatible.

### The retagging approach

Since we use ctypes (not a CPython C extension), setuptools has no way to know
the wheel contains native code. It produces a `py3-none-any` wheel. We fix this
by retagging after the build:

```bash
python -m build --wheel
python -m wheel tags --remove --platform-tag manylinux_2_17_x86_64 dist/*.whl
```

This is cleaner than the alternative of faking a C extension module in
`setup.py` just to trick setuptools into adding platform tags.

### No auditwheel or delocate needed

Tools like `auditwheel` (Linux) and `delocate` (macOS) exist to bundle
third-party shared library dependencies (like libssl) into a wheel and fix
up runtime library paths. We don't need them because Go shared libraries are
self-contained — the only external dependency is libc/libpthread, which is
always present on the system.

### No sdist published

We only publish wheels, not a source distribution (sdist). An sdist would
require the user to have Go installed to build the shared library from source,
which would be a confusing installation experience. If someone needs to build
from source, they should clone the repo and use `make build-clib`.

## CI workflow structure

The wheel build workflow (`.github/workflows/python-wheels.yml`) has three jobs:

### Job 1: `build-clib` (matrix, native runners)

Builds the Go shared library on 5 platform runners. This is the **only** job
that runs on expensive runners (macOS, Windows). It does the absolute minimum:
checkout, setup Go, compile, upload artifact. The Go build takes ~5 seconds.

| Target | Runner |
|--------|--------|
| linux-x64 | `ubuntu-latest` |
| linux-arm64 | `ubuntu-24.04-arm` |
| macos-x64 | `macos-15-intel` |
| macos-arm64 | `macos-latest` |
| windows-x64 | `windows-latest` |

Windows needs MinGW installed (`choco install mingw`) because Go's cgo on
Windows defaults to gcc, which isn't included on the GitHub runner.

`macos-13` (the old Intel runner) was retired in December 2025. `macos-15-intel`
is the current Intel macOS runner. GitHub plans to remove all Intel macOS
runners by August 2027.

### Job 2: `build-wheels` (single Linux runner)

Runs on `ubuntu-latest` only. Downloads all 5 shared library artifacts and
builds 5 platform-tagged wheels in a loop. All the Python tooling (`pip`,
`build`, `wheel tags`) runs here on the cheapest runner. This keeps expensive
runner time to the bare minimum.

### Job 3: `publish`

Only runs on tag pushes (not `workflow_dispatch`). Downloads the wheels,
publishes to PyPI via OIDC trusted publishing, and attaches the `.whl` files to
the GitHub Release.

## Versioning

The package version is derived from git tags at build time using `setuptools-scm`.
There is no hardcoded version in `pyproject.toml`. When CI builds wheels from a
`v1.2.3` tag, `setuptools-scm` reads that tag and sets the package version to
`1.2.3`.

This means the release process (`scripts/release.sh`) doesn't need to touch
any Python files — it just creates a git tag and pushes it, which triggers the
wheel build workflow.

The `setuptools-scm` config uses `root = ".."` because `pyproject.toml` lives
in `python/` but the git tags are on the repository root.

## Package naming

The PyPI distribution name is `mmobeus-luadata` (the name `luadata` was already
taken). The Python import name remains `luadata`:

```bash
pip install mmobeus-luadata
```

```python
from luadata import lua_to_json, lua_to_dict
```

PyPI normalizes hyphens and underscores in package names, so `mmobeus-luadata`
and `mmobeus_luadata` are treated as the same package.

## Local development

For local development, you don't need wheels. Just build the shared library
and run the Python code directly:

```bash
make build-clib          # builds bin/clib/libluadata.{so,dylib,dll}
cd python
pip install -e ".[dev]"  # or just: python -m pytest tests -v
```

The binding's library search (path #3) finds `bin/clib/libluadata.*` relative
to the project root automatically.

"""Low-level ctypes binding to libluadata shared library."""

import ctypes
import json
import os
import platform


def _find_library() -> str:
    """Locate libluadata shared library.

    Search order:
    1. LUADATA_LIB_PATH environment variable (exact path to the .so/.dylib/.dll)
    2. Next to this Python file (for installed wheels with bundled lib)
    3. Project bin/clib/ directory (for local development)
    """
    ext = {
        "Darwin": ".dylib",
        "Linux": ".so",
        "Windows": ".dll",
    }.get(platform.system(), ".so")

    lib_name = f"libluadata{ext}"

    # 1. Explicit env var
    env_path = os.environ.get("LUADATA_LIB_PATH")
    if env_path and os.path.isfile(env_path):
        return env_path

    # 2. Next to this file (bundled in wheel)
    pkg_dir = os.path.dirname(os.path.abspath(__file__))
    bundled = os.path.join(pkg_dir, lib_name)
    if os.path.isfile(bundled):
        return bundled

    # 3. Project bin/clib/ (local dev)
    project_root = os.path.dirname(os.path.dirname(pkg_dir))
    dev_path = os.path.join(project_root, "bin", "clib", lib_name)
    if os.path.isfile(dev_path):
        return dev_path

    raise FileNotFoundError(
        f"Could not find {lib_name}. Either:\n"
        f"  - Set LUADATA_LIB_PATH to the full path of the shared library\n"
        f"  - Run 'make build-clib' from the project root\n"
        f"  - Install the luadata wheel (which bundles the library)"
    )


def _load_library():
    lib_path = _find_library()
    lib = ctypes.CDLL(lib_path)

    lib.LuaDataToJSON.argtypes = [ctypes.c_char_p, ctypes.c_char_p]
    lib.LuaDataToJSON.restype = ctypes.c_void_p

    lib.LuaDataFree.argtypes = [ctypes.c_void_p]
    lib.LuaDataFree.restype = None

    return lib


_lib = _load_library()


def call_lua_data_to_json(input_text: str, options_json: str) -> str:
    """Call the Go LuaDataToJSON function and return the raw JSON response.

    Returns the JSON response envelope string: {"result": "..."} or {"error": "..."}.
    """
    ptr = _lib.LuaDataToJSON(
        input_text.encode("utf-8"),
        options_json.encode("utf-8"),
    )
    try:
        raw = ctypes.cast(ptr, ctypes.c_char_p).value
        return raw.decode("utf-8")
    finally:
        _lib.LuaDataFree(ptr)

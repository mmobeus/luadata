#!/usr/bin/env bash
set -uo pipefail

expect_fail=false

while [[ $# -gt 0 ]]; do
    case "$1" in
        --expect-fail)
            expect_fail=true
            shift
            ;;
        -*)
            echo "Unknown option: $1" >&2
            echo "Usage: validate-folder.sh [--expect-fail] <directory>" >&2
            exit 1
            ;;
        *)
            break
            ;;
    esac
done

if [[ $# -lt 1 ]]; then
    echo "Usage: validate-folder.sh [--expect-fail] <directory>" >&2
    exit 1
fi

dir="$1"
if [[ ! -d "$dir" ]]; then
    echo "Error: $dir is not a directory" >&2
    exit 1
fi

pass=0
fail=0

while IFS= read -r -d '' file; do
    if luadata validate "$file" 2>/dev/null; then
        if [[ "$expect_fail" == true ]]; then
            echo "UNEXPECTED PASS: $file" >&2
            ((fail++))
        else
            ((pass++))
        fi
    else
        if [[ "$expect_fail" == true ]]; then
            ((pass++))
        else
            echo "FAIL: $file" >&2
            ((fail++))
        fi
    fi
done < <(find "$dir" -type f \( -name '*.lua' -o -name '*.lua.bak' \) -print0)

total=$((pass + fail))

if [[ "$expect_fail" == true ]]; then
    echo "${pass}/${total} files correctly failed validation, ${fail} unexpected passes."
else
    echo "${pass}/${total} files validated successfully, ${fail} failed."
fi

if [[ $fail -gt 0 ]]; then
    exit 1
fi

#!/usr/bin/env bash
set -euo pipefail

BUMP="${1:-patch}"

# Validate working tree is clean
if [ -n "$(git status --porcelain)" ]; then
	echo "Error: working tree is not clean. Commit or stash changes first." >&2
	exit 1
fi

# Validate we're on main
BRANCH="$(git rev-parse --abbrev-ref HEAD)"
if [ "$BRANCH" != "main" ]; then
	echo "Error: must be on the main branch (currently on '$BRANCH')." >&2
	exit 1
fi

# Find the latest semver tag
LATEST="$(git tag --sort=-v:refname | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$' | head -1 || true)"

if [ -z "$LATEST" ]; then
	MAJOR=0
	MINOR=0
	PATCH=0
	CURRENT="none"
else
	CURRENT="$LATEST"
	VERSION="${LATEST#v}"
	MAJOR="${VERSION%%.*}"
	REST="${VERSION#*.}"
	MINOR="${REST%%.*}"
	PATCH="${REST#*.}"
fi

# Compute next version
case "$BUMP" in
patch)
	PATCH=$((PATCH + 1))
	NEXT="v${MAJOR}.${MINOR}.${PATCH}"
	;;
minor)
	MINOR=$((MINOR + 1))
	NEXT="v${MAJOR}.${MINOR}.0"
	;;
major)
	MAJOR=$((MAJOR + 1))
	NEXT="v${MAJOR}.0.0"
	;;
manual)
	printf "Enter version (e.g. v1.0.0-rc1): "
	read -r NEXT
	if [[ ! "$NEXT" =~ ^v ]]; then
		echo "Error: version must start with 'v'." >&2
		exit 1
	fi
	;;
*)
	echo "Error: unknown bump type '$BUMP'. Use patch, minor, major, or manual." >&2
	exit 1
	;;
esac

# Confirm
printf "Release %s? (current: %s) [y/N] " "$NEXT" "$CURRENT"
read -r CONFIRM
if [[ "$CONFIRM" != "y" && "$CONFIRM" != "Y" ]]; then
	echo "Aborted."
	exit 0
fi

# Tag and push
git tag "$NEXT"
git push origin "$NEXT"

# Create GitHub release
gh release create "$NEXT" --generate-notes

echo "Released $NEXT"

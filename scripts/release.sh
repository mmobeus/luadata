#!/usr/bin/env bash
#
# Initiate a release by tagging an RC on main.
#
# Usage:
#   scripts/release.sh [patch|minor|major|manual]
#   make release              # defaults to patch
#   make release BUMP=minor
#
# What happens:
#   1. Validates clean working tree on main
#   2. Computes next version from existing semver tags
#   3. Tags main with v<version>-rc.1 (or increments rc.N if retrying)
#   4. Pushes the tag to origin
#
# CI (release.yml) then:
#   - Cross-compiles the Rust cdylib for all platforms
#   - Runs Rust + Go tests
#   - Creates a release branch commit with embedded shared libs
#   - Tags the final v<version>
#   - Creates a GitHub Release with downloadable artifacts
#
# If CI fails, fix the issue on main and run this script again —
# it will increment the RC number (v0.5.0-rc.2, etc.).
#
set -euo pipefail

BUMP="${1:-patch}"

# ── Validate ──────────────────────────────────────────────────────

if [ -n "$(git status --porcelain)" ]; then
	echo "Error: working tree is not clean. Commit or stash changes first." >&2
	exit 1
fi

BRANCH="$(git rev-parse --abbrev-ref HEAD)"
if [ "$BRANCH" != "main" ]; then
	echo "Error: must be on the main branch (currently on '$BRANCH')." >&2
	exit 1
fi

# Make sure we have the latest tags
git fetch origin --tags --quiet

# ── Compute next version ──────────────────────────────────────────

# Find latest final release tag (v1.2.3, not rc tags)
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

# Check for an in-flight RC (an RC tag exists but no matching final release)
LATEST_ANY_RC="$(git tag --sort=-v:refname | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+-rc\.[0-9]+$' | head -1 || true)"
INFLIGHT_VERSION=""

if [ -n "$LATEST_ANY_RC" ]; then
	INFLIGHT_VERSION="${LATEST_ANY_RC%%-rc.*}"
	# Check if a completed GitHub Release exists for this version.
	# The git tag may exist (prepare-release.sh creates it) even if the
	# release failed partway through, so we check the actual release.
	if command -v gh &>/dev/null; then
		if gh release view "$INFLIGHT_VERSION" &>/dev/null; then
			INFLIGHT_VERSION="" # GitHub Release exists, not in-flight
		fi
	else
		# No gh CLI — fall back to tag check
		if git tag --list "$INFLIGHT_VERSION" | grep -q .; then
			INFLIGHT_VERSION=""
		fi
	fi
fi

if [ -n "$INFLIGHT_VERSION" ]; then
	echo ""
	echo "  Found in-flight release: ${LATEST_ANY_RC}"
	echo ""
	printf "Retry this release (${INFLIGHT_VERSION})? [Y/n] "
	read -r RETRY
	if [[ "$RETRY" == "n" || "$RETRY" == "N" ]]; then
		INFLIGHT_VERSION=""
	fi
fi

if [ -n "$INFLIGHT_VERSION" ]; then
	NEXT="$INFLIGHT_VERSION"
else
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
		printf "Enter version (e.g. v1.0.0): "
		read -r NEXT
		if [[ ! "$NEXT" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
			echo "Error: version must be semver like v1.0.0." >&2
			exit 1
		fi
		;;
	*)
		echo "Error: unknown bump type '$BUMP'. Use patch, minor, major, or manual." >&2
		exit 1
		;;
	esac
fi

# ── Compute RC number ─────────────────────────────────────────────

# Find existing RC tags for this version and increment
LATEST_RC="$(git tag --sort=-v:refname | grep -E "^${NEXT}-rc\.[0-9]+$" | head -1 || true)"

if [ -z "$LATEST_RC" ]; then
	RC_NUM=1
else
	RC_NUM="${LATEST_RC##*-rc.}"
	RC_NUM=$((RC_NUM + 1))
fi

RC_TAG="${NEXT}-rc.${RC_NUM}"

# ── Confirm ───────────────────────────────────────────────────────

echo ""
echo "  Current release: ${CURRENT}"
echo "  Next release:    ${NEXT}"
echo "  RC tag:          ${RC_TAG}"
echo ""
echo "This will tag main and push to origin."
echo "CI will then build, test, and create the final ${NEXT} release."
echo ""
printf "Proceed? [y/N] "
read -r CONFIRM
if [[ "$CONFIRM" != "y" && "$CONFIRM" != "Y" ]]; then
	echo "Aborted."
	exit 0
fi

# ── Tag and push ──────────────────────────────────────────────────

git tag "$RC_TAG"
git push origin "$RC_TAG"

echo ""
echo "Tagged ${RC_TAG} and pushed to origin."
echo "Watch CI progress: gh run list --workflow=release.yml"

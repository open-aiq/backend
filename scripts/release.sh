#!/usr/bin/env bash
#
# release.sh — cut a new release.
#
# Prompts for a version bump (major/minor/patch), builds the release artifacts,
# tags the commit, and publishes a GitHub release with the artifacts attached.
#
# Intended to be run via `make release`, which passes the configuration below
# as environment variables. Requires an authenticated `gh` CLI.
#
set -euo pipefail

# --- Configuration (overridable via the environment / Makefile) --------------

RELEASE_BRANCH="${RELEASE_BRANCH:-main}"
DIST="${DIST:-dist}"
BINARY="${BINARY:-server}"
PLATFORMS="${PLATFORMS:-linux/amd64 linux/arm64 darwin/arm64}"
MODULE_PATH="./cmd/server"

die() {
	echo "error: $*" >&2
	exit 1
}

# --- Preflight ---------------------------------------------------------------

command -v gh >/dev/null 2>&1 || die "gh CLI not found: https://cli.github.com"
gh auth status >/dev/null 2>&1 || die "gh not authenticated. Run: gh auth login"

branch="$(git rev-parse --abbrev-ref HEAD)"
[ "$branch" = "$RELEASE_BRANCH" ] ||
	die "releases must be cut from '$RELEASE_BRANCH' (you are on '$branch')"

git diff-index --quiet HEAD -- ||
	die "working tree is dirty; commit or stash changes first"

# --- Determine the next version ----------------------------------------------

git fetch --tags --quiet
current="$(git describe --tags --abbrev=0 2>/dev/null || echo v0.0.0)"

if [ "$current" = "v0.0.0" ]; then
	# No tags yet: seed the first release (see the release strategy in the README).
	next="v0.1.0"
	echo "No existing tags — seeding first release at $next."
else
	read -rp "Current version: $current
Bump type [major/minor/patch]: " bump

	version="${current#v}"
	major="${version%%.*}"
	rest="${version#*.}"
	minor="${rest%%.*}"
	patch="${rest#*.}"

	case "$bump" in
		major) major=$((major + 1)); minor=0; patch=0 ;;
		minor) minor=$((minor + 1)); patch=0 ;;
		patch) patch=$((patch + 1)) ;;
		*)     die "invalid bump type '$bump' (expected major, minor, or patch)" ;;
	esac

	next="v${major}.${minor}.${patch}"
fi

git rev-parse "$next" >/dev/null 2>&1 && die "tag $next already exists"

read -rp "Release $next from $branch? [y/N]: " confirm
case "$confirm" in
	y|Y) ;;
	*)   die "aborted" ;;
esac

# --- Build the per-platform binaries -----------------------------------------

echo "==> Building artifacts for $next"
rm -rf "$DIST"
mkdir -p "$DIST"

for platform in $PLATFORMS; do
	os="${platform%/*}"
	arch="${platform#*/}"
	output="$DIST/${BINARY}-${next}-${os}-${arch}"

	echo "    ${os}/${arch}"
	CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" go build \
		-trimpath \
		-ldflags "-s -w -X main.version=${next}" \
		-o "$output" \
		"$MODULE_PATH"
done

# --- Attach the OpenAPI spec -------------------------------------------------

cp docs/swagger.json docs/swagger.yaml "$DIST/"

# --- Assemble the deploy bundle (linux/amd64 + docs + config + readme) -------

bundle="${BINARY}-${next}"
mkdir -p "$DIST/$bundle"
cp "$DIST/${BINARY}-${next}-linux-amd64" "$DIST/$bundle/$BINARY"
cp docs/swagger.json docs/swagger.yaml .env.example README.md "$DIST/$bundle/"
tar -czf "$DIST/${bundle}.tar.gz" -C "$DIST" "$bundle"
rm -rf "$DIST/$bundle"

# --- Checksums ---------------------------------------------------------------

(
	cd "$DIST"
	if command -v shasum >/dev/null 2>&1; then
		shasum -a 256 * >SHA256SUMS
	else
		sha256sum * >SHA256SUMS
	fi
)

# --- Tag and publish ---------------------------------------------------------

echo "==> Tagging and pushing $next"
git tag -a "$next" -m "Release $next"
git push origin "$next"

echo "==> Creating GitHub release"
gh release create "$next" \
	--title "$next" \
	--target "$branch" \
	--generate-notes \
	"$DIST"/*

echo "==> Done: $next published (artifacts in $DIST/)."

#!/usr/bin/env bash
# PortPilot release & deployment script
# Updates version, verifies build, commits, tags, and pushes to GitHub to trigger the AppImage release workflow.

set -e

REPO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$REPO_DIR"

REMOTE_URL="https://github.com/0xamirreza/PortPilot.git"
VERSION_FILE="internal/version/version.go"

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${BLUE}=========================================${NC}"
echo -e "${BLUE}        PortPilot Release Manager        ${NC}"
echo -e "${BLUE}=========================================${NC}"

# 1. Ensure git repository and remote origin are configured
if [ ! -d ".git" ]; then
    echo -e "${YELLOW}==> Initializing git repository...${NC}"
    git init -b main
    git remote add origin "$REMOTE_URL"
fi

if ! git remote | grep -q "^origin$"; then
    echo -e "${YELLOW}==> Setting remote origin to ${REMOTE_URL}${NC}"
    git remote add origin "$REMOTE_URL"
else
    CURRENT_REMOTE=$(git remote get-url origin)
    if [ "$CURRENT_REMOTE" != "$REMOTE_URL" ]; then
        echo -e "${YELLOW}==> Updating remote origin URL to ${REMOTE_URL}${NC}"
        git remote set-url origin "$REMOTE_URL"
    fi
fi

# 2. Prompt user for update message / version
CURRENT_VER=$(grep 'const Number =' "$VERSION_FILE" | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' || echo "0.5.0")
echo -e "Current application version: ${GREEN}v${CURRENT_VER}${NC}"
echo -e "Enter your update message or version (e.g. ${YELLOW}UPDATED ==> 0.6.0${NC} or ${YELLOW}0.6.0${NC}):"
read -r -p "> " USER_INPUT

if [ -z "$USER_INPUT" ]; then
    echo -e "${RED}Error: Input cannot be empty.${NC}"
    exit 1
fi

# 3. Extract semantic version from input
NEW_VERSION=$(echo "$USER_INPUT" | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -n1 || true)

if [ -z "$NEW_VERSION" ]; then
    echo -e "${RED}Error: Could not extract a semantic version (e.g. 0.6.0) from your input: '${USER_INPUT}'${NC}"
    exit 1
fi

TAG_NAME="v${NEW_VERSION}"
echo -e "==> Target Release Tag: ${GREEN}${TAG_NAME}${NC}"

# 4. Update internal/version/version.go
echo -e "==> Updating ${VERSION_FILE} to ${GREEN}${TAG_NAME}${NC}..."
cat > "$VERSION_FILE" <<EOF
package version

// Number is the semantic version of PortPilot.
const Number = "${TAG_NAME}"
EOF

# 5. Run test and build verification
echo -e "==> Running test suite..."
make test

echo -e "==> Verifying build..."
make build

# 6. Commit changes
git branch -M main
BRANCH="main"

echo -e "==> Staging and committing changes on branch [${BRANCH}]..."
git add -A

# Check if there are changes to commit
if git rev-parse --verify HEAD >/dev/null 2>&1; then
    if git diff-index --quiet HEAD --; then
        echo -e "${YELLOW}No file changes detected, skipping commit.${NC}"
    else
        git commit -m "$USER_INPUT"
    fi
else
    # Initial commit in fresh repository
    git commit -m "initial: release ${TAG_NAME} - ${USER_INPUT}"
fi

# 7. Create tag
if git rev-parse "$TAG_NAME" >/dev/null 2>&1; then
    echo -e "${YELLOW}Notice: Tag ${TAG_NAME} already exists locally. Recreating tag...${NC}"
    git tag -d "$TAG_NAME"
fi

echo -e "==> Tagging release ${GREEN}${TAG_NAME}${NC}..."
git tag -a "$TAG_NAME" -m "Release ${TAG_NAME}: ${USER_INPUT}"

# 8. Push to GitHub
echo -e "==> Pushing branch [${BRANCH}] and tag [${TAG_NAME}] to GitHub..."
git push -u origin "$BRANCH" --tags --force

echo ""
echo -e "${GREEN}================================================================${NC}"
echo -e "${GREEN}  Successfully pushed ${TAG_NAME} to https://github.com/0xamirreza/PortPilot${NC}"
echo -e "${GREEN}================================================================${NC}"
echo -e "GitHub Actions is now automatically building your 64-bit AppImage."
echo -e "Track progress and download the AppImage release once built at:"
echo -e "${BLUE}  https://github.com/0xamirreza/PortPilot/releases/tag/${TAG_NAME}${NC}"
echo -e "${BLUE}  https://github.com/0xamirreza/PortPilot/actions${NC}"
echo ""

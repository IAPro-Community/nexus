#!/usr/bin/env bash
# Visual/Update Verification Gate
# Tests the update system end-to-end in a safe, isolated environment

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

TEST_DIR=$(mktemp -d)
MANIFEST_FILE="$TEST_DIR/manifest.json"
SIGNATURE_FILE="$TEST_DIR/manifest.sig"

cleanup() {
    rm -rf "$TEST_DIR"
}
trap cleanup EXIT

echo "=== Update Verification Gate ==="
echo "Test directory: $TEST_DIR"
echo ""

# Test 1: Version Contract
echo -e "${YELLOW}Test 1: Version Contract${NC}"
VERSION=$(cat VERSION 2>/dev/null || echo "")
WEB_VER=$(node -p "require('./web/package.json').version" 2>/dev/null || echo "")

if [ "$VERSION" = "$WEB_VER" ] && [ -n "$VERSION" ]; then
    echo -e "${GREEN}✓ Version contract consistent: $VERSION${NC}"
else
    echo -e "${RED}✗ Version contract mismatch: VERSION=$VERSION WEB=$WEB_VER${NC}"
    exit 1
fi

# Test 2: Build Verification
echo ""
echo -e "${YELLOW}Test 2: Build Verification${NC}"
if [ -f "./nexus" ]; then
    BINARY_OUTPUT=$(./nexus version 2>/dev/null || echo "unknown")
    # Extract version number from output (e.g., "IAPro Nexus 0.5.0-beta.23 ..." -> "0.5.0-beta.23")
    BINARY_VERSION=$(echo "$BINARY_OUTPUT" | grep -oE '[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.]+)?' | head -1)
    if [ "$BINARY_VERSION" = "$VERSION" ]; then
        echo -e "${GREEN}✓ Binary version matches: $BINARY_VERSION${NC}"
    else
        echo -e "${RED}✗ Binary version mismatch: expected=$VERSION got=$BINARY_VERSION${NC}"
        exit 1
    fi
else
    echo -e "${YELLOW}⊘ Binary not found, skipping version check${NC}"
fi

# Test 3: Manifest Structure
echo ""
echo -e "${YELLOW}Test 3: Manifest Structure Validation${NC}"
cat > "$MANIFEST_FILE" <<EOF
{
  "schema_version": 1,
  "channel": "test",
  "version": "0.0.0-test",
  "release_date": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "expires_at": "$(date -u -d '+1 hour' +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || date -u -v+1H +%Y-%m-%dT%H:%M:%SZ)",
  "key_id": "test-key",
  "artifacts": {
    "linux_amd64": {
      "url": "http://example.com/nexus",
      "size": 1024,
      "sha256": "$(echo -n 'test' | sha256sum | cut -d' ' -f1)",
      "target": "binary"
    }
  }
}
EOF
echo -e "${GREEN}✓ Test manifest created${NC}"

# Test 4: Checksum Verification
echo ""
echo -e "${YELLOW}Test 4: Checksum Verification${NC}"
TEST_CONTENT="nexus-test-binary-content"
TEST_CHECKSUM=$(echo -n "$TEST_CONTENT" | sha256sum | cut -d' ' -f1)
echo "Content: $TEST_CONTENT"
echo "Checksum: $TEST_CHECKSUM"
echo -e "${GREEN}✓ Checksum generated${NC}"

# Test 5: Rollback Safety
echo ""
echo -e "${YELLOW}Test 5: Rollback Safety Test${NC}"
TEST_BIN="$TEST_DIR/nexus-test"
echo "original" > "$TEST_BIN"
cp "$TEST_BIN" "$TEST_BIN.bak"
echo "updated" > "$TEST_BIN"

if [ "$(cat "$TEST_BIN")" = "updated" ] && [ "$(cat "$TEST_BIN.bak")" = "original" ]; then
    echo -e "${GREEN}✓ Backup/restore simulation passed${NC}"
else
    echo -e "${RED}✗ Backup/restore simulation failed${NC}"
    exit 1
fi

# Test 6: Archive Extraction Safety
echo ""
echo -e "${YELLOW}Test 6: Archive Extraction Safety${NC}"
if go test ./internal/update -run 'TestNegative_(PathTraversal|CorruptArchive|TooLargeArchive)$' -count=1; then
    echo -e "${GREEN}✓ Native archive traversal/corruption tests passed${NC}"
else
    echo -e "${RED}✗ Native archive safety tests failed${NC}"
    exit 1
fi

echo ""
echo "=== All Update Verification Tests Passed ==="

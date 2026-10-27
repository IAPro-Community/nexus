package handoff

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"
)

func maestroContextFixture(content string, used, version int, digest string) []byte {
	return []byte(fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"result":{"content":%q,"used":%d,"manifest":{"version":%d,"contentDigest":%q}}}`, content, used, version, digest))
}

func TestDecodeMaestroContextBrief(t *testing.T) {
	content := "# DEV State\nVerified project context"
	digest := sha256.Sum256([]byte(content))
	digestString := hex.EncodeToString(digest[:])
	got, err := decodeMaestroContextBrief(maestroContextFixture(content, len(content), 1, digestString), maestroContextBudget)
	if err != nil || got != content { t.Fatalf("verified context mismatch: %q, %v", got, err) }
	for _, input := range [][]byte{
		maestroContextFixture(content, len(content), 1, strings.Repeat("0", 64)),
		maestroContextFixture(content, maestroContextBudget+1, 1, digestString),
		maestroContextFixture(content, len(content), 2, digestString),
		[]byte(`{"jsonrpc":"2.0","id":1,"error":{"code":-32601}}`),
		[]byte("invalid json"),
	} {
		if _, err := decodeMaestroContextBrief(input, maestroContextBudget); err == nil {
			t.Fatalf("invalid Maestro context accepted: %q", input)
		}
	}
}

func TestDecodeMaestroContextBriefRejectsForgedBudgetAndSupportsUnicode(t *testing.T) {
	content := "Etapa ✅ — contexto validado"
	digest := sha256.Sum256([]byte(content))
	verified := hex.EncodeToString(digest[:])
	// The emoji consumes two UTF-16 code units in the Maestro JavaScript CLI.
	actualUsed := len([]rune(content)) + 1
	if got, err := decodeMaestroContextBrief(maestroContextFixture(content, actualUsed, 1, verified), maestroContextBudget); err != nil || got != content {
		t.Fatalf("valid UTF-16 budget rejected: %q: %v", got, err)
	}
	if _, err := decodeMaestroContextBrief(maestroContextFixture(content, 1, 1, verified), maestroContextBudget); err == nil {
		t.Fatal("forged used count must not pass even with correct content digest")
	}
}

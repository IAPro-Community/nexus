package externalusage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func fixture(id string, tokens int) string {
	return fmt.Sprintf("{\"type\":\"assistant\",\"timestamp\":%q,\"message\":{\"id\":%q,\"model\":\"claude-test\",\"usage\":{\"input_tokens\":%d,\"output_tokens\":4,\"cache_read_input_tokens\":3,\"cache_creation_input_tokens\":2}}}\n",
		time.Now().UTC().Format(time.RFC3339Nano), id, tokens)
}

func TestManualClaudeScanIncrementalAndNoDuplication(t *testing.T) {
	root := t.TempDir()
	s, err := store.Open(filepath.Join(t.TempDir(),"nexus.db"))
	if err != nil { t.Fatal(err) }
	defer s.Close()
	project := filepath.Join(root, "my-project")
	if err := os.MkdirAll(project, 0700); err != nil { t.Fatal(err) }
	file := filepath.Join(project, "session.jsonl")
	if err := os.WriteFile(file, []byte(fixture("same-message",10)+fixture("second",20)+"{\"type\":\"assistant\""),0600); err != nil { t.Fatal(err) }
	ctx := context.Background()
	a, err := ScanClaudeProjects(ctx,s,root)
	if err != nil { t.Fatal(err) }
	if a.Messages != 2 || !a.HasMore { t.Fatalf("first incremental scan: %+v",a) }
	agg, err := s.SummarizeExternalClaude(ctx, 7)
	if err != nil || agg.Messages != 2 || agg.InputTokens != 30 {
		t.Fatalf("summary: %+v %v",agg,err)
	}
	a, err = ScanClaudeProjects(ctx,s,root)
	if err != nil || a.Messages != 0 { t.Fatalf("no-op rescan: %+v %v",a,err) }
	// Complete the partial line with a duplicate message ID.
	duplicate := fixture("same-message",25)
	f, err := os.OpenFile(file,os.O_APPEND|os.O_WRONLY,0600)
	if err != nil { t.Fatal(err) }
	_,err = f.Write([]byte(duplicate[len("{\"type\":\"assistant\""):]))
	_ = f.Close()
	if err != nil { t.Fatal(err) }
	a, err = ScanClaudeProjects(ctx,s,root)
	if err != nil { t.Fatal(err) }
	if a.Messages != 1 { t.Fatalf("completed line missed: %+v",a) }
	agg,err=s.SummarizeExternalClaude(ctx,7)
	if err!=nil || agg.Messages!=2 || agg.InputTokens!=45 {
		t.Fatalf("duplicate-counted message: %+v %v",agg,err)
	}
}

func TestClaudeScanRejectsSymlinkRoot(t *testing.T) {
	target:=t.TempDir()
	link:=filepath.Join(t.TempDir(),"escape")
	if err:=os.Symlink(target,link);err!=nil{t.Skipf("symlink unsupported: %v",err)}
	s,err:=store.Open(filepath.Join(t.TempDir(),"db.sqlite"))
	if err!=nil{t.Fatal(err)}
	defer s.Close()
	if _,err:=ScanClaudeProjects(context.Background(),s,link);err==nil {
		t.Fatal("scanner followed linked source")
	}
}

func TestClaudeUsageRejectsFutureTimestampAndHugeModel(t *testing.T) {
	good := []byte(fixture("message-a", 3))
	if _, ok := parseLine(good); !ok { t.Fatal("normal usage fixture rejected") }
	future := fmt.Sprintf("{\"type\":\"assistant\",\"timestamp\":%q,\"message\":{\"id\":\"a\",\"model\":\"claude-test\",\"usage\":{\"input_tokens\":2,\"output_tokens\":4}}}",
		time.Now().UTC().Add(48*time.Hour).Format(time.RFC3339Nano))
	if _, ok := parseLine([]byte(future)); ok { t.Fatal("future observation accepted") }
	oversized := fmt.Sprintf("{\"type\":\"assistant\",\"timestamp\":%q,\"message\":{\"id\":\"a\",\"model\":%q,\"usage\":{\"input_tokens\":2,\"output_tokens\":4}}}",
		time.Now().UTC().Format(time.RFC3339Nano), strings.Repeat("x", 129))
	if _, ok := parseLine([]byte(oversized)); ok { t.Fatal("unbounded model metadata accepted") }
}

func TestClaudeScanSkipsSymlinkedTranscript(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.jsonl")
	if err := os.WriteFile(outside, []byte(fixture("outside-message", 99)), 0600); err != nil { t.Fatal(err) }
	if err := os.Symlink(outside, filepath.Join(root,"linked.jsonl")); err != nil { t.Skipf("symlink unsupported: %v", err) }
	s, err := store.Open(filepath.Join(t.TempDir(), "nexus.db"))
	if err != nil { t.Fatal(err) }
	defer s.Close()
	report, err := ScanClaudeProjects(context.Background(), s, root)
	if err != nil { t.Fatal(err) }
	if report.Messages != 0 || report.Files != 0 { t.Fatalf("symlinked transcript read: %+v", report) }
	summary, err := s.SummarizeExternalClaude(context.Background(), 7)
	if err != nil || summary.Messages != 0 { t.Fatalf("outside transcript stored: %+v %v", summary, err) }
}

package autonomyguard

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func TestAllowedPathSupportsRecursiveGlob(t *testing.T) {
	patterns := []string{"src/**", "package.json"}
	for _, path := range []string{"src/main.ts", "src/features/auth/index.ts", "package.json"} {
		if !AllowedPath(path, patterns) {
			t.Fatalf("expected %s to be allowed", path)
		}
	}
	if AllowedPath("README.md", patterns) {
		t.Fatal("README.md should not be allowed")
	}
}

func TestForbiddenGitCommandHonorsContract(t *testing.T) {
	cases := []struct {
		args              []string
		destructive, push bool
		forbidden         bool
	}{
		{[]string{"status"}, true, false, false},
		{[]string{"diff"}, true, false, false},
		{[]string{"push", "origin", "main"}, true, false, true},
		{[]string{"push", "origin", "main"}, true, true, false},
		{[]string{"reset", "--hard", "HEAD~1"}, true, false, true},
		{[]string{"clean", "-fd"}, true, false, true},
		{[]string{"branch", "-D", "old"}, true, false, true},
		{[]string{"checkout", "--", "file"}, true, false, true},
		{[]string{"restore", "--source=HEAD", "."}, true, false, true},
		{[]string{"merge", "feature"}, true, false, true},
		{[]string{"rebase", "main"}, true, false, true},
		{[]string{"cherry-pick", "deadbeef"}, true, false, true},
		{[]string{"merge", "feature"}, false, false, false},
		{[]string{"reset", "--hard", "HEAD"}, false, false, false},
	}
	for _, tc := range cases {
		if got := ForbiddenGitCommand(tc.args, tc.destructive, tc.push); got != tc.forbidden {
			t.Fatalf("args=%v got=%v want=%v", tc.args, got, tc.forbidden)
		}
	}
}

func TestChangedSinceDetectsModifiedPreexistingDirtyPath(t *testing.T) {
	before := Snapshot{"src/a.ts": "v1", "README.md": "dirty-v1"}
	after := Snapshot{"src/a.ts": "v2", "README.md": "dirty-v2"}
	changed := ChangedSince(before, after)
	if len(changed) != 2 || changed[0] != "README.md" || changed[1] != "src/a.ts" {
		t.Fatalf("unexpected changed paths: %v", changed)
	}
}

func TestValidateAllowedChangesRejectsOutOfScopeMutation(t *testing.T) {
	err := ValidateAllowedChanges([]string{"src/a.ts", "README.md"}, []string{"src/**"})
	if err == nil {
		t.Fatal("expected out-of-scope mutation to fail")
	}
}

func TestGitChangedPathsIncludesTrackedStagedAndUntracked(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	dir := t.TempDir()
	runGit := func(args ...string) {
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	runGit("init")
	runGit("config", "user.email", "test@example.com")
	runGit("config", "user.name", "Test")
	if err := os.WriteFile(filepath.Join(dir, "tracked.txt"), []byte("a"), 0600); err != nil {
		t.Fatal(err)
	}
	runGit("add", "tracked.txt")
	runGit("commit", "-m", "init")
	if err := os.WriteFile(filepath.Join(dir, "tracked.txt"), []byte("b"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "staged.txt"), []byte("s"), 0600); err != nil {
		t.Fatal(err)
	}
	runGit("add", "staged.txt")
	if err := os.WriteFile(filepath.Join(dir, "new.txt"), []byte("n"), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := GitChangedPaths(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"new.txt", "staged.txt", "tracked.txt"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestWriteCommandGuardsBlocksGitPushAndProxiesSafeGit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix wrapper execution test")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	dir := t.TempDir()
	created, err := WriteCommandGuards(dir, Policy{DisallowDestructiveGit: true, AllowGitPush: false, AllowDeploy: false})
	if err != nil {
		t.Fatal(err)
	}
	if len(created) == 0 {
		t.Fatal("expected at least git guard")
	}
	guardGit := filepath.Join(dir, "git")
	cmd := exec.Command(guardGit, "--version")
	if out, err := cmd.CombinedOutput(); err != nil || !strings.Contains(string(out), "git version") {
		t.Fatalf("safe git command was not proxied: %v %s", err, out)
	}
	cmd = exec.Command(guardGit, "push", "origin", "main")
	out, err := cmd.CombinedOutput()
	if err == nil || !strings.Contains(string(out), "NEXUS_AUTONOMY_BLOCKED") {
		t.Fatalf("git push was not blocked: %v %s", err, out)
	}
	cmd = exec.Command(guardGit, "merge", "feature")
	out, err = cmd.CombinedOutput()
	if err == nil || !strings.Contains(string(out), "NEXUS_AUTONOMY_BLOCKED") {
		t.Fatalf("git merge was not blocked: %v %s", err, out)
	}
}

func TestRenderGuardBlocksDeployMutationButAllowsReadOnly(t *testing.T) {
	script := renderUnixCommandGuard("/usr/bin/kubectl", "kubectl", Policy{AllowDeploy: false})
	if !strings.Contains(script, "apply") || !strings.Contains(script, "NEXUS_AUTONOMY_BLOCKED") {
		t.Fatalf("kubectl guard missing mutation block: %s", script)
	}
	if strings.Contains(script, `" get "*) exit`) {
		t.Fatalf("read-only kubectl get must not be blocked: %s", script)
	}
}

func TestRenderGuardBlocksNetworkSecretAndPaidToolsWithDeployDenied(t *testing.T) {
	policy := Policy{AllowDeploy: false}
	for _, tool := range []string{"curl", "vault", "aws"} {
		script := renderUnixCommandGuard("/usr/bin/"+tool, tool, policy)
		if !strings.Contains(script, "block\n") {
			t.Fatalf("%s guard must block unconditionally: %s", tool, script)
		}
		if strings.Contains(script, "exec ") {
			t.Fatalf("%s guard must not exec real binary when blocked: %s", tool, script)
		}
	}
}

func TestWriteCommandGuardsBlocksNetworkToolsWhenDeployDenied(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix wrapper execution test")
	}
	if _, err := exec.LookPath("curl"); err != nil {
		t.Skip("curl not installed")
	}
	dir := t.TempDir()
	created, err := WriteCommandGuards(dir, Policy{AllowDeploy: false})
	if err != nil {
		t.Fatal(err)
	}
	if len(created) == 0 {
		t.Fatal("expected guards to be created")
	}
	guardCurl := filepath.Join(dir, "curl")
	out, err := exec.Command(guardCurl, "https://example.com").CombinedOutput()
	if err == nil || !strings.Contains(string(out), "NEXUS_AUTONOMY_BLOCKED") {
		t.Fatalf("curl was not blocked with deploy denied: %v %s", err, out)
	}
}

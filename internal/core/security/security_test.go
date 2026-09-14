package security

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IAPro-Community/nexus/internal/core/model"
)

func TestRedaction(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string
		excludes []string
	}{
		{
			name:     "OpenAI Key",
			input:    "Authorization: Bearer sk-1234567890abcdef1234567890\napi_key: sk-1234567890abcdef1234567890",
			contains: "[REDACTED",
			excludes: []string{"sk-1234567890abcdef1234567890"},
		},
		{
			name:     "Anthropic Key",
			input:    "x-api-key: sk-ant-api03-abcdef1234567890abcdef1234567890-test",
			contains: "[REDACTED",
			excludes: []string{"sk-ant-api03-abcdef1234567890abcdef1234567890-test"},
		},
		{
			name:     "Google OAuth Token",
			input:    "access_token: ya29.a0AfH6SMBxyz1234567890abcdef1234567890",
			contains: "[REDACTED",
			excludes: []string{"ya29.a0AfH6SMBxyz1234567890abcdef1234567890"},
		},
		{
			name:     "Private Key",
			input:    "-----BEGIN RSA PRIVATE KEY-----\nMIIEowIBAAKCAQEA0...\n-----END RSA PRIVATE KEY-----",
			contains: "[REDACTED_PRIVATE_KEY]",
			excludes: []string{"MIIEowIBAAKCAQEA0"},
		},
		{
			name:     "JWT Token",
			input:    "token: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIn0.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c",
			contains: "[REDACTED",
			excludes: []string{"SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"},
		},
		{
			name:     "AWS Access Key",
			input:    "AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE",
			contains: "[REDACTED_AWS_KEY]",
			excludes: []string{"AKIAIOSFODNN7EXAMPLE"},
		},
		{
			name:     "GitHub Token",
			input:    "GITHUB_TOKEN=ghp_1234567890abcdefghijklmnopqrstuvwxyz",
			contains: "[REDACTED_GITHUB_TOKEN]",
			excludes: []string{"ghp_1234567890abcdefghijklmnopqrstuvwxyz"},
		},
		{
			name:     "GitHub OAuth Token",
			input:    "GITHUB_TOKEN=gho_abcdef1234567890abcdefghijklmnopqrstuvwxyz",
			contains: "[REDACTED_GITHUB_TOKEN]",
			excludes: []string{"gho_abcdef1234567890abcdefghijklmnopqrstuvwxyz"},
		},
		{
			name:     "Database Connection URI Password",
			input:    "DATABASE_URL=postgres://dbuser:s3cr3tP@ss!@localhost:5432/mydb",
			contains: "postgres://dbuser:[REDACTED]@localhost:5432/mydb",
			excludes: []string{"s3cr3tP@ss!"},
		},
		{
			name:     "Password with special chars",
			input:    "PASSWORD=myComplexP@ssw0rd!#456",
			contains: "PASSWORD=[REDACTED]",
			excludes: []string{"myComplexP@ssw0rd!#456"},
		},
		{
			name: "env block",
			input: "OPENAI_API_KEY=sk-proj-1234567890abcdef1234567890abcdef1234567890\n" +
				"ANTHROPIC_API_KEY=sk-ant-api03-abcdef1234567890abcdef1234567890-test\n" +
				"GOOGLE_API_KEY=AIzaSyD-1234567890abcdefghijklmnopqrstuvwxyz\n" +
				"GITHUB_TOKEN=ghp_1234567890abcdefghijklmnopqrstuvwxyz",
			contains: "[REDACTED",
			excludes: []string{"sk-proj-1234567890", "sk-ant-api03-abcdef", "AIzaSyD-1234567890", "ghp_1234567890"},
		},
		{
			name:     "auth.json",
			input:    `auth: {"access_token": "ya29.a0AfH6SMBxyz1234567890abcdef1234567890", "refresh_token": "1//0cdefghijklmnopqrstuvwxyzABCDEF"}`,
			contains: "[REDACTED",
			excludes: []string{"ya29.a0AfH6SMB", "1//0cdefghijklmnopqrstuvwxyz"},
		},
		{
			name:     "cookies header",
			input:    "Cookie: sessionid=abc123; csrftoken=xyz789; password=supersecret",
			contains: "[REDACTED",
			excludes: []string{"sessionid=abc123", "csrftoken=xyz789", "supersecret"},
		},
		{
			name:     "pem private key",
			input:    "-----BEGIN OPENSSH PRIVATE KEY-----\nb3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQ==\n-----END OPENSSH PRIVATE KEY-----",
			contains: "[REDACTED_PRIVATE_KEY]",
			excludes: []string{"b3BlbnNzaC1rZXktdjE"},
		},
		{
			name:     "generic credentials",
			input:    "CREDENTIALS=admin:topsecret123",
			contains: "[REDACTED",
			excludes: []string{"topsecret123"},
		},
		{
			name:     "Bearer with base64 padding",
			input:    "Authorization: Bearer abc1234567===",
			contains: "[REDACTED",
			excludes: []string{"abc1234567==="},
		},
		{
			name:     "short secret value",
			input:    "password=abcde",
			contains: "password=[REDACTED]",
			excludes: []string{"abcde"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Redact(tc.input)
			if !strings.Contains(got, tc.contains) {
				t.Fatalf("expected output to contain %q, got %q", tc.contains, got)
			}
			for _, ex := range tc.excludes {
				if strings.Contains(got, ex) {
					t.Fatalf("redaction leaked secret %q in %q", ex, got)
				}
			}
		})
	}
}

func TestRedactSlice(t *testing.T) {
	input := []string{
		"safe-arg",
		"PASSWORD=secret123",
		"Authorization: Bearer sk-ant-api03-abcdef1234567890abcdef1234567890",
	}
	res := RedactSlice(input)
	if len(res) != 3 {
		t.Fatalf("expected 3 items, got %d", len(res))
	}
	if res[0] != "safe-arg" {
		t.Errorf("expected safe-arg unchanged, got %q", res[0])
	}
	if strings.Contains(res[1], "secret123") {
		t.Errorf("expected password redacted, got %q", res[1])
	}
	if strings.Contains(res[2], "sk-ant") {
		t.Errorf("expected api key redacted, got %q", res[2])
	}
}

func TestIsolationPolicies(t *testing.T) {
	hostHome := t.TempDir()
	t.Setenv("AI_REAL_HOME", hostHome)

	// Create host secrets
	_ = os.WriteFile(filepath.Join(hostHome, ".git-credentials"), []byte("https://user:pass@github.com\n"), 0600)
	_ = os.MkdirAll(filepath.Join(hostHome, ".ssh"), 0700)
	_ = os.WriteFile(filepath.Join(hostHome, ".ssh", "id_rsa"), []byte("fake_key"), 0600)
	_ = os.WriteFile(filepath.Join(hostHome, ".gitconfig"), []byte("[user]\nname=Developer\n"), 0600)

	// Test Developer preset
	profileDev := t.TempDir()
	devPolicy := GetPolicy(model.IsolationDeveloper)
	if err := ApplyIsolation(profileDev, devPolicy); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Lstat(filepath.Join(profileDev, ".gitconfig")); err != nil {
		t.Errorf("developer preset should share .gitconfig")
	}
	if _, err := os.Lstat(filepath.Join(profileDev, ".ssh")); err == nil {
		t.Errorf("developer preset MUST NOT share ~/.ssh")
	}
	if _, err := os.Lstat(filepath.Join(profileDev, ".git-credentials")); err == nil {
		t.Errorf("developer preset MUST NOT share ~/.git-credentials")
	}

	audit := AuditProfile("codex", "dev", profileDev, hostHome, model.IsolationDeveloper)
	if len(audit.Warnings) > 0 {
		t.Fatalf("unexpected audit warnings: %+v", audit.Warnings)
	}

	// Test Strict preset
	profileStrict := t.TempDir()
	strictPolicy := GetPolicy(model.IsolationStrict)
	if err := ApplyIsolation(profileStrict, strictPolicy); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Lstat(filepath.Join(profileStrict, ".gitconfig")); err == nil {
		t.Errorf("strict preset MUST NOT share .gitconfig")
	}
	if _, err := os.Lstat(filepath.Join(profileStrict, ".ssh")); err == nil {
		t.Errorf("strict preset MUST NOT share ~/.ssh")
	}
}

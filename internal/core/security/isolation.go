package security

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/IAPro-Community/nexus/internal/core/model"
)

// Policy defines what paths and environment features are permitted to be shared.
type Policy struct {
	Preset          model.IsolationPreset `json:"preset"`
	AllowGitConfig  bool                  `json:"allow_git_config"`
	AllowSSHAgent   bool                  `json:"allow_ssh_agent"`
	AllowNpmrc      bool                  `json:"allow_npmrc"`
	AllowHostSSHDir bool                  `json:"allow_host_ssh_dir"`
	AllowGitCreds   bool                  `json:"allow_git_creds"`
	AllowKube       bool                  `json:"allow_kube"`
	AllowDocker     bool                  `json:"allow_docker"`
	AllowGPG        bool                  `json:"allow_gpg"`
}

// GetPolicy returns the policy configuration for a given preset.
func GetPolicy(preset model.IsolationPreset) Policy {
	switch preset {
	case model.IsolationStrict:
		return Policy{
			Preset:          model.IsolationStrict,
			AllowGitConfig:  false,
			AllowSSHAgent:   true, // Safe socket forwarding if present
			AllowNpmrc:      false,
			AllowHostSSHDir: false,
			AllowGitCreds:   false,
			AllowKube:       false,
			AllowDocker:     false,
			AllowGPG:        false,
		}
	case model.IsolationCompat:
		return Policy{
			Preset:          model.IsolationCompat,
			AllowGitConfig:  true,
			AllowSSHAgent:   true,
			AllowNpmrc:      true,
			AllowHostSSHDir: true,
			AllowGitCreds:   true,
			AllowKube:       true,
			AllowDocker:     true,
			AllowGPG:        true,
		}
	case model.IsolationDeveloper:
		fallthrough
	default:
		return Policy{
			Preset:          model.IsolationDeveloper,
			AllowGitConfig:  true,
			AllowSSHAgent:   true,
			AllowNpmrc:      true,
			AllowHostSSHDir: false, // Never share ~/.ssh private keys by default
			AllowGitCreds:   false, // Never share ~/.git-credentials by default
			AllowKube:       false, // Never share ~/.kube by default
			AllowDocker:     false, // Never share ~/.docker by default
			AllowGPG:        false, // Never share ~/.gnupg by default
		}
	}
}

// SecurityAudit represents the audited permissions for a profile.
type SecurityAudit struct {
	Profile   string                `json:"profile"`
	Preset    model.IsolationPreset `json:"preset"`
	Shared    []string              `json:"shared"`
	Protected []string              `json:"protected"`
	Warnings  []string              `json:"warnings,omitempty"`
}

// AuditProfile checks what host directories are shared vs protected for a profile home.
func AuditProfile(provider, profileName, profileHome, hostHome string, preset model.IsolationPreset) SecurityAudit {
	policy := GetPolicy(preset)
	audit := SecurityAudit{
		Profile:   fmt.Sprintf("%s:%s", provider, profileName),
		Preset:    policy.Preset,
		Shared:    []string{},
		Protected: []string{},
	}

	checkItem := func(name string, allowed bool, description string) {
		profilePath := filepath.Join(profileHome, name)
		fi, err := os.Lstat(profilePath)
		isLinked := err == nil && (fi.Mode()&os.ModeSymlink != 0 || fi.IsDir() || fi.Mode().IsRegular())

		if isLinked {
			if allowed {
				audit.Shared = append(audit.Shared, fmt.Sprintf("%s (%s)", name, description))
			} else {
				audit.Shared = append(audit.Shared, fmt.Sprintf("%s (EXPOSED - not permitted under %s preset)", name, preset))
				audit.Warnings = append(audit.Warnings, fmt.Sprintf("Sensitive path %s is linked into profile home despite %s preset", name, preset))
			}
		} else {
			audit.Protected = append(audit.Protected, fmt.Sprintf("%s (%s)", name, description))
		}
	}

	checkItem(".gitconfig", policy.AllowGitConfig, "Git user name/email configuration")
	checkItem(".git-credentials", policy.AllowGitCreds, "Git plain-text credentials store")
	checkItem(".ssh", policy.AllowHostSSHDir, "SSH directory and private keys")
	checkItem(".gnupg", policy.AllowGPG, "GPG keyrings and private secrets")
	checkItem(".kube", policy.AllowKube, "Kubernetes cluster credentials")
	checkItem(".docker", policy.AllowDocker, "Docker registry authentication tokens")
	checkItem(".npmrc", policy.AllowNpmrc, "NPM registry token config")

	if os.Getenv("SSH_AUTH_SOCK") != "" && policy.AllowSSHAgent {
		audit.Shared = append(audit.Shared, "SSH_AUTH_SOCK (SSH Agent socket forwarding - no private keys exposed)")
	}

	return audit
}

// FindHostHome determines the real host home directory dynamically without hardcoded usernames.
func FindHostHome() string {
	if v := os.Getenv("AI_REAL_HOME"); v != "" {
		if st, err := os.Stat(v); err == nil && st.IsDir() {
			return v
		}
	}
	if v := os.Getenv("USERPROFILE"); v != "" {
		if st, err := os.Stat(v); err == nil && st.IsDir() {
			return v
		}
	}
	if h, err := os.UserHomeDir(); err == nil && h != "" {
		if st, err := os.Stat(h); err == nil && st.IsDir() {
			return h
		}
	}
	if d, p := os.Getenv("HOMEDRIVE"), os.Getenv("HOMEPATH"); d != "" && p != "" {
		full := filepath.Join(d, p)
		if st, err := os.Stat(full); err == nil && st.IsDir() {
			return full
		}
	}
	if u := os.Getenv("USERNAME"); u != "" {
		p := filepath.Join(`C:\Users`, u)
		if st, err := os.Stat(p); err == nil && st.IsDir() {
			return p
		}
	}
	if u := os.Getenv("USER"); u != "" {
		p := filepath.Join("/home", u)
		if st, err := os.Stat(p); err == nil && st.IsDir() {
			return p
		}
	}
	return ""
}

// ApplyIsolation links allowed files and cleans disallowed links from a profile home.
func ApplyIsolation(profileHome string, policy Policy) error {
	hostHome := FindHostHome()
	if hostHome == "" || profileHome == "" || hostHome == profileHome {
		return nil
	}

	syncLink := func(name string, allow bool) {
		src := filepath.Join(hostHome, name)
		dst := filepath.Join(profileHome, name)

		if !allow {
			if fi, err := os.Lstat(dst); err == nil && fi.Mode()&os.ModeSymlink != 0 {
				_ = os.Remove(dst)
			}
			return
		}

		_ = SafeLinkOrCopy(src, dst)
	}

	syncLink(".gitconfig", policy.AllowGitConfig)
	syncLink(".git-credentials", policy.AllowGitCreds)
	syncLink(".ssh", policy.AllowHostSSHDir)
	syncLink(".gnupg", policy.AllowGPG)
	syncLink(".kube", policy.AllowKube)
	syncLink(".docker", policy.AllowDocker)
	syncLink(".npmrc", policy.AllowNpmrc)

	if policy.Preset != model.IsolationStrict {
		hostConfig := filepath.Join(hostHome, ".config")
		profileConfig := filepath.Join(profileHome, ".config")
		_ = os.MkdirAll(profileConfig, 0700)
		if entries, err := os.ReadDir(hostConfig); err == nil {
			for _, e := range entries {
				if e.Name() == "ai-cli" || e.Name() == "ai-manager" {
					continue
				}
				src := filepath.Join(hostConfig, e.Name())
				dst := filepath.Join(profileConfig, e.Name())
				_ = SafeLinkOrCopy(src, dst)
			}
		}
	}

	return nil
}

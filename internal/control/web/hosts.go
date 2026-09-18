package web

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

const (
	nexusHostname = "nexus.dev"
	nexusIP       = "127.0.0.1"
	hostsComment  = "# IAPro Nexus — local development hostname"
)

// EnsureHostsEntry adds a hostname → IP mapping to /etc/hosts (or equivalent)
// if it does not already exist. On failure it returns an error but never
// prevents the caller from continuing.
func EnsureHostsEntry(hostname, ip string) error {
	return modifyHostsEntry(hostname, ip, true)
}

// RemoveHostsEntry removes a previously registered hostname → IP mapping from
// /etc/hosts. It is idempotent: if the entry does not exist it returns nil.
func RemoveHostsEntry(hostname, ip string) error {
	return modifyHostsEntry(hostname, ip, false)
}

// IsHostsEntryPresent reports whether /etc/hosts already contains the given
// hostname/IP pair.
func IsHostsEntryPresent(hostname, ip string) (bool, error) {
	lines, err := readHostsFile()
	if err != nil {
		return false, err
	}
	target := formatHostsLine(ip, hostname)
	for _, line := range lines {
		if normalizedLine(line) == normalizedLine(target) {
			return true, nil
		}
	}
	return false, nil
}

func modifyHostsEntry(hostname, ip string, add bool) error {
	if runtime.GOOS == "windows" {
		return modifyHostsWindows(hostname, ip, add)
	}
	return modifyHostsUnix(hostname, ip, add)
}

// --- Unix (Linux / macOS) ---

func modifyHostsUnix(hostname, ip string, add bool) error {
	lines, err := readHostsFile()
	if err != nil {
		return fmt.Errorf("reading /etc/hosts: %w", err)
	}

	target := formatHostsLine(ip, hostname)
	found := false
	for _, line := range lines {
		if normalizedLine(line) == normalizedLine(target) {
			found = true
			break
		}
	}

	if add && found {
		return nil // already present
	}
	if !add && !found {
		return nil // already absent
	}

	var newLines []string
	for _, line := range lines {
		if normalizedLine(line) == normalizedLine(target) {
			continue // remove
		}
		// Also remove the comment line that precedes this entry
		if strings.TrimSpace(line) == hostsComment {
			if len(newLines) > 0 {
				prev := newLines[len(newLines)-1]
				if strings.TrimSpace(prev) == hostsComment {
					newLines = newLines[:len(newLines)-1]
				}
			}
			continue
		}
		newLines = append(newLines, line)
	}

	if add {
		// Append with a descriptive comment
		newLines = append(newLines, hostsComment)
		newLines = append(newLines, target)
	}

	// Ensure trailing newline
	content := strings.Join(newLines, "\n")
	if len(content) > 0 && content[len(content)-1] != '\n' {
		content += "\n"
	}

	return writeHostsFile(content)
}

func readHostsFile() ([]string, error) {
	f, err := os.Open(hostsFilePath())
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

func writeHostsFile(content string) error {
	path := hostsFilePath()

	// Try direct write first (works if running as root or with passwordless sudo).
	writeErr := os.WriteFile(path, []byte(content), 0o644)
	if writeErr == nil {
		return nil
	}

	// Fallback: use sudo tee (Unix only).
	if runtime.GOOS == "windows" {
		return fmt.Errorf("writing %s: %w", path, writeErr)
	}
	cmd := exec.Command("sudo", "-n", "tee", path)
	cmd.Stdin = strings.NewReader(content)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("sudo tee %s: %w", path, err)
	}
	return nil
}

func hostsFilePath() string {
	if runtime.GOOS == "windows" {
		return `C:\Windows\System32\drivers\etc\hosts`
	}
	return "/etc/hosts"
}

// --- Windows ---

func modifyHostsWindows(hostname, ip string, add bool) error {
	lines, err := readHostsFile()
	if err != nil {
		return fmt.Errorf("reading hosts file: %w", err)
	}

	target := formatHostsLine(ip, hostname)
	found := false
	for _, line := range lines {
		if normalizedLine(line) == normalizedLine(target) {
			found = true
			break
		}
	}

	if add && found {
		return nil
	}
	if !add && !found {
		return nil
	}

	var newLines []string
	for _, line := range lines {
		if normalizedLine(line) == normalizedLine(target) {
			continue
		}
		if strings.TrimSpace(line) == hostsComment {
			if len(newLines) > 0 {
				prev := newLines[len(newLines)-1]
				if strings.TrimSpace(prev) == hostsComment {
					newLines = newLines[:len(newLines)-1]
				}
			}
			continue
		}
		newLines = append(newLines, line)
	}

	if add {
		newLines = append(newLines, hostsComment)
		newLines = append(newLines, target)
	}

	content := strings.Join(newLines, "\n")
	if len(content) > 0 && content[len(content)-1] != '\n' {
		content += "\n"
	}

	path := hostsFilePath()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("writing %s (may require Administrator): %w", path, err)
	}
	return nil
}

// --- Helpers ---

func formatHostsLine(ip, hostname string) string {
	return ip + "\t" + hostname
}

func normalizedLine(s string) string {
	return strings.TrimSpace(s)
}

// EnsureNexusHostsEntry is a convenience wrapper for the standard nexus.dev entry.
func EnsureNexusHostsEntry() error {
	return EnsureHostsEntry(nexusHostname, nexusIP)
}

// RemoveNexusHostsEntry is a convenience wrapper for the standard nexus.dev entry.
func RemoveNexusHostsEntry() error {
	return RemoveHostsEntry(nexusHostname, nexusIP)
}

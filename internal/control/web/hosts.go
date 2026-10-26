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
	return hostsEntryPresent(lines, ip, hostname), nil
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

	if hostsEntryPresent(lines, ip, hostname) == add {
		return nil // already present (add) or already absent (remove)
	}

	newLines := applyHostsChange(lines, hostname, ip, add)

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

	if hostsEntryPresent(lines, ip, hostname) == add {
		return nil // already present (add) or already absent (remove)
	}

	newLines := applyHostsChange(lines, hostname, ip, add)

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

// hostsLineFields parses the address and host aliases before any inline comment.
func hostsLineFields(line string) []string {
	if comment := strings.IndexByte(line, '#'); comment >= 0 {
		line = line[:comment]
	}
	return strings.Fields(line)
}

// hostsEntryMatch compares tokens rather than a whole line, since a hosts entry
// may share its address with aliases or use a different separator.
func hostsEntryMatch(line, ip, hostname string) bool {
	fields := hostsLineFields(line)
	if len(fields) < 2 || !strings.EqualFold(fields[0], ip) {
		return false
	}
	for _, alias := range fields[1:] {
		if strings.EqualFold(alias, hostname) {
			return true
		}
	}
	return false
}

func hostsEntryPresent(lines []string, ip, hostname string) bool {
	for _, line := range lines {
		if hostsEntryMatch(line, ip, hostname) {
			return true
		}
	}
	return false
}

// stripHostnameFromLine removes only the requested alias, preserving other
// aliases and any inline comment on the same hosts-file line.
func stripHostnameFromLine(line, ip, hostname string) (string, bool) {
	comment := ""
	if commentStart := strings.IndexByte(line, '#'); commentStart >= 0 {
		comment = strings.TrimSpace(line[commentStart:])
		line = line[:commentStart]
	}
	fields := strings.Fields(line)
	if len(fields) < 2 || !strings.EqualFold(fields[0], ip) {
		return line, true
	}

	kept := []string{fields[0]}
	for _, alias := range fields[1:] {
		if !strings.EqualFold(alias, hostname) {
			kept = append(kept, alias)
		}
	}
	if len(kept) == 1 {
		if comment == "" {
			return "", false
		}
		return comment, true
	}
	result := strings.Join(kept, "\t")
	if comment != "" {
		result += "\t" + comment
	}
	return result, true
}

// applyHostsChange updates matching aliases without dropping unrelated aliases
// or comments. It never prunes, rewrites permissions, or touches a real hosts
// file; callers decide how the returned lines are persisted.
func applyHostsChange(lines []string, hostname, ip string, add bool) []string {
	if add && hostsEntryPresent(lines, ip, hostname) {
		return lines
	}

	updated := make([]string, 0, len(lines)+2)
	for _, line := range lines {
		if add || !hostsEntryMatch(line, ip, hostname) {
			updated = append(updated, line)
			continue
		}

		remaining, keepLine := stripHostnameFromLine(line, ip, hostname)
		if keepLine {
			if len(hostsLineFields(remaining)) == 0 && len(updated) > 0 && strings.TrimSpace(updated[len(updated)-1]) == hostsComment {
				updated = updated[:len(updated)-1]
			}
			updated = append(updated, remaining)
			continue
		}
		if len(updated) > 0 && strings.TrimSpace(updated[len(updated)-1]) == hostsComment {
			updated = updated[:len(updated)-1]
		}
	}

	if add {
		updated = append(updated, hostsComment, formatHostsLine(ip, hostname))
	}
	return updated
}

// EnsureNexusHostsEntry is a convenience wrapper for the standard nexus.dev entry.
func EnsureNexusHostsEntry() error {
	return EnsureHostsEntry(nexusHostname, nexusIP)
}

// RemoveNexusHostsEntry is a convenience wrapper for the standard nexus.dev entry.
func RemoveNexusHostsEntry() error {
	return RemoveHostsEntry(nexusHostname, nexusIP)
}

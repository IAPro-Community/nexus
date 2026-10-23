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

func normalizedLine(s string) string {
	return strings.TrimSpace(s)
}

// hostsLineFields splits a hosts-file line into its address and hostnames,
// dropping any inline comment.
func hostsLineFields(line string) []string {
	if i := strings.IndexByte(line, '#'); i >= 0 {
		line = line[:i]
	}
	return strings.Fields(line)
}

// hostsEntryMatch reports whether a hosts-file line already maps ip to
// hostname. It compares parsed fields instead of the literal line so that the
// separator (space vs tab), extra aliases and inline comments cannot hide an
// entry that is already present.
func hostsEntryMatch(line, ip, hostname string) bool {
	fields := hostsLineFields(line)
	if len(fields) < 2 || !strings.EqualFold(fields[0], ip) {
		return false
	}
	for _, name := range fields[1:] {
		if strings.EqualFold(name, hostname) {
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

// stripHostnameFromLine removes hostname from a hosts-file line that maps it to
// ip, keeping any other aliases. It reports whether an entry remains on the
// line; when false the caller must drop the whole line.
func stripHostnameFromLine(line, ip, hostname string) (string, bool) {
	fields := hostsLineFields(line)
	if len(fields) < 2 || !strings.EqualFold(fields[0], ip) {
		return line, true
	}
	kept := []string{fields[0]}
	for _, name := range fields[1:] {
		if !strings.EqualFold(name, hostname) {
			kept = append(kept, name)
		}
	}
	if len(kept) == 1 {
		return "", false
	}
	return strings.Join(kept, "	"), true
}

// applyHostsChange returns the hosts-file lines with hostname mapped to ip
// added or removed. It is a no-op when the entry is already in the requested
// state. Removing an entry keeps the other aliases sharing its line.
func applyHostsChange(lines []string, hostname, ip string, add bool) []string {
	if add && hostsEntryPresent(lines, ip, hostname) {
		return lines // already present, leave the file untouched
	}

	target := formatHostsLine(ip, hostname)
	out := make([]string, 0, len(lines)+2)
	for _, line := range lines {
		if !add && hostsEntryMatch(line, ip, hostname) {
			if kept, ok := stripHostnameFromLine(line, ip, hostname); ok {
				out = append(out, kept)
				continue
			}
			// Whole entry removed: drop the descriptive comment we own above it.
			if len(out) > 0 && strings.TrimSpace(out[len(out)-1]) == hostsComment {
				out = out[:len(out)-1]
			}
			continue
		}
		if add && strings.TrimSpace(line) == hostsComment {
			continue // moved next to the new entry below
		}
		out = append(out, line)
	}
	if add {
		out = append(out, hostsComment, target)
	}
	return out
}

// EnsureNexusHostsEntry is a convenience wrapper for the standard nexus.dev entry.
func EnsureNexusHostsEntry() error {
	return EnsureHostsEntry(nexusHostname, nexusIP)
}

// RemoveNexusHostsEntry is a convenience wrapper for the standard nexus.dev entry.
func RemoveNexusHostsEntry() error {
	return RemoveHostsEntry(nexusHostname, nexusIP)
}

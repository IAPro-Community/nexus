//go:build linux

package runtime

import "path/filepath"

const isolatedSecretServiceScript = `TMPDIR=$(mktemp -d /tmp/nexus-kr-XXXXXX) || exit 1
trap 'rm -rf "$TMPDIR"' EXIT INT TERM
export GNOME_KEYRING_CONTROL="$TMPDIR"
daemon="$1"
shift
"$daemon" --daemonize --components=secrets --control-directory="$TMPDIR" >/dev/null 2>&1 || true
exec "$@"`

func alreadyIsolatedSecretServiceArgv(args []string) bool {
	if len(args) >= 4 && args[0] == "--" && args[1] == "/bin/sh" && args[2] == "-c" {
		return true
	}
	if len(args) >= 2 && args[0] == "--" {
		base := filepath.Base(args[1])
		if base != "sh" && base != "dbus-run-session" && args[1] != "--" {
			return true
		}
	}
	return false
}

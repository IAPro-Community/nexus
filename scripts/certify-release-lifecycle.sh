#!/usr/bin/env bash
# Signed release lifecycle certification:
#   install previous -> upgrade candidate -> reject signed wrong-version artifact
#   -> reject tampered artifact -> uninstall preserving data -> purge data.
#
# Uses the protected release Ed25519 key so the exact installer trust contract is
# exercised before a stable release is authorized.
set -euo pipefail

: "${NEXUS_UPDATE_PRIVATE_KEY:?NEXUS_UPDATE_PRIVATE_KEY is required}"
: "${NEXUS_UPDATE_PUBLIC_KEY:?NEXUS_UPDATE_PUBLIC_KEY is required}"
: "${NEXUS_UPDATE_KEY_ID:?NEXUS_UPDATE_KEY_ID is required}"

NEXUS_BIN="${NEXUS_BIN:-./nexus}"
CANDIDATE_VERSION="${NEXUS_RELEASE_VERSION:-$(tr -d '\r\n[:space:]' < VERSION)}"
CANDIDATE_VERSION="${CANDIDATE_VERSION#v}"
if [[ "$CANDIDATE_VERSION" =~ -(alpha|beta|rc)\. ]]; then
    default_old_version="0.9.999-beta.cert.1"
else
    default_old_version="0.9.999-cert.1"
fi
OLD_VERSION="${NEXUS_LIFECYCLE_OLD_VERSION:-$default_old_version}"
MISMATCH_VERSION="${CANDIDATE_VERSION}-mismatch.1"

canonical="$(tr -d '[:space:]' < UPDATE_PUBLIC_KEY)"
canonical_key_id="$(tr -d '\r\n[:space:]' < UPDATE_KEY_ID)"
test "$NEXUS_UPDATE_PUBLIC_KEY" = "$canonical" || {
    echo "NEXUS_UPDATE_PUBLIC_KEY does not match UPDATE_PUBLIC_KEY" >&2
    exit 1
}
test "$NEXUS_UPDATE_KEY_ID" = "$canonical_key_id" || {
    echo "NEXUS_UPDATE_KEY_ID does not match UPDATE_KEY_ID" >&2
    exit 1
}

command -v go >/dev/null
command -v python3 >/dev/null
command -v curl >/dev/null
command -v tar >/dev/null
python3 -c 'from cryptography.hazmat.primitives.asymmetric.ed25519 import Ed25519PublicKey' >/dev/null

if [ ! -x "$NEXUS_BIN" ]; then
    echo "Candidate binary is not executable: $NEXUS_BIN" >&2
    exit 2
fi

candidate_reported="$("$NEXUS_BIN" version --json | python3 -c 'import json,sys; print(json.load(sys.stdin)["version"])')"
if [ "$candidate_reported" != "$CANDIDATE_VERSION" ]; then
    echo "Candidate binary version=$candidate_reported, expected $CANDIDATE_VERSION" >&2
    exit 1
fi

case "$(uname -m)" in
    x86_64|amd64) archive_arch="x86_64" ;;
    arm64|aarch64) archive_arch="arm64" ;;
    *) echo "Unsupported lifecycle-cert architecture: $(uname -m)" >&2; exit 2 ;;
esac

root="${NEXUS_RELEASE_LIFECYCLE_WORKDIR:-$(mktemp -d)}"
preserve_root=false
if [ -n "${NEXUS_RELEASE_LIFECYCLE_WORKDIR:-}" ]; then
    preserve_root=true
fi
serve="$root/serve"
home="$root/home"
prefix="$root/prefix"
bad_prefix="$root/bad-prefix"
mismatch_prefix="$root/mismatch-prefix"
old_bin="$root/old/nexus"
mkdir -p "$serve" "$home" "$prefix/bin" "$bad_prefix/bin" "$mismatch_prefix/bin" "$(dirname "$old_bin")"

server_pid=""
cleanup() {
    if [ -n "${server_pid:-}" ] && kill -0 "$server_pid" 2>/dev/null; then
        kill "$server_pid" 2>/dev/null || true
        wait "$server_pid" 2>/dev/null || true
    fi
    if [ "$preserve_root" != true ]; then
        rm -rf "$root"
    fi
}
trap cleanup EXIT

build_date="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
go build -trimpath -ldflags="-s -w -X github.com/IAPro-Community/nexus/internal/buildinfo.Version=$OLD_VERSION -X github.com/IAPro-Community/nexus/internal/buildinfo.Commit=lifecycle-old -X github.com/IAPro-Community/nexus/internal/buildinfo.BuildDate=$build_date -X github.com/IAPro-Community/nexus/internal/update.ProductionTrustRootHex=$NEXUS_UPDATE_PUBLIC_KEY"     -o "$old_bin" ./cmd/nexus

old_reported="$("$old_bin" version --json | python3 -c 'import json,sys; print(json.load(sys.stdin)["version"])')"
test "$old_reported" = "$OLD_VERSION"

port="$(python3 - <<'PY'
import socket
s = socket.socket()
s.bind(("127.0.0.1", 0))
print(s.getsockname()[1])
s.close()
PY
)"
base="http://127.0.0.1:$port"

package_release() {
    local version="$1"
    local binary="$2"
    local release_dir="$serve/v$version"
    local stage="$root/stage-$version"
    local archive="$release_dir/nexus_Linux_${archive_arch}.tar.gz"

    rm -rf "$release_dir" "$stage"
    mkdir -p "$release_dir" "$stage"
    install -m 0755 "$binary" "$stage/nexus"
    tar -czf "$archive" -C "$stage" nexus

    go run ./scripts/sign-update-manifest         --dist "$release_dir"         --version "v$version"         --key-id "$NEXUS_UPDATE_KEY_ID"         --private-key "$NEXUS_UPDATE_PRIVATE_KEY"         --public-key "$NEXUS_UPDATE_PUBLIC_KEY"         --base-url "$base/v$version"
}

package_release "$OLD_VERSION" "$old_bin"
package_release "$CANDIDATE_VERSION" "$NEXUS_BIN"
package_release "$MISMATCH_VERSION" "$old_bin"

python3 -m http.server "$port" --bind 127.0.0.1 --directory "$serve" >"$root/http.log" 2>&1 &
server_pid=$!
for _ in $(seq 1 30); do
    if curl -fsS "$base/v$CANDIDATE_VERSION/update-manifest.json" >/dev/null; then
        break
    fi
    if ! kill -0 "$server_pid" 2>/dev/null; then
        echo "local release server exited" >&2
        cat "$root/http.log" >&2 || true
        exit 1
    fi
    sleep 1
done
curl -fsS "$base/v$CANDIDATE_VERSION/update-manifest.json" >/dev/null

install_version() {
    local version="$1"
    local install_dir="$2"
    HOME="$home"     NEXUS_INSTALL_DIR="$install_dir"     NEXUS_RELEASE_BASE_URL="$base"     bash ./install.sh --version="v$version" --no-desktop --no-path
}

echo "== signed install: $OLD_VERSION =="
install_version "$OLD_VERSION" "$prefix/bin"
installed_old="$("$prefix/bin/nexus" version --json | python3 -c 'import json,sys; print(json.load(sys.stdin)["version"])')"
test "$installed_old" = "$OLD_VERSION"

echo "== signed upgrade: $OLD_VERSION -> $CANDIDATE_VERSION =="
install_version "$CANDIDATE_VERSION" "$prefix/bin"
installed_candidate="$("$prefix/bin/nexus" version --json | python3 -c 'import json,sys; print(json.load(sys.stdin)["version"])')"
test "$installed_candidate" = "$CANDIDATE_VERSION"

echo "== signed self-update: $OLD_VERSION -> $CANDIDATE_VERSION =="
update_prefix="$root/update-prefix"
mkdir -p "$update_prefix/bin"
install -m 0755 "$old_bin" "$update_prefix/bin/nexus"
HOME="$home" \
NEXUS_UPDATE_REGISTRY_URL="$base/v$CANDIDATE_VERSION/update-manifest.json" \
NEXUS_INSTALL_METHOD=STANDALONE \
"$update_prefix/bin/nexus" update
updated_version="$("$update_prefix/bin/nexus" version --json | python3 -c 'import json,sys; print(json.load(sys.stdin)["version"])')"
test "$updated_version" = "$CANDIDATE_VERSION"

echo "== signed wrong-version artifact must fail =="
if install_version "$MISMATCH_VERSION" "$mismatch_prefix/bin"; then
    echo "Installer accepted a correctly signed archive containing the wrong embedded version" >&2
    exit 1
fi
test ! -e "$mismatch_prefix/bin/nexus"

echo "== tampered signed artifact must fail =="
candidate_archive="$serve/v$CANDIDATE_VERSION/nexus_Linux_${archive_arch}.tar.gz"
printf 'tamper' >> "$candidate_archive"
if install_version "$CANDIDATE_VERSION" "$bad_prefix/bin"; then
    echo "Installer accepted an artifact whose bytes no longer match the signed manifest" >&2
    exit 1
fi
test ! -e "$bad_prefix/bin/nexus"

echo "== uninstall preserves data by default =="
mkdir -p "$home/.local/share/nexus"
printf 'preserve-me\n' > "$home/.local/share/nexus/lifecycle-sentinel"
HOME="$home" PREFIX="$prefix" bash ./uninstall.sh
test ! -e "$prefix/bin/nexus"
test -f "$home/.local/share/nexus/lifecycle-sentinel"

echo "== uninstall is idempotent =="
HOME="$home" PREFIX="$prefix" bash ./uninstall.sh

echo "== explicit purge removes Nexus-owned data =="
HOME="$home" PREFIX="$prefix" bash ./uninstall.sh --purge-data
test ! -e "$home/.local/share/nexus"

cat > "$root/result.txt" <<EOF
signed install: PASS
signed upgrade: PASS
signed self-update: PASS
embedded version identity: PASS
tamper rejection: PASS
uninstall preserve-data: PASS
uninstall idempotency: PASS
explicit purge: PASS
candidate version: $CANDIDATE_VERSION
EOF

cat "$root/result.txt"
echo "certify-release-lifecycle PASS"

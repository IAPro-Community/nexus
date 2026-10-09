# Build and Release Process

## Version Contract

IAPro Nexus uses a strict version contract across multiple files:

| File | Field | Description |
|------|-------|-------------|
| `VERSION` | Content | Canonical version source |
| `web/package.json` | `version` | Web frontend version |
| `cmd/nexus-desktop/wails.json` | `info.productVersion` | Desktop app version |
| `internal/buildinfo` | `Version` | Build-time injection |

### Version Contract Enforcement

The `make version-check` target validates consistency:

```bash
make version-check
# Version consistency PASS: 0.5.0-beta.23
```

The contract verifier (`scripts/verify-version-contract.go`) checks:
- `VERSION` file exists and is non-empty
- `web/package.json` version matches
- `cmd/nexus-desktop/wails.json` version matches
- Git tag (if present) matches `v{VERSION}`

## Build Commands

### Development Build

```bash
make build
# Builds binary with version injection
# Output: ./nexus
```

### Frontend Build

```bash
make web
# Builds web assets for embedding
```

### Desktop Build (Wails)

```bash
make build-desktop
# Builds desktop app with production tags
```

### Full Build (Frontend + Backend)

```bash
make all  # or just: make build
```

## Version Injection

Version information is injected at build time via `-ldflags`:

```
-X github.com/IAPro-Community/nexus/internal/buildinfo.Version={VERSION}
-X github.com/IAPro-Community/nexus/internal/buildinfo.Commit={COMMIT}
-X github.com/IAPro-Community/nexus/internal/buildinfo.BuildDate={BUILDDATE}
```

## Quality Gates

Run before any release:

```bash
make quality
# Runs: format-check, lint-frontend, lint-styles, typecheck, lint-go, test-go, test-frontend, version-check
```

Full quality with race detection and security:

```bash
make quality-full
```

## Release Process

### 1. Pre-release Checklist

- [ ] All quality gates pass (`make quality`)
- [ ] Version contract verified (`make version-check`)
- [ ] Changelog updated
- [ ] Tests passing (`make test-go`, `make test-frontend`)
- [ ] No security vulnerabilities (`make security`)

### 2. Update Version

Edit `VERSION` file:

```bash
echo "0.5.0" > VERSION
```

Update `web/package.json` and `cmd/nexus-desktop/wails.json` to match.

### 3. Build and Verify

```bash
make release-dry-run
```

### 4. Create Git Tag

```bash
git tag v0.5.0
git push origin v0.5.0
```

### 5. Build Release Artifacts

```bash
make build
# Binary: ./nexus
```

## Architecture Decision Records

Key decisions are documented in `DEV/ARCH/ADR-*.md`:
- ADR-001: Atomic Backup Strategy
- ADR-002: Manifest Signing and Verification
- ADR-003: Platform Detection Heuristics
- ADR-004: Streaming Download with Archive Extraction
- ADR-005: Rollback and Receipt Strategy

## Troubleshooting

### Version Mismatch

If `make version-check` fails:

1. Check `VERSION` file
2. Update `web/package.json`
3. Update `cmd/nexus-desktop/wails.json`
4. Re-run `make version-check`

### Build Fails

1. Ensure Go 1.21+ is installed
2. Ensure Node.js 20+ is active (`.nvmrc` pins Node 22; the web build uses
   `Array.prototype.toSorted` which does not exist on Node 18):
   `nvm use` (or `export PATH="$HOME/.nvm/versions/node/v22.17.0/bin:$PATH"`)
3. Run `go mod tidy`
4. Check for dependency issues

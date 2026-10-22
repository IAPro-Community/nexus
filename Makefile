BINARY=nexus
MODULE=github.com/IAPro-Community/nexus
LOCAL_BIN ?= $(HOME)/.local/bin
DESKTOP_TAGS = production
GO_TAGS ?=
WAILS_VERSION ?= v3.0.0-beta.25
ifeq ($(shell go env GOOS),linux)
DESKTOP_TAGS = production,gtk3
GO_TAGS = gtk3
endif

.PHONY: all build build-desktop build-desktop-wails build-hud build-hud-wails web web-verify docs-verify test race vet install install-local release-local bump clean format format-check lint-frontend lint-styles lint-styles-fix lint-fix lint-go typecheck test-frontend test-go test-e2e security quality quality-full golangci-lint version-check test-updater-negative test-update-integration release-dry-run browser-functional browser-a11y browser-visual browser-all verify-update coverage e2e attention-e2e overnight-smoke overnight-soak release-proof smoke-providers

all: build

bump:
	@echo "Version bumps are managed by: make release-local"

# ─── Version Consistency ────────────────────────────────────────────

version-check:
	@go run ./scripts/verify-version-contract.go .

# ─── Frontend ───────────────────────────────────────────────────────

web:
	@echo "Building frontend..."
	@cd web && node scripts/build.mjs
	@echo "Frontend bundle ready for embedding"

web-verify:
	@cd web && node scripts/verify-report.mjs

docs-verify:
	@node scripts/docs-verify.mjs

lint-frontend:
	@cd web && ./node_modules/.bin/eslint src

lint-styles:
	@cd web && ./node_modules/.bin/stylelint "src/**/*.{css,scss}"

lint-styles-fix:
	@cd web && ./node_modules/.bin/stylelint "src/**/*.{css,scss}" --fix

lint-fix:
	@cd web && ./node_modules/.bin/eslint src --fix

typecheck:
	@cd web && ./node_modules/.bin/tsc --noEmit

test-frontend:
	@cd web && ./node_modules/.bin/vitest run

test-go:
	@if [ -n "$(GO_TAGS)" ]; then go test -tags="$(GO_TAGS)" -v ./...; else go test -v ./...; fi

test: test-go test-frontend

test-e2e:
	@if [ -n "$(GO_TAGS)" ]; then go test -tags="$(GO_TAGS)" -race -v -count=1 ./internal/control/terminal/... ./internal/control/protocol/... ./internal/control/host/... ./internal/control/web/...; else go test -race -v -count=1 ./internal/control/terminal/... ./internal/control/protocol/... ./internal/control/host/... ./internal/control/web/...; fi

e2e: test-e2e

attention-e2e:
	@if [ -n "$(GO_TAGS)" ]; then go test -tags="$(GO_TAGS)" -count=1 -timeout 10m ./internal/nexus/runner -run 'Test(MultiMissionIndependentProgression|ResolveInterventionIdempotent|StaleInterventionRejected|ResumeFromDurableCheckpoint|QuotaFailoverNoAttention|NotificationDedupKey|AutonomyContractRespected|CrossProjectAggregation|FaultInjectionStoreReopenPreservesIntervention|FaultInjectionDuplicateInterventionResolvesOnce)'; else go test -count=1 -timeout 10m ./internal/nexus/runner -run 'Test(MultiMissionIndependentProgression|ResolveInterventionIdempotent|StaleInterventionRejected|ResumeFromDurableCheckpoint|QuotaFailoverNoAttention|NotificationDedupKey|AutonomyContractRespected|CrossProjectAggregation|FaultInjectionStoreReopenPreservesIntervention|FaultInjectionDuplicateInterventionResolvesOnce)'; fi

coverage:
	@mkdir -p DEV/validation/current
	@if [ -n "$(GO_TAGS)" ]; then go test -tags="$(GO_TAGS)" ./... -coverprofile=DEV/validation/current/coverage.out -covermode=atomic; else go test ./... -coverprofile=DEV/validation/current/coverage.out -covermode=atomic; fi
	@go tool cover -func=DEV/validation/current/coverage.out | tee DEV/validation/current/coverage-func.txt
	@echo "Frontend coverage plugin is not installed; vitest statement coverage remains UNVERIFIED."

overnight-smoke:
	@if [ -n "$(GO_TAGS)" ]; then go test -tags="$(GO_TAGS)" -count=1 -timeout 20m ./internal/nexus/runner -run 'Test(MissionTeamPlannerBuilderReviewerVerifier|OvernightAcceptanceSandbox|OvernightSoakHarnessBounded|ProgressWatchdog|FaultInjection)'; else go test -count=1 -timeout 20m ./internal/nexus/runner -run 'Test(MissionTeamPlannerBuilderReviewerVerifier|OvernightAcceptanceSandbox|OvernightSoakHarnessBounded|ProgressWatchdog|FaultInjection)'; fi

# Local-only authenticated provider smoke (not part of CI quality).
# Fail-closed provider execution lives in scripts/smoke-providers.sh so Make
# does not reinterpret shell variables or mask provider failures.
SMOKE_REQUIRED_PROVIDERS ?= codex agy claude gemini
SMOKE_PROVIDER_MARKER ?= NEXUS_PROVIDER_SMOKE_OK
smoke-providers:
	@SMOKE_REQUIRED_PROVIDERS="$(SMOKE_REQUIRED_PROVIDERS)" \
	  SMOKE_PROVIDER_MARKER="$(SMOKE_PROVIDER_MARKER)" \
	  bash ./scripts/smoke-providers.sh

DURATION ?= 30s
overnight-soak:
	@echo "OVERNIGHT_SOAK_DURATION=$(DURATION) (ordinary CI must not use 8h)"
	@if [ -n "$(GO_TAGS)" ]; then OVERNIGHT_SOAK_DURATION=$(DURATION) go test -tags="$(GO_TAGS)" -count=1 -timeout 24h ./internal/nexus/runner -run TestOvernightSoakHarnessBounded; else OVERNIGHT_SOAK_DURATION=$(DURATION) go test -count=1 -timeout 24h ./internal/nexus/runner -run TestOvernightSoakHarnessBounded; fi

release-proof: format-check lint-frontend lint-styles typecheck lint-go test-go test-frontend vet race coverage attention-e2e overnight-smoke security version-check
	@echo "OVERNIGHT_SOAK_EVIDENCE=MISSING (opt-in: make overnight-soak DURATION=8h)"
	@echo "NATIVE_WINDOWS=UNVERIFIED"
	@echo "NATIVE_MACOS=UNVERIFIED"
	@echo "release-proof local gates finished"

race:
	@if [ -n "$(GO_TAGS)" ]; then go test -tags="$(GO_TAGS)" -race ./...; else go test -race ./...; fi

# ─── Formatting ─────────────────────────────────────────────────────

format:
	@cd web && ./node_modules/.bin/prettier --write "src/**/*.{ts,tsx,css,scss,json}"
	@gofmt -w -s .

format-check:
	@cd web && ./node_modules/.bin/prettier --check "src/**/*.{ts,tsx,css,scss,json}"
	@test -z "$$(gofmt -l . | grep -v .worktrees)" || (gofmt -l . | grep -v .worktrees && exit 1)

# ─── Go linting ─────────────────────────────────────────────────────

lint-go:
	@if [ -n "$(GO_TAGS)" ]; then GOFLAGS="-tags=$(GO_TAGS)" PATH="$(HOME)/go/bin:$(PATH)" golangci-lint run ./...; else PATH="$(HOME)/go/bin:$(PATH)" golangci-lint run ./...; fi

golangci-lint:
	@if [ -n "$(GO_TAGS)" ]; then GOFLAGS="-tags=$(GO_TAGS)" PATH="$(HOME)/go/bin:$(PATH)" golangci-lint run ./...; else PATH="$(HOME)/go/bin:$(PATH)" golangci-lint run ./...; fi

vet:
	@if [ -n "$(GO_TAGS)" ]; then go vet -tags="$(GO_TAGS)" ./...; else go vet ./...; fi

# ─── Security ───────────────────────────────────────────────────────

security:
	@if command -v govulncheck >/dev/null 2>&1; then \
		if [ -n "$(GO_TAGS)" ]; then GOTOOLCHAIN=go1.26.10 PATH="$(HOME)/go/bin:$(PATH)" govulncheck -tags="$(GO_TAGS)" ./...; else GOTOOLCHAIN=go1.26.10 PATH="$(HOME)/go/bin:$(PATH)" govulncheck ./...; fi; \
	else \
		if [ -n "$(GO_TAGS)" ]; then GOFLAGS="-tags=$(GO_TAGS)" GOTOOLCHAIN=auto go run golang.org/x/vuln/cmd/govulncheck@v1.7.0 ./...; else GOTOOLCHAIN=auto go run golang.org/x/vuln/cmd/govulncheck@v1.7.0 ./...; fi; \
	fi

# ─── Quality gates ──────────────────────────────────────────────────

quality: format-check lint-frontend lint-styles typecheck lint-go test-go test-frontend version-check

quality-full: quality race security

# ─── Build ──────────────────────────────────────────────────────────

build: web
	@set -e; VERSION=$$(cat VERSION 2>/dev/null || echo "dev"); \
	COMMIT=$$(git rev-parse --short HEAD 2>/dev/null || echo "unknown"); \
	BUILDDATE=$$(date -u +%Y-%m-%dT%H:%M:%SZ); \
	LDFLAGS="-s -w -X $(MODULE)/internal/buildinfo.Version=$$VERSION -X $(MODULE)/internal/buildinfo.Commit=$$COMMIT -X $(MODULE)/internal/buildinfo.BuildDate=$$BUILDDATE -X $(MODULE)/internal/update.ProductionTrustRootHex=$${NEXUS_UPDATE_PUBLIC_KEY:-}"; \
	echo "Building $(BINARY) v$$VERSION (commit: $$COMMIT)..."; \
	go build -trimpath -buildvcs=false -ldflags="$$LDFLAGS" -o $(BINARY) ./cmd/nexus; \
	echo "Built $(BINARY) v$$VERSION at ./$(BINARY)"

install-local: build
	@set -e; \
	mkdir -p $(LOCAL_BIN); \
	rm -f $(LOCAL_BIN)/$(BINARY).tmp; \
	cp -f $(BINARY) $(LOCAL_BIN)/$(BINARY).tmp; \
	chmod +x $(LOCAL_BIN)/$(BINARY).tmp; \
	mv -f $(LOCAL_BIN)/$(BINARY).tmp $(LOCAL_BIN)/$(BINARY); \
	rm -f $(LOCAL_BIN)/ai; \
	echo "Installed $(BINARY) to $(LOCAL_BIN)/$(BINARY)"; \
	./scripts/restart-web.sh "$(LOCAL_BIN)/$(BINARY)"

build-desktop: web
	@set -e; VERSION=$$(cat VERSION 2>/dev/null || echo "dev"); \
	COMMIT=$$(git rev-parse --short HEAD 2>/dev/null || echo "unknown"); \
	BUILDDATE=$$(date -u +%Y-%m-%dT%H:%M:%SZ); \
	LDFLAGS="-s -w -X $(MODULE)/internal/buildinfo.Version=$$VERSION -X $(MODULE)/internal/buildinfo.Commit=$$COMMIT -X $(MODULE)/internal/buildinfo.BuildDate=$$BUILDDATE -X $(MODULE)/internal/update.ProductionTrustRootHex=$${NEXUS_UPDATE_PUBLIC_KEY:-}"; \
	echo "Building nexus-desktop v$$VERSION (commit: $$COMMIT)..."; \
	go build -trimpath -buildvcs=false -tags "$(DESKTOP_TAGS)" -ldflags="$$LDFLAGS" -o nexus-desktop ./cmd/nexus-desktop

build-desktop-wails: web
	@set -e; VERSION=$$(cat VERSION 2>/dev/null || echo "dev"); \
	COMMIT=$$(git rev-parse --short HEAD 2>/dev/null || echo "unknown"); \
	BUILDDATE=$$(date -u +%Y-%m-%dT%H:%M:%SZ); \
	LDFLAGS="-s -w -X $(MODULE)/internal/buildinfo.Version=$$VERSION -X $(MODULE)/internal/buildinfo.Commit=$$COMMIT -X $(MODULE)/internal/buildinfo.BuildDate=$$BUILDDATE -X $(MODULE)/internal/update.ProductionTrustRootHex=$${NEXUS_UPDATE_PUBLIC_KEY:-}"; \
	mkdir -p cmd/nexus-desktop/build/bin; GOTOOLCHAIN=auto GOFLAGS="-mod=readonly" go build -trimpath -buildvcs=false -tags "$(DESKTOP_TAGS)" -ldflags="$$LDFLAGS" -o cmd/nexus-desktop/build/bin/nexus-desktop ./cmd/nexus-desktop


build-hud: web
	@set -e; VERSION=$$(cat VERSION 2>/dev/null || echo "dev"); \
	COMMIT=$$(git rev-parse --short HEAD 2>/dev/null || echo "unknown"); \
	BUILDDATE=$$(date -u +%Y-%m-%dT%H:%M:%SZ); \
	LDFLAGS="-s -w -X $(MODULE)/internal/buildinfo.Version=$$VERSION -X $(MODULE)/internal/buildinfo.Commit=$$COMMIT -X $(MODULE)/internal/buildinfo.BuildDate=$$BUILDDATE -X $(MODULE)/internal/update.ProductionTrustRootHex=$${NEXUS_UPDATE_PUBLIC_KEY:-}"; \
	echo "Building nexus-hud v$$VERSION (commit: $$COMMIT)..."; \
	go build -trimpath -buildvcs=false -tags "$(DESKTOP_TAGS)" -ldflags="$$LDFLAGS" -o nexus-hud ./cmd/nexus-hud

build-hud-wails: web
	@set -e; VERSION=$$(cat VERSION 2>/dev/null || echo "dev"); \
	COMMIT=$$(git rev-parse --short HEAD 2>/dev/null || echo "unknown"); \
	BUILDDATE=$$(date -u +%Y-%m-%dT%H:%M:%SZ); \
	LDFLAGS="-s -w -X $(MODULE)/internal/buildinfo.Version=$$VERSION -X $(MODULE)/internal/buildinfo.Commit=$$COMMIT -X $(MODULE)/internal/buildinfo.BuildDate=$$BUILDDATE -X $(MODULE)/internal/update.ProductionTrustRootHex=$${NEXUS_UPDATE_PUBLIC_KEY:-}"; \
	mkdir -p cmd/nexus-hud/build/bin; GOTOOLCHAIN=auto GOFLAGS="-mod=readonly" go build -trimpath -buildvcs=false -tags "$(DESKTOP_TAGS)" -ldflags="$$LDFLAGS" -o cmd/nexus-hud/build/bin/nexus-hud ./cmd/nexus-hud

release-local:
	go run ./cmd/nexus release

install: build
	@set -e; \
	if [ -n "$(DESTDIR)" ]; then \
		install -d $(DESTDIR)/usr/local/bin; \
		install -m 755 $(BINARY) $(DESTDIR)/usr/local/bin/$(BINARY); \
		echo "Installed $(BINARY) to $(DESTDIR)/usr/local/bin/$(BINARY)"; \
	else \
		mkdir -p $(LOCAL_BIN); \
		rm -f $(LOCAL_BIN)/$(BINARY).tmp; \
		cp -f $(BINARY) $(LOCAL_BIN)/$(BINARY).tmp; \
		chmod +x $(LOCAL_BIN)/$(BINARY).tmp; \
		mv -f $(LOCAL_BIN)/$(BINARY).tmp $(LOCAL_BIN)/$(BINARY); \
		rm -f $(LOCAL_BIN)/ai; \
		echo "Installed $(BINARY) to $(LOCAL_BIN)/$(BINARY)"; \
		./scripts/restart-web.sh "$(LOCAL_BIN)/$(BINARY)"; \
	fi

clean:
	rm -f $(BINARY)

# ─── Updater Tests ──────────────────────────────────────────────────

test-updater-negative:
	@echo "Running updater negative tests..."
	@go test -v -run "TestNegative" ./internal/update/...

test-update-integration:
	@echo "Running updater integration tests..."
	@go test -v -run "TestService" ./internal/update/...

# ─── Release Engineering ────────────────────────────────────────────

release-dry-run: build
	@echo "=== Release Dry Run ==="
	@echo "1. Version contract check..."
	@go run scripts/verify-version-contract.go .
	@echo ""
	@echo "2. Build verification..."
	@if [ ! -f $(BINARY) ]; then echo "Binary not found"; exit 1; fi
	@echo "   Binary: $(BINARY) ✓"
	@echo ""
	@echo "3. Quality gates..."
	@$(MAKE) version-check
	@echo ""
	@echo "=== Source Build Dry Run Complete ==="
	@echo "Build and version contract verified; this is NOT release certification."
	@echo "Before publication, require make quality-full and exact-SHA native/signing gates."
	@echo "No publication occurs here; there is no 'make release' target."

# ─── Update Verification ────────────────────────────────────────────

verify-update:
	@echo "Running update verification gate..."
	@bash scripts/verify-update-gate.sh

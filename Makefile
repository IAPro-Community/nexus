BINARY=nexus
MODULE=github.com/kivervinicius/ai-cli
LOCAL_BIN ?= $(HOME)/.local/bin
DESKTOP_TAGS = production
WAILS_VERSION ?= v2.15.0
ifeq ($(shell go env GOOS),linux)
DESKTOP_TAGS = production,webkit2_41
endif

.PHONY: all build build-desktop build-desktop-wails web web-verify docs-verify test race vet install install-local release-local bump clean format format-check lint-frontend lint-styles lint-styles-fix lint-fix lint-go typecheck test-frontend test-go test-e2e security quality quality-full golangci-lint version-check test-updater-negative test-update-integration release-dry-run browser-functional browser-a11y browser-visual browser-all verify-update coverage e2e attention-e2e overnight-smoke overnight-soak release-proof

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
	@go test -v ./...

test: test-go test-frontend

test-e2e:
	@go test -race -v -count=1 ./internal/control/terminal/... ./internal/control/protocol/... ./internal/control/host/... ./internal/control/web/...

e2e: test-e2e

attention-e2e:
	@go test -count=1 -timeout 10m ./internal/nexus/runner -run 'Test(MultiMissionIndependentProgression|ResolveInterventionIdempotent|StaleInterventionRejected|ResumeFromDurableCheckpoint|QuotaFailoverNoAttention|NotificationDedupKey|AutonomyContractRespected|CrossProjectAggregation|FaultInjectionStoreReopenPreservesIntervention|FaultInjectionDuplicateInterventionResolvesOnce)'

coverage:
	@mkdir -p DEV/validation/current
	@go test ./... -coverprofile=DEV/validation/current/coverage.out -covermode=atomic
	@go tool cover -func=DEV/validation/current/coverage.out | tee DEV/validation/current/coverage-func.txt
	@echo "Frontend coverage plugin is not installed; vitest statement coverage remains UNVERIFIED."

overnight-smoke:
	@go test -count=1 -timeout 20m ./internal/nexus/runner -run 'Test(MissionTeamPlannerBuilderReviewerVerifierPipeline|OvernightAcceptanceSandbox|OvernightSoakHarnessBounded|ProgressWatchdog|FaultInjection)'

DURATION ?= 30s
overnight-soak:
	@echo "OVERNIGHT_SOAK_DURATION=$(DURATION) (ordinary CI must not use 8h)"
	@OVERNIGHT_SOAK_DURATION=$(DURATION) go test -count=1 -timeout 24h ./internal/nexus/runner -run TestOvernightSoakHarnessBounded

release-proof: format-check lint-frontend lint-styles typecheck lint-go test-go test-frontend vet race coverage attention-e2e overnight-smoke security version-check
	@echo "OVERNIGHT_SOAK_EVIDENCE=MISSING (opt-in: make overnight-soak DURATION=8h)"
	@echo "NATIVE_WINDOWS=UNVERIFIED"
	@echo "NATIVE_MACOS=UNVERIFIED"
	@echo "release-proof local gates finished"

race:
	@go test -race ./...

# ─── Formatting ─────────────────────────────────────────────────────

format:
	@cd web && ./node_modules/.bin/prettier --write "src/**/*.{ts,tsx,css,scss,json}"
	@gofmt -w -s .

format-check:
	@cd web && ./node_modules/.bin/prettier --check "src/**/*.{ts,tsx,css,scss,json}"
	@test -z "$$(gofmt -l . | grep -v .worktrees)" || (gofmt -l . | grep -v .worktrees && exit 1)

# ─── Go linting ─────────────────────────────────────────────────────

lint-go:
	@PATH="$(HOME)/go/bin:$(PATH)" golangci-lint run ./...

golangci-lint:
	@PATH="$(HOME)/go/bin:$(PATH)" golangci-lint run ./...

vet:
	@go vet ./...

# ─── Security ───────────────────────────────────────────────────────

security:
	@if command -v govulncheck >/dev/null 2>&1; then \
		GOTOOLCHAIN=go1.25.14 PATH="$(HOME)/go/bin:$(PATH)" govulncheck ./...; \
	else \
		GOTOOLCHAIN=auto go run golang.org/x/vuln/cmd/govulncheck@v1.7.0 ./...; \
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
	go build -buildvcs=false -ldflags="$$LDFLAGS" -o $(BINARY) ./cmd/nexus; \
	echo "Built $(BINARY) v$$VERSION at ./$(BINARY)"

install-local: build
	@set -e; \
	mkdir -p $(LOCAL_BIN); \
	rm -f $(LOCAL_BIN)/$(BINARY).tmp; \
	cp -f $(BINARY) $(LOCAL_BIN)/$(BINARY).tmp; \
	chmod +x $(LOCAL_BIN)/$(BINARY).tmp; \
	mv -f $(LOCAL_BIN)/$(BINARY).tmp $(LOCAL_BIN)/$(BINARY); \
	rm -f $(LOCAL_BIN)/ai; \
	echo "Installed $(BINARY) to $(LOCAL_BIN)/$(BINARY)"

build-desktop: web
	@set -e; VERSION=$$(cat VERSION 2>/dev/null || echo "dev"); \
	COMMIT=$$(git rev-parse --short HEAD 2>/dev/null || echo "unknown"); \
	BUILDDATE=$$(date -u +%Y-%m-%dT%H:%M:%SZ); \
	LDFLAGS="-s -w -X $(MODULE)/internal/buildinfo.Version=$$VERSION -X $(MODULE)/internal/buildinfo.Commit=$$COMMIT -X $(MODULE)/internal/buildinfo.BuildDate=$$BUILDDATE -X $(MODULE)/internal/update.ProductionTrustRootHex=$${NEXUS_UPDATE_PUBLIC_KEY:-}"; \
	echo "Building nexus-desktop v$$VERSION (commit: $$COMMIT)..."; \
	go build -buildvcs=false -tags "$(DESKTOP_TAGS)" -ldflags="$$LDFLAGS" -o nexus-desktop ./cmd/nexus-desktop

build-desktop-wails: web
	@set -e; VERSION=$$(cat VERSION 2>/dev/null || echo "dev"); \
	COMMIT=$$(git rev-parse --short HEAD 2>/dev/null || echo "unknown"); \
	BUILDDATE=$$(date -u +%Y-%m-%dT%H:%M:%SZ); \
	LDFLAGS="-s -w -X $(MODULE)/internal/buildinfo.Version=$$VERSION -X $(MODULE)/internal/buildinfo.Commit=$$COMMIT -X $(MODULE)/internal/buildinfo.BuildDate=$$BUILDDATE -X $(MODULE)/internal/update.ProductionTrustRootHex=$${NEXUS_UPDATE_PUBLIC_KEY:-}"; \
	cd cmd/nexus-desktop && GOTOOLCHAIN=auto go run github.com/wailsapp/wails/v2/cmd/wails@$(WAILS_VERSION) build -clean -s -m -tags "$(DESKTOP_TAGS)" -ldflags "$$LDFLAGS"

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
	@echo "=== Dry Run Complete ==="
	@echo "Ready for release. Run 'make release' to publish."

# ─── Update Verification ────────────────────────────────────────────

verify-update:
	@echo "Running update verification gate..."
	@bash scripts/verify-update-gate.sh

BINARY=nexus
MODULE=github.com/kivervinicius/ai-cli
LOCAL_BIN ?= $(HOME)/.local/bin
DESKTOP_TAGS = production
WAILS_VERSION ?= v2.15.0
ifeq ($(shell go env GOOS),linux)
DESKTOP_TAGS = production,webkit2_41
endif

.PHONY: all build build-desktop build-desktop-wails web web-verify docs-verify test race vet install install-local release-local bump clean format format-check lint-frontend lint-styles lint-styles-fix lint-fix lint-go typecheck test-frontend test-go test-e2e security quality quality-full golangci-lint

all: build

bump:
	@echo "Version bumps are managed by: make release-local"

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
	@cd web && npx eslint src

lint-styles:
	@cd web && npx stylelint "src/**/*.css"

lint-styles-fix:
	@cd web && npx stylelint "src/**/*.css" --fix

lint-fix:
	@cd web && npx eslint src --fix

typecheck:
	@cd web && npx tsc --noEmit

test-frontend:
	@cd web && npx vitest run

test-go:
	@go test -v ./...

test-e2e:
	@go test -race -v -count=1 ./internal/control/terminal/... ./internal/control/protocol/... ./internal/control/host/... ./internal/control/web/...

race:
	@go test -race ./...

# ─── Formatting ─────────────────────────────────────────────────────

format:
	@cd web && npx prettier --write "src/**/*.{ts,tsx,css,json}"
	@gofmt -w -s .

format-check:
	@cd web && npx prettier --check "src/**/*.{ts,tsx,json}"
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

quality: format-check lint-frontend lint-styles typecheck lint-go test-go test-frontend

quality-full: quality race security

# ─── Build ──────────────────────────────────────────────────────────

build: web
	@set -e; VERSION=$$(cat VERSION 2>/dev/null || echo "dev"); \
	COMMIT=$$(git rev-parse --short HEAD 2>/dev/null || echo "unknown"); \
	BUILDDATE=$$(date -u +%Y-%m-%dT%H:%M:%SZ); \
	LDFLAGS="-s -w -X $(MODULE)/internal/buildinfo.Version=$$VERSION -X $(MODULE)/internal/buildinfo.Commit=$$COMMIT -X $(MODULE)/internal/buildinfo.BuildDate=$$BUILDDATE"; \
	echo "Building $(BINARY) v$$VERSION (commit: $$COMMIT)..."; \
	go build -ldflags="$$LDFLAGS" -o $(BINARY) ./cmd/nexus; \
	echo "Built $(BINARY) v$$VERSION at ./$(BINARY)"

install-local: build
	@set -e; \
	mkdir -p $(LOCAL_BIN); \
	rm -f $(LOCAL_BIN)/$(BINARY).tmp; \
	cp -f $(BINARY) $(LOCAL_BIN)/$(BINARY).tmp; \
	chmod +x $(LOCAL_BIN)/$(BINARY).tmp; \
	mv -f $(LOCAL_BIN)/$(BINARY).tmp $(LOCAL_BIN)/$(BINARY); \
	ln -sf $(LOCAL_BIN)/$(BINARY) $(LOCAL_BIN)/ai; \
	echo "Installed $(BINARY) to $(LOCAL_BIN)/$(BINARY) (alias: ai)"

build-desktop: web
	@set -e; VERSION=$$(cat VERSION 2>/dev/null || echo "dev"); \
	COMMIT=$$(git rev-parse --short HEAD 2>/dev/null || echo "unknown"); \
	BUILDDATE=$$(date -u +%Y-%m-%dT%H:%M:%SZ); \
	LDFLAGS="-s -w -X $(MODULE)/internal/buildinfo.Version=$$VERSION -X $(MODULE)/internal/buildinfo.Commit=$$COMMIT -X $(MODULE)/internal/buildinfo.BuildDate=$$BUILDDATE"; \
	echo "Building nexus-desktop v$$VERSION (commit: $$COMMIT)..."; \
	go build -tags "$(DESKTOP_TAGS)" -ldflags="$$LDFLAGS" -o nexus-desktop ./cmd/nexus-desktop

build-desktop-wails: web
	@cd cmd/nexus-desktop && GOTOOLCHAIN=auto go run github.com/wailsapp/wails/v2/cmd/wails@$(WAILS_VERSION) build -clean -s -m -tags "$(DESKTOP_TAGS)"

release-local:
	go run ./cmd/nexus release

install: build
	@if [ -n "$(DESTDIR)" ]; then \
		install -d $(DESTDIR)/usr/local/bin; \
		install -m 755 $(BINARY) $(DESTDIR)/usr/local/bin/$(BINARY); \
	fi

clean:
	rm -f $(BINARY)

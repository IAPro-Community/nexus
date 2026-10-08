package contextsnapshot

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/security"
)

const (
	ScannerVersion         = "project-intelligence-v1"
	MaxDiscoveryDepth      = 6
	MaxDiscoveryCandidates = 512
	MaxDiscoveryFileBytes  = 256 * 1024
	MaxDiscoveryTotalBytes = 4 * 1024 * 1024
	MaxProjectFacts        = 500
)

type FactValueType string

const (
	FactString FactValueType = "STRING"
	FactList   FactValueType = "LIST"
	FactMap    FactValueType = "MAP"
	FactBool   FactValueType = "BOOL"
)

type FactBasis string

const (
	FactObserved FactBasis = "OBSERVED"
	FactDerived  FactBasis = "DERIVED"
)

type FactConfidence string

const (
	ConfidenceHigh   FactConfidence = "HIGH"
	ConfidenceMedium FactConfidence = "MEDIUM"
	ConfidenceLow    FactConfidence = "LOW"
)

type FactProvenance struct {
	SourcePath string    `json:"source"`
	Locator    string    `json:"locator,omitempty"`
	Extractor  string    `json:"extractor"`
	Digest     string    `json:"digest,omitempty"`
	ObservedAt time.Time `json:"observed_at"`
}

type ProjectFact struct {
	Category   string           `json:"category"`
	Key        string           `json:"key"`
	ValueType  FactValueType    `json:"value_type"`
	Value      any              `json:"value"`
	Basis      FactBasis        `json:"basis"`
	Confidence FactConfidence   `json:"confidence"`
	ObservedAt time.Time        `json:"observed_at"`
	Provenance []FactProvenance `json:"provenance"`
}

type SnapshotCompleteness string

const (
	CompletenessComplete SnapshotCompleteness = "COMPLETE"
	CompletenessPartial  SnapshotCompleteness = "PARTIAL"
)

// ProjectContextSnapshot is an immutable, structured view of static project
// evidence. Values are intentionally JSON-compatible so the same contract can
// be persisted and consumed by Planning, Composer, and future routers.
type ProjectContextSnapshot struct {
	ID             string               `json:"id,omitempty"`
	ProjectID      string               `json:"project_id,omitempty"`
	CanonicalPath  string               `json:"canonical_path,omitempty"`
	ScannerVersion string               `json:"scanner_version"`
	Identity       CodeIdentity         `json:"identity"`
	Completeness   SnapshotCompleteness `json:"completeness"`
	Facts          []ProjectFact        `json:"facts"`
	Warnings       []string             `json:"warnings,omitempty"`
	ObservedAt     time.Time            `json:"observed_at"`
}

type discoveryState struct {
	root       string
	observedAt time.Time
	candidates int
	totalBytes int64
	facts      []ProjectFact
	warnings   []string
	seen       map[string]bool
}

func recognizedDocument(path string) bool {
	base := filepath.Base(path)
	if base == "go.mod" || base == "go.work" || base == "package.json" || base == "pnpm-workspace.yaml" || base == "yarn.lock" || base == "package-lock.json" || base == "Cargo.toml" || base == "pyproject.toml" || base == "requirements.txt" || base == "Makefile" || base == "Dockerfile" || base == "README.md" || base == "AGENTS.md" {
		return true
	}
	slash := filepath.ToSlash(path)
	return strings.HasPrefix(slash, ".github/workflows/") || strings.HasPrefix(slash, ".gitlab/")
}

func addFact(state *discoveryState, category, key string, valueType FactValueType, value any, confidence FactConfidence, source, locator, extractor string, digest string) {
	if len(state.facts) >= MaxProjectFacts {
		state.warnings = append(state.warnings, "project fact budget exceeded")
		return
	}
	identity := category + "\x00" + key
	if state.seen[identity] {
		return
	}
	state.seen[identity] = true
	state.facts = append(state.facts, ProjectFact{Category: category, Key: key, ValueType: valueType, Value: value, Basis: FactObserved, Confidence: confidence, ObservedAt: state.observedAt, Provenance: []FactProvenance{{SourcePath: filepath.ToSlash(source), Locator: locator, Extractor: extractor, Digest: digest, ObservedAt: state.observedAt}}})
}

// addDistinctFact keeps the stable short key for the first observation while
// retaining additional evidence from another manifest/workflow instead of
// silently discarding it through the global category/key deduplication.
func addDistinctFact(state *discoveryState, category, key string, valueType FactValueType, value any, confidence FactConfidence, source, locator, extractor, digest string) {
	if state.seen[category+"\x00"+key] {
		key += "@" + filepath.ToSlash(source)
	}
	addFact(state, category, key, valueType, value, confidence, source, locator, extractor, digest)
}

func readDiscoveryFile(state *discoveryState, path string) ([]byte, string, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, "", err
	}
	if info.Mode()&os.ModeSymlink != 0 || info.IsDir() {
		return nil, "", fmt.Errorf("unsafe source")
	}
	if info.Size() > MaxDiscoveryFileBytes {
		state.warnings = append(state.warnings, fmt.Sprintf("skipped oversized file %s", filepath.ToSlash(path)))
		return nil, "", nil
	}
	if state.totalBytes+info.Size() > MaxDiscoveryTotalBytes {
		state.warnings = append(state.warnings, "project discovery byte budget exceeded")
		return nil, "", nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", err
	}
	state.totalBytes += int64(len(data))
	return data, digestBytes(data), nil
}

func parseManifest(state *discoveryState, relative string, data []byte, digest string) {
	base := filepath.Base(relative)
	source := filepath.ToSlash(relative)
	switch base {
	case "go.mod":
		module := ""
		goVersion := ""
		scanner := bufio.NewScanner(strings.NewReader(string(data)))
		for scanner.Scan() {
			fields := strings.Fields(scanner.Text())
			if len(fields) >= 2 && fields[0] == "module" {
				module = fields[1]
			}
			if len(fields) >= 2 && fields[0] == "go" {
				goVersion = fields[1]
			}
		}
		if module != "" {
			addFact(state, "stack", "go.module", FactString, module, ConfidenceHigh, source, "module", "go.mod", digest)
		}
		if goVersion != "" {
			addFact(state, "stack", "go.version", FactString, goVersion, ConfidenceHigh, source, "go", "go.mod", digest)
		}
		addFact(state, "stack", "go", FactBool, true, ConfidenceHigh, source, "", "go.mod", digest)
		addFact(state, "frameworks", "go-test", FactString, "go test", ConfidenceHigh, source, "", "go.mod", digest)
		// Do not invent commands.build/test from go.mod alone. Only observed
		// scripts (Makefile, package.json, CI) may occupy the commands category.
	case "go.work":
		uses := make([]string, 0, 4)
		inUseBlock := false
		scanner := bufio.NewScanner(strings.NewReader(string(data)))
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "use (" {
				inUseBlock = true
				continue
			}
			if inUseBlock && line == ")" {
				inUseBlock = false
				continue
			}
			if strings.HasPrefix(line, "use ") {
				line = strings.TrimSpace(strings.TrimPrefix(line, "use "))
			}
			if inUseBlock || strings.HasPrefix(strings.TrimSpace(scanner.Text()), "use ") {
				line = strings.Trim(strings.TrimSpace(line), "\"")
				if line != "" && !strings.HasPrefix(line, "//") {
					uses = append(uses, line)
				}
			}
		}
		if len(uses) > 0 {
			addFact(state, "workspace", "go.use", FactList, uses, ConfidenceHigh, source, "use", "go.work", digest)
		}
		addFact(state, "workspace", "go.work", FactBool, true, ConfidenceHigh, source, "", "go.work", digest)
	case "package.json":
		var manifest struct {
			Name            string                     `json:"name"`
			PackageManager  string                     `json:"packageManager"`
			Scripts         map[string]json.RawMessage `json:"scripts"`
			Dependencies    map[string]json.RawMessage `json:"dependencies"`
			DevDependencies map[string]json.RawMessage `json:"devDependencies"`
		}
		if json.Unmarshal(data, &manifest) != nil {
			state.warnings = append(state.warnings, "invalid package.json")
			return
		}
		if manifest.Name != "" {
			addFact(state, "stack", "node.package", FactString, manifest.Name, ConfidenceHigh, source, "name", "package.json", digest)
		}
		if manifest.PackageManager != "" {
			addFact(state, "dependencies", "package_manager", FactString, manifest.PackageManager, ConfidenceHigh, source, "packageManager", "package.json", digest)
		}
		scripts := make([]string, 0, len(manifest.Scripts))
		for name, raw := range manifest.Scripts {
			scripts = append(scripts, name)
			var command string
			if json.Unmarshal(raw, &command) == nil && strings.TrimSpace(command) != "" {
				switch name {
				case "build", "test", "lint", "e2e", "typecheck", "format", "format:check":
					addDistinctFact(state, "commands", name, FactString, command, ConfidenceHigh, source, "scripts."+name, "package.json", digest)
				}
			}
		}
		sort.Strings(scripts)
		if len(scripts) > 0 {
			addFact(state, "scripts", "npm", FactList, scripts, ConfidenceHigh, source, "scripts", "package.json", digest)
		}
		deps := make([]string, 0, len(manifest.Dependencies)+len(manifest.DevDependencies))
		for name := range manifest.Dependencies {
			deps = append(deps, name)
		}
		for name := range manifest.DevDependencies {
			deps = append(deps, name)
		}
		sort.Strings(deps)
		if len(deps) > 0 {
			addFact(state, "dependencies", "node", FactList, deps, ConfidenceMedium, source, "dependencies", "package.json", digest)
		}
		for _, dependency := range deps {
			if framework, ok := recognizedFramework(dependency); ok {
				addDistinctFact(state, "frameworks", framework, FactString, dependency, ConfidenceHigh, source, "dependencies."+dependency, "package.json", digest)
			}
		}
		if relativeDir := filepath.ToSlash(filepath.Dir(relative)); relativeDir != "." {
			workspaceName := manifest.Name
			if workspaceName == "" {
				workspaceName = relativeDir
			}
			addFact(state, "workspace", "package."+relativeDir, FactString, workspaceName, ConfidenceHigh, source, "name", "package.json", digest)
		}
	case "Cargo.toml":
		addFact(state, "stack", "rust", FactBool, true, ConfidenceHigh, source, "", "Cargo.toml", digest)
	case "pyproject.toml", "requirements.txt":
		addFact(state, "stack", "python", FactBool, true, ConfidenceHigh, source, "", base, digest)
	case "Makefile":
		addFact(state, "scripts", "make", FactBool, true, ConfidenceHigh, source, "", "Makefile", digest)
	case "Dockerfile":
		addFact(state, "architecture", "containerized", FactBool, true, ConfidenceMedium, source, "", "Dockerfile", digest)
	}
}

func recognizedFramework(dependency string) (string, bool) {
	switch dependency {
	case "react", "react-dom":
		return "react", true
	case "vite":
		return "vite", true
	case "vitest":
		return "vitest", true
	case "@playwright/test", "playwright":
		return "playwright", true
	case "cypress":
		return "cypress", true
	case "jest":
		return "jest", true
	case "next":
		return "next", true
	case "typescript":
		return "typescript", true
	case "eslint":
		return "eslint", true
	default:
		return "", false
	}
}

func parseCIWorkflow(state *discoveryState, relative string, data []byte, digest string) {
	commands := make([]string, 0, 8)
	for _, rawLine := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(rawLine)
		if !strings.HasPrefix(line, "- run:") {
			continue
		}
		command := strings.TrimSpace(strings.TrimPrefix(line, "- run:"))
		if command != "" {
			commands = append(commands, command)
		}
	}
	if len(commands) > 0 {
		addDistinctFact(state, "ci", "commands", FactList, commands, ConfidenceHigh, filepath.ToSlash(relative), "run", "ci-workflow", digest)
	}
}

func discoverPath(state *discoveryState, path string, entry os.DirEntry, root string) error {
	if state.candidates >= MaxDiscoveryCandidates {
		state.warnings = append(state.warnings, "project discovery candidate budget exceeded")
		return filepath.SkipAll
	}
	state.candidates++
	if entry.IsDir() {
		return nil
	}
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return nil
	}
	source := filepath.ToSlash(relative)
	base := filepath.Base(relative)
	if strings.HasSuffix(base, "_test.go") || strings.HasSuffix(base, ".test.ts") || strings.HasSuffix(base, ".test.tsx") || strings.HasSuffix(base, ".spec.ts") || strings.HasSuffix(base, ".spec.tsx") {
		addFact(state, "tests", "present", FactBool, true, ConfidenceHigh, source, "", "filename-pattern", "")
	}
	if base == "main.go" || base == "main.ts" || base == "main.tsx" || base == "index.html" {
		addFact(state, "entrypoints", source, FactString, source, ConfidenceMedium, source, "filename", "filename-pattern", "")
	}
	if strings.HasPrefix(source, ".github/workflows/") || strings.HasPrefix(source, ".gitlab/") {
		addFact(state, "ci", "workflows", FactBool, true, ConfidenceHigh, source, "workflow", "path-pattern", "")
	}
	if base == "AGENTS.md" {
		addFact(state, "conventions", "agents", FactBool, true, ConfidenceMedium, source, "", "durable-doc", "")
	}
	if !recognizedDocument(relative) {
		return nil
	}
	data, digest, err := readDiscoveryFile(state, path)
	if err != nil {
		return nil
	}
	if len(data) == 0 {
		return nil
	}
	redacted := []byte(security.Redact(string(data)))
	parseManifest(state, relative, redacted, digest)
	slash := filepath.ToSlash(relative)
	if strings.HasPrefix(slash, ".github/workflows/") || strings.HasPrefix(slash, ".gitlab/") {
		parseCIWorkflow(state, relative, redacted, digest)
	}
	return nil
}

func classifyDirectory(state *discoveryState, path, root string) {
	relative, err := filepath.Rel(root, path)
	if err != nil || relative == "." {
		return
	}
	source := filepath.ToSlash(relative)
	if noiseDiscoveryPath(source) {
		return
	}
	// Nested fixture trees (…/fixtures/case-001/src) are not product modules.
	if strings.Count(source, "/") > 2 {
		return
	}
	base := filepath.Base(relative)
	if base == "cmd" || base == "internal" || base == "src" || base == "web" || base == "packages" || base == "modules" {
		addFact(state, "architecture", "module."+source, FactBool, true, ConfidenceMedium, source, "directory", "directory-pattern", "")
	}
}

func noiseDiscoveryPath(source string) bool {
	for _, segment := range strings.Split(source, "/") {
		switch segment {
		case "fixtures", "testdata", "node_modules", "vendor", "dist", "build", "coverage",
			"__tests__", "e2e-artifacts", ".worktrees", ".cache", ".git":
			return true
		}
	}
	return false
}

// Discover performs bounded, static project discovery. It never executes a
// project command and returns PARTIAL with warnings when budgets are reached.
func Discover(root string, metadata Metadata) (ProjectContextSnapshot, error) {
	root, err := validateDiscoveryRoot(root)
	if err != nil {
		return ProjectContextSnapshot{}, err
	}
	identity, err := InspectCodeIdentity(root)
	if err != nil {
		return ProjectContextSnapshot{}, err
	}
	return discoverWithIdentity(root, metadata, identity)
}

// DiscoverWithIdentity performs the bounded static scan using an identity that
// was already inspected for this same project. Callers that have just loaded
// the current ProjectIntelligenceView can avoid repeating the Git subprocesses.
func DiscoverWithIdentity(root string, metadata Metadata, identity CodeIdentity) (ProjectContextSnapshot, error) {
	root, err := validateDiscoveryRoot(root)
	if err != nil {
		return ProjectContextSnapshot{}, err
	}
	return discoverWithIdentity(root, metadata, identity)
}

func validateDiscoveryRoot(root string) (string, error) {
	root = filepath.Clean(strings.TrimSpace(root))
	if root == "" || root == "." {
		return "", fmt.Errorf("project root is required")
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("project root unavailable")
	}
	return root, nil
}

func discoverWithIdentity(root string, metadata Metadata, identity CodeIdentity) (ProjectContextSnapshot, error) {
	now := time.Now().UTC()
	state := &discoveryState{root: root, observedAt: now, seen: map[string]bool{}}
	walkErr := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			state.warnings = append(state.warnings, walkErr.Error())
			return nil
		}
		if entry.IsDir() {
			name := entry.Name()
			if path != root && (name == ".git" || name == "node_modules" || name == "vendor" || name == "dist" || name == "build" || name == ".cache" || name == "fixtures" || name == "testdata" || name == "coverage" || name == "__tests__" || name == "e2e-artifacts" || name == ".worktrees" || name == "DEV") {
				if name == "DEV" {
					addFact(state, "conventions", "dev-docs", FactBool, true, ConfidenceMedium, "DEV", "", "durable-doc", "")
				}
				return filepath.SkipDir
			}
			depth := strings.Count(filepath.ToSlash(strings.TrimPrefix(path, root)), "/")
			if depth > MaxDiscoveryDepth {
				return filepath.SkipDir
			}
			classifyDirectory(state, path, root)
			return nil
		}
		return discoverPath(state, path, entry, root)
	})
	if walkErr != nil && walkErr != filepath.SkipAll {
		return ProjectContextSnapshot{}, walkErr
	}
	sort.Slice(state.facts, func(i, j int) bool {
		if state.facts[i].Category == state.facts[j].Category {
			return state.facts[i].Key < state.facts[j].Key
		}
		return state.facts[i].Category < state.facts[j].Category
	})
	completeness := CompletenessComplete
	if len(state.warnings) > 0 || state.candidates >= MaxDiscoveryCandidates || state.totalBytes >= MaxDiscoveryTotalBytes {
		completeness = CompletenessPartial
	}
	return ProjectContextSnapshot{ProjectID: metadata.ProjectID, CanonicalPath: root, ScannerVersion: ScannerVersion, Identity: identity, Completeness: completeness, Facts: state.facts, Warnings: state.warnings, ObservedAt: now}, nil
}

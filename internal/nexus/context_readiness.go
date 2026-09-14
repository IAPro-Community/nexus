package nexus

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/nexus/contextsnapshot"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

type ContextReadinessState string

const (
	ContextMissing   ContextReadinessState = "MISSING"
	ContextHydrating ContextReadinessState = "HYDRATING"
	ContextReady     ContextReadinessState = "READY"
	ContextStale     ContextReadinessState = "STALE"
	ContextFailed    ContextReadinessState = "FAILED"
)

type ContextFingerprint struct {
	ProjectID        string `json:"project_id"`
	CanonicalPath    string `json:"canonical_path"`
	Branch           string `json:"branch"`
	Head             string `json:"head"`
	DirtyFingerprint string `json:"dirty_fingerprint"`
	MaestroVersion   string `json:"maestro_version"`
}

type ContextReadiness struct {
	ProjectID             string                `json:"project_id"`
	State                 ContextReadinessState `json:"state"`
	CurrentFingerprint    ContextFingerprint    `json:"current_fingerprint"`
	CurrentFingerprintID  string                `json:"current_fingerprint_id"`
	HydratedFingerprintID string                `json:"hydrated_fingerprint_id,omitempty"`
	MaestroAvailable      bool                  `json:"maestro_available"`
	MaestroVersion        string                `json:"maestro_version"`
	Error                 string                `json:"error,omitempty"`
	HydratedAt            *time.Time            `json:"hydrated_at,omitempty"`
	UpdatedAt             time.Time             `json:"updated_at,omitempty"`
}

func gitContextValue(root string, args ...string) string {
	out, err := exec.Command("git", append([]string{"-C", root}, args...)...).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func currentContextFingerprint(project *store.Project, maestro MaestroStatus) (ContextFingerprint, string, string, error) {
	// OFF and ASSIST treat Maestro as optional enrichment: the durable project
	// context fingerprint must not thrash when Maestro appears or disappears.
	// ORCHESTRATE pins Maestro version because execution depends on it.
	version := "optional"
	if project.MaestroMode == store.MaestroOrchestrate {
		version = "unavailable"
		if maestro.Capabilities != nil && strings.TrimSpace(maestro.Capabilities.Version) != "" {
			version = strings.TrimSpace(maestro.Capabilities.Version)
		}
	}
	dirtyRaw := gitContextValue(project.CanonicalPath, "status", "--porcelain=v1", "--untracked-files=normal")
	dirtyID := ""
	if dirtyRaw != "" {
		h := sha256.New()
		_, _ = h.Write([]byte(dirtyRaw))
		for _, line := range strings.Split(dirtyRaw, "\n") {
			line = strings.TrimSpace(line)
			if len(line) < 4 {
				continue
			}
			rel := strings.TrimSpace(line[3:])
			if rel == "" {
				continue
			}
			data, err := os.ReadFile(filepath.Join(project.CanonicalPath, rel))
			if err != nil {
				_, _ = h.Write([]byte(rel))
				_, _ = h.Write([]byte{0})
				continue
			}
			fileSum := sha256.Sum256(data)
			_, _ = h.Write([]byte(rel))
			_, _ = h.Write(fileSum[:])
		}
		dirtyID = hex.EncodeToString(h.Sum(nil))
	}
	branch := gitContextValue(project.CanonicalPath, "branch", "--show-current")
	if branch == "" {
		branch = project.DefaultBranch
	}
	fp := ContextFingerprint{ProjectID: project.ID, CanonicalPath: project.CanonicalPath, Branch: branch, Head: gitContextValue(project.CanonicalPath, "rev-parse", "HEAD"), DirtyFingerprint: dirtyID, MaestroVersion: version}
	raw, err := json.Marshal(fp)
	if err != nil {
		return fp, "", "", err
	}
	sum := sha256.Sum256(raw)
	return fp, hex.EncodeToString(sum[:]), string(raw), nil
}

func hasDurableProjectContext(root string) bool {
	candidates := []string{"AGENTS.md", filepath.Join("DEV", "INDEX.md"), filepath.Join("DEV", "CONTEXT.md")}
	for _, rel := range candidates {
		if info, err := os.Stat(filepath.Join(root, rel)); err == nil && !info.IsDir() {
			return true
		}
	}
	return false
}

const defaultProjectContext = `# Contexto do projeto

Este arquivo foi criado pelo IAPro Nexus para habilitar o contexto durável do projeto.

## Objetivo

Descreva aqui o objetivo principal deste projeto.

## Regras de trabalho

- Preserve a arquitetura e as convenções existentes.
- Execute os testes relevantes antes de concluir alterações.
- Não exponha credenciais ou dados sensíveis.
`

func createDurableProjectContext(root string) error {
	if hasDurableProjectContext(root) {
		return nil
	}
	path := filepath.Join(root, "AGENTS.md")
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		if os.IsExist(err) || hasDurableProjectContext(root) {
			return nil
		}
		return fmt.Errorf("create durable project context: %w", err)
	}
	defer file.Close()
	if _, err := file.WriteString(defaultProjectContext); err != nil {
		_ = os.Remove(path)
		return fmt.Errorf("write durable project context: %w", err)
	}
	return nil
}

func (n *Nexus) currentMaestroStatus() MaestroStatus {
	if n != nil && n.maestroStatus != nil {
		return n.maestroStatus()
	}
	return NewMaestroClient().Status()
}

func (n *Nexus) ObserveContextReadiness(projectID string) (*ContextReadiness, error) {
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	project, err := st.GetProject(projectID)
	if err != nil {
		return nil, err
	}
	maestro := n.currentMaestroStatus()
	fp, currentID, _, err := currentContextFingerprint(&project, maestro)
	if err != nil {
		return nil, err
	}
	record, err := st.GetContextReadiness(projectID)
	if err != nil && store.IsContextReadinessMissing(err) {
		return &ContextReadiness{ProjectID: projectID, State: ContextMissing, CurrentFingerprint: fp, CurrentFingerprintID: currentID, MaestroAvailable: maestro.Available, MaestroVersion: fp.MaestroVersion}, nil
	}
	if err != nil {
		return nil, err
	}
	state := ContextReadinessState(record.State)
	staleReason := ""
	if (state == ContextReady || state == ContextHydrating) && record.FingerprintHash != currentID {
		staleReason = "project or Maestro fingerprint changed since readiness checkpoint"
	}
	if state == ContextReady && !hasDurableProjectContext(project.CanonicalPath) {
		staleReason = "durable project context used by Composer is no longer present"
	}
	// Only ORCHESTRATE requires Maestro at runtime. ASSIST degrades without
	// Maestro and must not invalidate an otherwise READY durable context.
	if state == ContextReady && project.MaestroMode == store.MaestroOrchestrate && !maestro.Available {
		staleReason = "Maestro became unavailable after the readiness checkpoint"
	}
	if staleReason != "" {
		state = ContextStale
		record.State = string(state)
		record.Error = staleReason
		if updated, updateErr := st.PutContextReadiness(*record); updateErr == nil {
			record = updated
		}
	}
	return &ContextReadiness{ProjectID: projectID, State: state, CurrentFingerprint: fp, CurrentFingerprintID: currentID, HydratedFingerprintID: record.FingerprintHash, MaestroAvailable: maestro.Available, MaestroVersion: fp.MaestroVersion, Error: record.Error, HydratedAt: record.HydratedAt, UpdatedAt: record.UpdatedAt}, nil
}

// PrepareContext evaluates durable context readiness. Nexus does not invent
// Maestro hydration: READY is recorded when durable project context files
// exist. Maestro is required only for ORCHESTRATE; ASSIST degrades without it.
func (n *Nexus) PrepareContext(projectID string) (*ContextReadiness, error) {
	return n.prepareContext(projectID, false)
}

// PrepareContextWithBootstrap creates a minimal AGENTS.md only when the
// project has no supported durable context file. Existing files are preserved.
func (n *Nexus) PrepareContextWithBootstrap(projectID string) (*ContextReadiness, error) {
	return n.prepareContext(projectID, true)
}

func (n *Nexus) prepareContext(projectID string, bootstrap bool) (*ContextReadiness, error) {
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	project, err := st.GetProject(projectID)
	if err != nil {
		return nil, err
	}
	maestro := n.currentMaestroStatus()
	if bootstrap && !hasDurableProjectContext(project.CanonicalPath) {
		if err := createDurableProjectContext(project.CanonicalPath); err != nil {
			return nil, err
		}
	}
	fp, currentID, raw, err := currentContextFingerprint(&project, maestro)
	if err != nil {
		return nil, err
	}
	record := store.ContextReadinessRecord{ProjectID: projectID, State: string(ContextHydrating), FingerprintHash: currentID, FingerprintJSON: raw, MaestroVersion: fp.MaestroVersion}
	if _, err = st.PutContextReadiness(record); err != nil {
		return nil, err
	}
	switch {
	case project.MaestroMode == store.MaestroOrchestrate && !maestro.Available:
		record.State = string(ContextFailed)
		record.Error = "Maestro ORCHESTRATE requires Maestro, but it is unavailable"
	case !hasDurableProjectContext(project.CanonicalPath):
		record.State = string(ContextFailed)
		record.Error = "durable project context is missing (expected AGENTS.md or DEV context files)"
	default:
		now := time.Now().UTC()
		record.State = string(ContextReady)
		record.Error = ""
		record.HydratedAt = &now
	}
	updated, err := st.PutContextReadiness(record)
	if err != nil {
		return nil, err
	}
	return &ContextReadiness{ProjectID: projectID, State: ContextReadinessState(updated.State), CurrentFingerprint: fp, CurrentFingerprintID: currentID, HydratedFingerprintID: updated.FingerprintHash, MaestroAvailable: maestro.Available, MaestroVersion: fp.MaestroVersion, Error: updated.Error, HydratedAt: updated.HydratedAt, UpdatedAt: updated.UpdatedAt}, nil
}

type ComposerContextNotReadyError struct {
	Readiness *ContextReadiness
}

func (e *ComposerContextNotReadyError) Error() string {
	if e == nil || e.Readiness == nil {
		return "Composer context is not READY"
	}
	if strings.TrimSpace(e.Readiness.Error) != "" {
		return fmt.Sprintf("Composer context is %s: %s", e.Readiness.State, e.Readiness.Error)
	}
	return fmt.Sprintf("Composer context is %s", e.Readiness.State)
}

// ComposerContextData returns the bounded, redacted context envelope supplied
// to Intelligence providers. Backend callers cannot bypass the READY gate.
func (n *Nexus) ComposerContextData(projectID string) (map[string]any, error) {
	readiness, err := n.ObserveContextReadiness(projectID)
	if err != nil {
		return nil, err
	}
	if err := readiness.ValidateComposerReady(); err != nil {
		return nil, &ComposerContextNotReadyError{Readiness: readiness}
	}
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	project, err := st.GetProject(projectID)
	if err != nil {
		return nil, err
	}
	envelope, err := contextsnapshot.Build(project.CanonicalPath, contextsnapshot.Metadata{
		ProjectID: projectID, Branch: readiness.CurrentFingerprint.Branch, Head: readiness.CurrentFingerprint.Head,
		DirtyFingerprint: readiness.CurrentFingerprint.DirtyFingerprint, MaestroVersion: readiness.CurrentFingerprint.MaestroVersion,
	})
	if err != nil {
		return nil, err
	}
	return map[string]any{"project_context": envelope}, nil
}

func (r ContextReadiness) ValidateComposerReady() error {
	if r.State != ContextReady {
		return fmt.Errorf("composer planning requires READY context, current state is %s", r.State)
	}
	return nil
}

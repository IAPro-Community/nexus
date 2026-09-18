package store

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/ids"
)

// Evidence outcomes are intentionally descriptive rather than boolean. A
// timeout, skipped scenario, and failed assertion must remain distinguishable
// in the durable ledger.
const (
	EvidencePass        = "PASS"
	EvidenceFail        = "FAIL"
	EvidenceError       = "ERROR"
	EvidenceTimeout     = "TIMEOUT"
	EvidenceSkipped     = "SKIPPED"
	EvidenceNotVerified = "NOT_VERIFIED"

	EvidenceObserved  = "OBSERVED"
	EvidenceVerified  = "VERIFIED"
	EvidenceCertified = "CERTIFIED"
)

var (
	ErrValidationEvidenceConflict  = errors.New("validation evidence compare-and-swap conflict")
	ErrValidationEvidenceNotFound  = errors.New("validation evidence not found")
	ErrValidationEvidenceInvalid   = errors.New("invalid validation evidence")
	ErrValidationEvidenceIntegrity = errors.New("validation evidence integrity failure")
)

// ValidationEvidenceStream is the append-only head for a project's evidence
// ledger. LastSequence and LastHash are updated in the same transaction as
// every entry, and form the CAS token for the next append.
type ValidationEvidenceStream struct {
	ID           string    `json:"id"`
	ProjectID    string    `json:"project_id,omitempty"`
	Name         string    `json:"name"`
	CreatedAt    time.Time `json:"created_at"`
	LastSequence int64     `json:"last_sequence"`
	LastHash     string    `json:"last_hash,omitempty"`
}

// ValidationEvidenceEntry is the immutable metadata for one validation run.
// EvidenceJSON is deliberately opaque JSON: command-specific details belong
// to the scenario, while the identity/provider fields remain queryable.
type ValidationEvidenceEntry struct {
	ID              string    `json:"id"`
	StreamID        string    `json:"stream_id"`
	Sequence        int64     `json:"sequence"`
	GitSHA          string    `json:"git_sha,omitempty"`
	IdentityDigest  string    `json:"identity_digest,omitempty"`
	RepositoryState string    `json:"repository_state,omitempty"`
	EnvironmentJSON string    `json:"environment_json,omitempty"`
	Provider        string    `json:"provider,omitempty"`
	Profile         string    `json:"profile,omitempty"`
	Model           string    `json:"model,omitempty"`
	Scenario        string    `json:"scenario"`
	CommandDisplay  string    `json:"command_display,omitempty"`
	Outcome         string    `json:"outcome"`
	Confidence      string    `json:"confidence"`
	ExitCode        *int      `json:"exit_code,omitempty"`
	DurationMS      int64     `json:"duration_ms,omitempty"`
	EvidenceJSON    string    `json:"evidence_json,omitempty"`
	PreviousHash    string    `json:"previous_hash,omitempty"`
	EntryHash       string    `json:"entry_hash"`
	CreatedAt       time.Time `json:"created_at"`
}

// ValidationEvidenceAppendRequest carries one immutable entry and the
// optional optimistic CAS token. If ExpectedSequence/ExpectedHash are nil,
// the current stream head is read and used as the token; the conditional
// update still prevents two concurrent writers from appending the same head.
type ValidationEvidenceAppendRequest struct {
	StreamID         string
	Entry            ValidationEvidenceEntry
	ExpectedSequence *int64
	ExpectedHash     *string
}

func (s *Store) CreateValidationEvidenceStream(projectID, name string) (ValidationEvidenceStream, error) {
	if strings.TrimSpace(name) == "" {
		return ValidationEvidenceStream{}, fmt.Errorf("%w: stream name is required", ErrValidationEvidenceInvalid)
	}
	stream := ValidationEvidenceStream{ID: "ves_" + ids.NewRuntimeID(), ProjectID: projectID, Name: name, CreatedAt: time.Now().UTC()}
	_, err := s.db.Exec(`INSERT INTO validation_evidence_streams(id, project_id, name, created_at) VALUES(?, ?, ?, ?)`,
		stream.ID, nullableString(projectID), stream.Name, stream.CreatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return ValidationEvidenceStream{}, fmt.Errorf("create validation evidence stream: %w", err)
	}
	return stream, nil
}

func (s *Store) GetValidationEvidenceStream(id string) (ValidationEvidenceStream, error) {
	stream, err := scanValidationEvidenceStream(s.db.QueryRow(`SELECT id, project_id, name, created_at, last_sequence, last_hash FROM validation_evidence_streams WHERE id=?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return ValidationEvidenceStream{}, fmt.Errorf("%w: stream %s", ErrValidationEvidenceNotFound, id)
	}
	return stream, err
}

func (s *Store) GetValidationEvidenceStreamByName(projectID, name string) (ValidationEvidenceStream, error) {
	stream, err := scanValidationEvidenceStream(s.db.QueryRow(`SELECT id, project_id, name, created_at, last_sequence, last_hash FROM validation_evidence_streams WHERE project_id IS ? AND name=?`, nullableString(projectID), name))
	if errors.Is(err, sql.ErrNoRows) {
		return ValidationEvidenceStream{}, fmt.Errorf("%w: stream %s", ErrValidationEvidenceNotFound, name)
	}
	return stream, err
}

func (s *Store) ListValidationEvidenceStreams(projectID string) ([]ValidationEvidenceStream, error) {
	query := `SELECT id, project_id, name, created_at, last_sequence, last_hash FROM validation_evidence_streams`
	args := []any{}
	if projectID == "" {
		query += ` WHERE project_id IS NULL`
	} else {
		query += ` WHERE project_id=?`
		args = append(args, projectID)
	}
	query += ` ORDER BY created_at, id`
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list validation evidence streams: %w", err)
	}
	defer rows.Close()
	result := []ValidationEvidenceStream{}
	for rows.Next() {
		stream, scanErr := scanValidationEvidenceStream(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		result = append(result, stream)
	}
	return result, rows.Err()
}

func (s *Store) AppendValidationEvidence(request ValidationEvidenceAppendRequest) (ValidationEvidenceEntry, error) {
	if strings.TrimSpace(request.StreamID) == "" {
		return ValidationEvidenceEntry{}, fmt.Errorf("%w: stream id is required", ErrValidationEvidenceInvalid)
	}
	if request.ExpectedSequence == nil && request.ExpectedHash != nil || request.ExpectedSequence != nil && request.ExpectedHash == nil {
		return ValidationEvidenceEntry{}, fmt.Errorf("%w: expected sequence and hash must be provided together", ErrValidationEvidenceInvalid)
	}

	entry := request.Entry
	entry.StreamID = request.StreamID
	if entry.ID == "" {
		entry.ID = "vee_" + ids.NewRuntimeID()
	}
	if strings.TrimSpace(entry.Scenario) == "" {
		return ValidationEvidenceEntry{}, fmt.Errorf("%w: scenario is required", ErrValidationEvidenceInvalid)
	}
	if strings.TrimSpace(entry.Outcome) == "" || strings.TrimSpace(entry.Confidence) == "" {
		return ValidationEvidenceEntry{}, fmt.Errorf("%w: outcome and confidence are required", ErrValidationEvidenceInvalid)
	}
	if entry.EnvironmentJSON == "" {
		entry.EnvironmentJSON = "{}"
	}
	if !json.Valid([]byte(entry.EnvironmentJSON)) {
		return ValidationEvidenceEntry{}, fmt.Errorf("%w: environment_json must be valid JSON", ErrValidationEvidenceInvalid)
	}
	if entry.EvidenceJSON == "" {
		entry.EvidenceJSON = "{}"
	}
	if !json.Valid([]byte(entry.EvidenceJSON)) {
		return ValidationEvidenceEntry{}, fmt.Errorf("%w: evidence_json must be valid JSON", ErrValidationEvidenceInvalid)
	}
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now().UTC()
	}

	tx, err := s.db.Begin()
	if err != nil {
		return ValidationEvidenceEntry{}, fmt.Errorf("begin evidence append: %w", err)
	}
	defer tx.Rollback()

	// An explicit ID is the idempotency key. Retrying the same append returns
	// the already committed immutable row, without advancing the stream.
	if existing, getErr := getValidationEvidenceEntry(tx, entry.ID); getErr == nil {
		if existing.StreamID != request.StreamID {
			return ValidationEvidenceEntry{}, fmt.Errorf("%w: entry id already belongs to another stream", ErrValidationEvidenceConflict)
		}
		return existing, nil
	} else if !errors.Is(getErr, sql.ErrNoRows) {
		return ValidationEvidenceEntry{}, getErr
	}

	stream, err := scanValidationEvidenceStream(tx.QueryRow(`SELECT id, project_id, name, created_at, last_sequence, last_hash FROM validation_evidence_streams WHERE id=?`, request.StreamID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ValidationEvidenceEntry{}, fmt.Errorf("%w: stream %s", ErrValidationEvidenceNotFound, request.StreamID)
		}
		return ValidationEvidenceEntry{}, err
	}
	expectedSequence, expectedHash := stream.LastSequence, stream.LastHash
	if request.ExpectedSequence != nil {
		expectedSequence, expectedHash = *request.ExpectedSequence, *request.ExpectedHash
		if expectedSequence != stream.LastSequence || expectedHash != stream.LastHash {
			return ValidationEvidenceEntry{}, ErrValidationEvidenceConflict
		}
	}

	entry.Sequence = expectedSequence + 1
	entry.PreviousHash = expectedHash
	entry.EntryHash = validationEvidenceHash(entry)

	result, err := tx.Exec(`UPDATE validation_evidence_streams SET last_sequence=?, last_hash=? WHERE id=? AND last_sequence=? AND last_hash=?`,
		entry.Sequence, entry.EntryHash, request.StreamID, expectedSequence, expectedHash)
	if err != nil {
		return ValidationEvidenceEntry{}, fmt.Errorf("update evidence stream head: %w", err)
	}
	if affected, affectedErr := result.RowsAffected(); affectedErr != nil {
		return ValidationEvidenceEntry{}, affectedErr
	} else if affected != 1 {
		return ValidationEvidenceEntry{}, ErrValidationEvidenceConflict
	}

	_, err = tx.Exec(`INSERT INTO validation_evidence_entries(
		id, stream_id, sequence, git_sha, identity_digest, repository_state,
		environment_json, provider, profile, model, scenario, command_display,
		outcome, confidence, exit_code, duration_ms, evidence_json,
		previous_hash, entry_hash, created_at)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		entry.ID, entry.StreamID, entry.Sequence, entry.GitSHA, entry.IdentityDigest, entry.RepositoryState,
		entry.EnvironmentJSON, entry.Provider, entry.Profile, entry.Model, entry.Scenario, entry.CommandDisplay,
		entry.Outcome, entry.Confidence, entry.ExitCode, entry.DurationMS, entry.EvidenceJSON,
		entry.PreviousHash, entry.EntryHash, entry.CreatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return ValidationEvidenceEntry{}, fmt.Errorf("insert validation evidence entry: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return ValidationEvidenceEntry{}, fmt.Errorf("commit validation evidence entry: %w", err)
	}
	return entry, nil
}

func (s *Store) GetValidationEvidenceEntry(id string) (ValidationEvidenceEntry, error) {
	entry, err := getValidationEvidenceEntry(s.db, id)
	if errors.Is(err, sql.ErrNoRows) {
		return ValidationEvidenceEntry{}, fmt.Errorf("%w: entry %s", ErrValidationEvidenceNotFound, id)
	}
	return entry, err
}

func (s *Store) ListValidationEvidenceEntries(streamID string, limit int) ([]ValidationEvidenceEntry, error) {
	if strings.TrimSpace(streamID) == "" {
		return nil, fmt.Errorf("%w: stream id is required", ErrValidationEvidenceInvalid)
	}
	if limit <= 0 {
		limit = 100
	}
	return s.listValidationEvidenceEntries(streamID, &limit)
}

// ListAllValidationEvidenceEntries returns the complete immutable stream in
// sequence order. It is intentionally separate from the bounded list API so
// report projections can make their no-truncation requirement explicit.
func (s *Store) ListAllValidationEvidenceEntries(streamID string) ([]ValidationEvidenceEntry, error) {
	if strings.TrimSpace(streamID) == "" {
		return nil, fmt.Errorf("%w: stream id is required", ErrValidationEvidenceInvalid)
	}
	return s.listValidationEvidenceEntries(streamID, nil)
}

func (s *Store) listValidationEvidenceEntries(streamID string, limit *int) ([]ValidationEvidenceEntry, error) {
	query := `SELECT id, stream_id, sequence, git_sha, identity_digest, repository_state,
		environment_json, provider, profile, model, scenario, command_display,
		outcome, confidence, exit_code, duration_ms, evidence_json,
		previous_hash, entry_hash, created_at
		FROM validation_evidence_entries WHERE stream_id=? ORDER BY sequence`
	args := []any{streamID}
	if limit != nil {
		query += ` LIMIT ?`
		args = append(args, *limit)
	}
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list validation evidence entries: %w", err)
	}
	defer rows.Close()
	result := []ValidationEvidenceEntry{}
	for rows.Next() {
		entry, scanErr := scanValidationEvidenceEntry(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		result = append(result, entry)
	}
	return result, rows.Err()
}

// VerifyValidationEvidenceChain recomputes all entry hashes and checks that
// the stream head matches the final row. It is read-only and suitable for a
// validation gate before promoting a stream's claims.
func (s *Store) VerifyValidationEvidenceChain(streamID string) error {
	stream, err := s.GetValidationEvidenceStream(streamID)
	if err != nil {
		return err
	}
	entries, err := s.listValidationEvidenceEntries(streamID, nil)
	if err != nil {
		return err
	}
	previousHash := ""
	for i, entry := range entries {
		if entry.Sequence != int64(i+1) || entry.PreviousHash != previousHash || entry.EntryHash != validationEvidenceHash(entry) {
			return fmt.Errorf("%w: stream %s sequence %d", ErrValidationEvidenceIntegrity, streamID, entry.Sequence)
		}
		previousHash = entry.EntryHash
	}
	if stream.LastSequence != int64(len(entries)) || stream.LastHash != previousHash {
		return fmt.Errorf("%w: stream head does not match entries", ErrValidationEvidenceIntegrity)
	}
	return nil
}

type validationEvidenceHashPayload struct {
	ID              string `json:"id"`
	StreamID        string `json:"stream_id"`
	Sequence        int64  `json:"sequence"`
	GitSHA          string `json:"git_sha"`
	IdentityDigest  string `json:"identity_digest"`
	RepositoryState string `json:"repository_state"`
	EnvironmentJSON string `json:"environment_json"`
	Provider        string `json:"provider"`
	Profile         string `json:"profile"`
	Model           string `json:"model"`
	Scenario        string `json:"scenario"`
	CommandDisplay  string `json:"command_display"`
	Outcome         string `json:"outcome"`
	Confidence      string `json:"confidence"`
	ExitCode        *int   `json:"exit_code"`
	DurationMS      int64  `json:"duration_ms"`
	EvidenceJSON    string `json:"evidence_json"`
	PreviousHash    string `json:"previous_hash"`
	CreatedAt       string `json:"created_at"`
}

func validationEvidenceHash(entry ValidationEvidenceEntry) string {
	payload := validationEvidenceHashPayload{
		ID: entry.ID, StreamID: entry.StreamID, Sequence: entry.Sequence, GitSHA: entry.GitSHA,
		IdentityDigest: entry.IdentityDigest, RepositoryState: entry.RepositoryState,
		EnvironmentJSON: entry.EnvironmentJSON, Provider: entry.Provider, Profile: entry.Profile,
		Model: entry.Model, Scenario: entry.Scenario, CommandDisplay: entry.CommandDisplay,
		Outcome: entry.Outcome, Confidence: entry.Confidence, ExitCode: entry.ExitCode,
		DurationMS: entry.DurationMS, EvidenceJSON: entry.EvidenceJSON, PreviousHash: entry.PreviousHash,
		CreatedAt: entry.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
	raw, _ := json.Marshal(payload)
	digest := sha256.Sum256(raw)
	return hex.EncodeToString(digest[:])
}

func scanValidationEvidenceStream(scanner interface{ Scan(...any) error }) (ValidationEvidenceStream, error) {
	var stream ValidationEvidenceStream
	var projectID sql.NullString
	var created string
	if err := scanner.Scan(&stream.ID, &projectID, &stream.Name, &created, &stream.LastSequence, &stream.LastHash); err != nil {
		return stream, err
	}
	if projectID.Valid {
		stream.ProjectID = projectID.String
	}
	stream.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	return stream, nil
}

func getValidationEvidenceEntry(scanner interface {
	QueryRow(string, ...any) *sql.Row
}, id string) (ValidationEvidenceEntry, error) {
	return scanValidationEvidenceEntry(scanner.QueryRow(`SELECT id, stream_id, sequence, git_sha, identity_digest, repository_state,
		environment_json, provider, profile, model, scenario, command_display,
		outcome, confidence, exit_code, duration_ms, evidence_json,
		previous_hash, entry_hash, created_at FROM validation_evidence_entries WHERE id=?`, id))
}

func scanValidationEvidenceEntry(scanner interface{ Scan(...any) error }) (ValidationEvidenceEntry, error) {
	var entry ValidationEvidenceEntry
	var exitCode sql.NullInt64
	var created string
	if err := scanner.Scan(&entry.ID, &entry.StreamID, &entry.Sequence, &entry.GitSHA, &entry.IdentityDigest, &entry.RepositoryState,
		&entry.EnvironmentJSON, &entry.Provider, &entry.Profile, &entry.Model, &entry.Scenario, &entry.CommandDisplay,
		&entry.Outcome, &entry.Confidence, &exitCode, &entry.DurationMS, &entry.EvidenceJSON,
		&entry.PreviousHash, &entry.EntryHash, &created); err != nil {
		return entry, err
	}
	if exitCode.Valid {
		value := int(exitCode.Int64)
		entry.ExitCode = &value
	}
	entry.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	return entry, nil
}

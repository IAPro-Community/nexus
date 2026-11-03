package store

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/ids"
	"github.com/IAPro-Community/nexus/internal/core/memoryrecovery"
)

// IntelligenceMemory is a visible, deletable habit fact for a project.
type IntelligenceMemory struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"project_id"`
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	Source    string    `json:"source"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (s *Store) UpsertIntelligenceMemory(m IntelligenceMemory) (*IntelligenceMemory, error) {
	m.Key = strings.TrimSpace(m.Key)
	m.Value = strings.TrimSpace(m.Value)
	m.Source = strings.TrimSpace(m.Source)
	if m.ProjectID == "" || m.Key == "" || m.Value == "" {
		return nil, fmt.Errorf("intelligence memory requires project_id, key and value")
	}
	if m.Source == "" {
		m.Source = "composer"
	}
	now := time.Now().UTC()
	if m.ID == "" {
		m.ID = "imem_" + ids.NewRuntimeID()
	}
	if m.CreatedAt.IsZero() {
		m.CreatedAt = now
	}
	m.UpdatedAt = now
	var saved *IntelligenceMemory
	err := memoryrecovery.WithLegacyWriter(s.memoryFenceDir, func() error {
		_, err := s.db.Exec(`INSERT INTO intelligence_memories(id,project_id,key,value,source,created_at,updated_at)
			VALUES(?,?,?,?,?,?,?)
			ON CONFLICT(project_id, key) DO UPDATE SET
				value=excluded.value,
				source=excluded.source,
				updated_at=excluded.updated_at`,
			m.ID, m.ProjectID, m.Key, m.Value, m.Source,
			m.CreatedAt.Format(time.RFC3339Nano), m.UpdatedAt.Format(time.RFC3339Nano))
		if err != nil {
			return fmt.Errorf("upsert intelligence memory: %w", err)
		}
		saved, err = s.GetIntelligenceMemoryByKey(m.ProjectID, m.Key)
		return err
	})
	return saved, err
}

func (s *Store) GetIntelligenceMemoryByKey(projectID, key string) (*IntelligenceMemory, error) {
	row := s.db.QueryRow(`SELECT id,project_id,key,value,source,created_at,updated_at
		FROM intelligence_memories WHERE project_id=? AND key=?`, projectID, key)
	return scanIntelligenceMemory(row)
}

func (s *Store) ListIntelligenceMemories(projectID string) ([]IntelligenceMemory, error) {
	rows, err := s.db.Query(`SELECT id,project_id,key,value,source,created_at,updated_at
		FROM intelligence_memories WHERE project_id=? ORDER BY updated_at DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []IntelligenceMemory{}
	for rows.Next() {
		m, err := scanIntelligenceMemory(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *m)
	}
	return out, rows.Err()
}

func (s *Store) DeleteIntelligenceMemory(projectID, id string) error {
	return memoryrecovery.WithLegacyWriter(s.memoryFenceDir, func() error {
		res, err := s.db.Exec(`DELETE FROM intelligence_memories WHERE project_id=? AND id=?`, projectID, id)
		if err != nil {
			return err
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			return sql.ErrNoRows
		}
		return nil
	})
}

func (s *Store) DeleteIntelligenceMemoryByKey(projectID, key string) error {
	return memoryrecovery.WithLegacyWriter(s.memoryFenceDir, func() error {
		_, err := s.db.Exec(`DELETE FROM intelligence_memories WHERE project_id=? AND key=?`, projectID, key)
		return err
	})
}

type memoryScanner interface {
	Scan(dest ...any) error
}

func scanIntelligenceMemory(row memoryScanner) (*IntelligenceMemory, error) {
	var m IntelligenceMemory
	var created, updated string
	if err := row.Scan(&m.ID, &m.ProjectID, &m.Key, &m.Value, &m.Source, &created, &updated); err != nil {
		return nil, err
	}
	m.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	m.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
	return &m, nil
}

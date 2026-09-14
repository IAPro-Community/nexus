package codex

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/config"
)

// hostAuthObservation records when a chatgpt_account_id was observed as the
// host ~/.codex/auth.json login. Used to attribute legacy shared-host rollouts
// only while that account owned the host login.
type hostAuthObservation struct {
	ObservedAt time.Time `json:"observed_at"`
	AccountID  string    `json:"account_id"`
}

var hostAuthLedgerMu sync.Mutex

func hostAuthLedgerPath() (string, error) {
	data, err := config.DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(data, "codex-host-auth-history.jsonl"), nil
}

// recordHostAuthObservation appends a ledger entry when the host auth account
// changes. Returns the current account id and whether recording succeeded.
func recordHostAuthObservation(accountID string) {
	accountID = strings.TrimSpace(accountID)
	if accountID == "" {
		return
	}
	hostAuthLedgerMu.Lock()
	defer hostAuthLedgerMu.Unlock()

	path, err := hostAuthLedgerPath()
	if err != nil {
		return
	}
	entries, _ := readHostAuthLedgerLocked(path)
	if len(entries) > 0 && strings.EqualFold(entries[len(entries)-1].AccountID, accountID) {
		return
	}
	_ = os.MkdirAll(filepath.Dir(path), 0700)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return
	}
	defer f.Close()
	rec := hostAuthObservation{ObservedAt: time.Now().UTC(), AccountID: accountID}
	b, err := json.Marshal(rec)
	if err != nil {
		return
	}
	_, _ = f.Write(append(b, '\n'))
}

func readHostAuthLedger() []hostAuthObservation {
	hostAuthLedgerMu.Lock()
	defer hostAuthLedgerMu.Unlock()
	path, err := hostAuthLedgerPath()
	if err != nil {
		return nil
	}
	entries, _ := readHostAuthLedgerLocked(path)
	return entries
}

func readHostAuthLedgerLocked(path string) ([]hostAuthObservation, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out []hostAuthObservation
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var rec hostAuthObservation
		if json.Unmarshal([]byte(line), &rec) == nil && rec.AccountID != "" {
			out = append(out, rec)
		}
	}
	return out, sc.Err()
}

// hostOwnedAt reports whether accountID owned the host login at the given time.
// With no ledger yet, the current authenticated host account is the only
// available ownership evidence. Once a ledger exists, historical attribution
// is bounded by its observations below.
func hostOwnedAt(accountID string, at time.Time, currentHostAccountID string) bool {
	accountID = strings.TrimSpace(accountID)
	if accountID == "" {
		return false
	}
	entries := readHostAuthLedger()
	if len(entries) == 0 {
		return strings.EqualFold(accountID, strings.TrimSpace(currentHostAccountID))
	}
	// Walk newest-first: find the latest observation at or before `at`.
	var owner string
	for i := len(entries) - 1; i >= 0; i-- {
		if !entries[i].ObservedAt.After(at) {
			owner = entries[i].AccountID
			break
		}
	}
	if owner == "" {
		// Rollout predates every observation — fail closed.
		return false
	}
	return strings.EqualFold(accountID, owner)
}

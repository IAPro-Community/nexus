package codex

import (
	"context"
	"database/sql"
	"net/url"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// Codex also keeps titles in state_N.sqlite. An immutable read avoids creating
// WAL/shared-memory files while the user is only browsing or canceling.
func readResumeSQLiteIndex(ctx context.Context, home string) map[string]sessionIndexEntry {
	result := make(map[string]sessionIndexEntry)
	for _, dir := range []string{home, filepath.Join(home, ".codex")} {
		paths, _ := filepath.Glob(filepath.Join(dir, "state_*.sqlite"))
		for _, path := range paths {
			uri := (&url.URL{Scheme: "file", Path: filepath.ToSlash(path), RawQuery: "mode=ro&immutable=1"}).String()
			db, err := sql.Open("sqlite", uri)
			if err != nil {
				continue
			}
			rows, err := db.QueryContext(ctx, `SELECT id, title, cwd, updated_at FROM threads`)
			if err != nil {
				_ = db.Close()
				continue
			}
			for rows.Next() {
				var id string
				var title, cwd sql.NullString
				var updated sql.NullInt64
				if rows.Scan(&id, &title, &cwd, &updated) != nil || id == "" {
					continue
				}
				entry := sessionIndexEntry{ID: id, ThreadName: title.String, CWD: cwd.String}
				if updated.Valid && updated.Int64 > 0 {
					entry.UpdatedAt = time.Unix(updated.Int64, 0).UTC().Format(time.RFC3339Nano)
				}
				if previous, ok := result[id]; !ok || indexEntryNewer(entry, previous) {
					result[id] = entry
				}
			}
			_ = rows.Close()
			_ = db.Close()
		}
	}
	return result
}

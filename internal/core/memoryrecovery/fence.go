package memoryrecovery

import (
 "crypto/rand"
 "encoding/hex"
 "errors"
 "fmt"
 "os"
 "path/filepath"
 "runtime"
 "time"
)

var ErrLegacyWriteFenced = errors.New("legacy memory write fenced by recovery journal")

// WithLegacyWriter serializes cooperative legacy memory writers with the
// journal's .active.lock via O_EXCL across processes sharing the same DataDir.
// Unlike a mere preflight check, the lock is held THROUGH the database or
// config write. No timeout removes locks after process crashes.
func WithLegacyWriter(dataDir string, write func() error) error {
 if write == nil { return errors.New("legacy memory writer callback required") }
 root := filepath.Join(dataDir, "memory-recovery-journal")
 if err := os.MkdirAll(root, 0700); err != nil { return err }
 info, err := os.Lstat(root)
 if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
  return errors.New("unsafe legacy memory fence directory")
 }
 nonce := make([]byte, 16)
 if _, err := rand.Read(nonce); err != nil { return err }
 owner := hex.EncodeToString(nonce)
 lockPath := filepath.Join(root, ".active.lock")
 var lock *os.File
 // Bounded retries allow normal short writes by other Nexus processes to
 // finish. A recovery journal or orphan lock remains fail-closed.
 for attempt := 0; attempt < 40; attempt++ {
  lock, err = os.OpenFile(lockPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
  if err == nil { break }
  // Windows file sharing can surface a contended lock as access denied
  // instead of EEXIST. Retry only within the same strict time bound.
  contention := os.IsExist(err) || (runtime.GOOS=="windows" && os.IsPermission(err))
  if !contention { return fmt.Errorf("%w: %v", ErrLegacyWriteFenced, err) }
  if attempt < 39 { time.Sleep(25 * time.Millisecond) }
 }
 if lock == nil { return ErrLegacyWriteFenced }
 if _, err := lock.WriteString(owner); err != nil { _ = lock.Close(); return err }
 if err := lock.Sync(); err != nil { _ = lock.Close(); return err }
 if err := lock.Close(); err != nil { return err }

 // We own the lock, so Inspect().Blocked is expected. However, any
 // incomplete journal, unknown entry or broken hash chain is not.
 inspection, err := Inspect(dataDir)
 if err != nil || len(inspection.Problems) != 0 || unfinished(inspection.Operations) {
  return errors.Join(ErrLegacyWriteFenced, releaseWriterLock(lockPath, owner))
 }
 // This is the only point at which the protected write may execute.
 writeErr := write()
 unlockErr := releaseWriterLock(lockPath, owner)
 return errors.Join(writeErr, unlockErr)
}

func releaseWriterLock(lockPath, owner string) error {
 info, err := os.Lstat(lockPath)
 if err != nil || !info.Mode().IsRegular() || info.Size() != 32 {
  return errors.New("legacy memory fence lock missing or replaced; human review required")
 }
 content, err := os.ReadFile(lockPath)
 if err != nil || string(content) != owner {
  return errors.New("legacy memory fence owner changed; human review required")
 }
 if err := os.Remove(lockPath); err != nil {
  return fmt.Errorf("release legacy memory fence: %w", err)
 }
 return nil
}

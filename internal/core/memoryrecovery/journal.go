// Package memoryrecovery journals the operational steps of a potential
// legacy-memory restore. It never stores or changes semantic memory itself.
package memoryrecovery

import (
 "crypto/rand"
 "crypto/sha256"
 "encoding/hex"
 "encoding/json"
 "errors"
 "fmt"
 "io"
 "os"
 "path/filepath"
 "regexp"
 "runtime"
 "sort"
 "strings"
 "time"
)

type Stage string
const (
 Prepared Stage = "PREPARED"
 ProjectCommitted Stage = "PROJECT_COMMITTED"
 InstallCommitted Stage = "INSTALL_COMMITTED"
 Verified Stage = "VERIFIED"
 Aborted Stage = "ABORTED"
 NeedsRecovery Stage = "NEEDS_RECOVERY"
)

const (
 maxBackupBytes = 16 << 20
 maxJournalBytes = 8192
 maxOperations = 1000
)
var (
 safeID = regexp.MustCompile(`^prj_[A-Za-z0-9_-]{1,120}$`)
 safeDigest = regexp.MustCompile(`^[a-f0-9]{64}$`)
 safeOp = regexp.MustCompile(`^[a-f0-9]{32}$`)
)

type Event struct {
 Version int `json:"version"`
 OperationID string `json:"operation_id"`
 ProjectID string `json:"project_id"`
 BackupSHA256 string `json:"backup_sha256"`
 SourceDigestSHA256 string `json:"source_digest_sha256"`
 Stage Stage `json:"stage"`
 Sequence int `json:"sequence"`
 PreviousSHA256 string `json:"previous_sha256,omitempty"`
 Timestamp time.Time `json:"timestamp"`
 Hash string `json:"hash"`
}

type Operation struct {
 ID string `json:"id"`
 ProjectID string `json:"project_id"`
 Stage Stage `json:"stage"`
 Events int `json:"events"`
 UpdatedAt time.Time `json:"updated_at"`
}
type Inspection struct {
 Operations []Operation `json:"operations"`
 Blocked bool `json:"blocked"`
 LockPresent bool `json:"lock_present"`
 Problems []string `json:"problems"`
}

type Journal struct {
 directory string
 lockPath string
 lock *os.File
 last Event
}

// Inspect has no side effects. Unknown files, an incomplete operation,
// an orphan lock, a malformed checksum or event chain all fail closed.
func Inspect(dataDir string) (Inspection, error) {
 result := Inspection{Operations: []Operation{}, Problems: []string{}}
 root := filepath.Join(dataDir, "memory-recovery-journal")
 info, err := os.Lstat(root)
 if os.IsNotExist(err) { return result, nil }
 if err != nil { return result, err }
 if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
  return Inspection{}, errors.New("journal directory is not a real directory")
 }
 entries, err := os.ReadDir(root)
 if err != nil { return result, err }
 if len(entries) > maxOperations+1 { return Inspection{}, errors.New("journal operation limit exceeded") }
 for _, entry := range entries {
  if entry.Name() == ".active.lock" {
   result.LockPresent = true
   result.Blocked = true
   lockInfo, lockErr := os.Lstat(filepath.Join(root, entry.Name()))
   if lockErr != nil || !lockInfo.Mode().IsRegular() || lockInfo.Size() != 32 {
    result.Problems = append(result.Problems, "INVALID_RECOVERY_LOCK")
   } else {
    owner, readErr := os.ReadFile(filepath.Join(root, entry.Name()))
    if readErr != nil || !safeOp.Match(owner) {
     result.Problems = append(result.Problems, "INVALID_RECOVERY_LOCK")
    }
   }
   continue
  }
  if !safeOp.MatchString(entry.Name()) || !entry.IsDir() {
   result.Problems = append(result.Problems, "UNKNOWN_JOURNAL_ENTRY")
   result.Blocked = true
   continue
  }
  events, err := readEvents(filepath.Join(root, entry.Name()), entry.Name())
  if err != nil {
   result.Problems = append(result.Problems, "CORRUPT_OR_INCOMPLETE_JOURNAL")
   result.Blocked = true
   continue
  }
  last := events[len(events)-1]
  result.Operations = append(result.Operations, Operation{
   ID: last.OperationID, ProjectID: last.ProjectID, Stage: last.Stage,
   Events: len(events), UpdatedAt: last.Timestamp,
  })
  if !terminal(last.Stage) { result.Blocked = true }
 }
 sort.Slice(result.Operations, func(i,j int)bool {return result.Operations[i].ID < result.Operations[j].ID})
 return result, nil
}

// Begin checks a PRIVATE Nexus backup and creates an append-only audit journal.
// It never applies a restore, nor does the lock fence legacy config/SQLite
// writers. Other restore coordinators must honor this same lock.
func Begin(dataDir, projectID, backupPath, backupSHA, sourceSHA string) (*Journal, error) {
 if !safeID.MatchString(projectID) || !safeDigest.MatchString(backupSHA) ||
   !safeDigest.MatchString(sourceSHA) {
  return nil, errors.New("invalid restore journal identity or digest")
 }
 if err := verifyLocalBackup(dataDir, projectID, backupPath, backupSHA); err != nil {
  return nil, err
 }
 root := filepath.Join(dataDir, "memory-recovery-journal")
 if err := os.MkdirAll(root, 0700); err != nil { return nil, err }
 info, err := os.Lstat(root)
 if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
  return nil, errors.New("unsafe journal directory")
 }
 before, err := Inspect(dataDir)
 if err != nil { return nil, err }
 if before.Blocked { return nil, errors.New("unfinished or locked recovery requires human review") }
 nonce := make([]byte, 16)
 if _, err := rand.Read(nonce); err != nil { return nil, err }
 id := hex.EncodeToString(nonce)
 lockPath := filepath.Join(root, ".active.lock")
 file, err := os.OpenFile(lockPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
 if err != nil { return nil, fmt.Errorf("recovery journal is already locked: %w", err) }
 if _, err := file.WriteString(id); err != nil { _ = file.Close(); return nil, err }
 if err := file.Sync(); err != nil { _ = file.Close(); return nil, err }
 if err := syncRecoveryDirectory(root); err != nil {
  _ = file.Close()
  return nil, fmt.Errorf("recovery lock durability unverified: %w", err)
 }
 after, err := Inspect(dataDir)
 if err != nil || len(after.Problems) != 0 || unfinished(after.Operations) {
  _ = file.Close()
  return nil, errors.New("journal changed while acquiring recovery lock")
 }
 opDir := filepath.Join(root,id)
 if err := os.Mkdir(opDir,0700); err != nil {
  _ = file.Close(); _ = os.Remove(lockPath); return nil, err
 }
 if err := syncRecoveryDirectory(root); err != nil {
  _ = file.Close()
  return nil, fmt.Errorf("recovery operation directory durability unverified: %w", err)
 }
 journal := &Journal{directory: opDir,lockPath: lockPath,lock:file}
 first := Event{Version:1,OperationID:id,ProjectID:projectID,
  BackupSHA256:backupSHA,SourceDigestSHA256:sourceSHA,
  Stage:Prepared,Sequence:0,Timestamp:time.Now().UTC()}
 if err := journal.write(first); err != nil {
  // Keep an orphan lock when a write reached the filesystem:
  // operator inspection must decide whether recovery is needed.
  _ = file.Close()
  return nil, err
 }
 return journal,nil
}

func (j *Journal) Advance(next Stage) error {
 if j == nil || j.lock == nil { return errors.New("recovery journal session is not active") }
 if !allowed(j.last.Stage,next) { return errors.New("invalid recovery journal transition") }
 e := j.last
 e.Sequence++
 e.Stage=next
 e.PreviousSHA256=j.last.Hash
 e.Timestamp=time.Now().UTC()
 e.Hash=""
 return j.write(e)
}

func (j *Journal) Close() error {
 if j == nil || j.lock == nil { return nil }
 err := j.lock.Close()
 j.lock=nil
 // Never remove a lock after a partially applied operation or failed
 // journal write. This intentionally blocks any new restore until reviewed.
 if !terminal(j.last.Stage) { return errors.New("unfinished recovery journal; lock retained for review") }
 if err != nil { return err }
 lockInfo, err := os.Lstat(j.lockPath)
 if err != nil || !lockInfo.Mode().IsRegular() {
  return errors.New("recovery lock disappeared or changed type")
 }
 owner, err := os.ReadFile(j.lockPath)
 if err != nil || string(owner) != j.last.OperationID {
  return errors.New("recovery lock ownership changed; manual review required")
 }
 return os.Remove(j.lockPath)
}

func allowed(previous, next Stage) bool {
 switch previous {
 case Prepared: return next==ProjectCommitted || next==Aborted || next==NeedsRecovery
 case ProjectCommitted: return next==InstallCommitted || next==NeedsRecovery
 case InstallCommitted: return next==Verified || next==NeedsRecovery
 default: return false
 }
}
func terminal(stage Stage) bool { return stage==Verified || stage==Aborted }
func unfinished(ops []Operation) bool {
 for _, op := range ops { if !terminal(op.Stage) { return true } }
 return false
}
func eventHash(e Event) string {
 e.Hash=""
 bytes,_:=json.Marshal(e)
 sum:=sha256.Sum256(bytes)
 return hex.EncodeToString(sum[:])
}

func (j *Journal) write(event Event) error {
 event.Hash=eventHash(event)
 bytes,err:=json.MarshalIndent(event,"","  ")
 if err!=nil{return err}
 if len(bytes)>maxJournalBytes{return errors.New("journal event too large")}
 filename:=filepath.Join(j.directory,fmt.Sprintf("%04d.json",event.Sequence))
 f,err:=os.OpenFile(filename,os.O_CREATE|os.O_EXCL|os.O_WRONLY,0600)
 if err!=nil{return err}
 if _,err=f.Write(bytes);err!=nil {f.Close();return err}
 if err=f.Sync();err!=nil {f.Close();return err}
 if err=f.Close();err!=nil{return err}
 // Syncing the parent directory strengthens rename/create durability on
 // POSIX. Windows directory sync support is platform-specific.
 if err := syncRecoveryDirectory(j.directory); err != nil {
  return fmt.Errorf("journal event durability unverified: %w", err)
 }
 j.last=event
 return nil
}
func readEvents(dir,id string)([]Event,error){
 list,err:=os.ReadDir(dir);if err!=nil{return nil,err}
 if len(list)==0||len(list)>6{return nil,errors.New("missing or excessive events")}
 events:=make([]Event,0,len(list))
 for i,e:=range list{
  if e.IsDir()||e.Name()!=fmt.Sprintf("%04d.json",i){return nil,errors.New("journal event order invalid")}
  p:=filepath.Join(dir,e.Name())
  info,err:=os.Lstat(p)
  if err!=nil||!info.Mode().IsRegular()||info.Size()<1||info.Size()>maxJournalBytes {
   return nil,errors.New("journal event not a bounded regular file")
  }
  bytes,err:=os.ReadFile(p);if err!=nil{return nil,err}
  var evt Event
  if err=json.Unmarshal(bytes,&evt);err!=nil{return nil,err}
  if evt.Version!=1||evt.OperationID!=id||evt.Sequence!=i||
    !safeID.MatchString(evt.ProjectID)||!safeDigest.MatchString(evt.BackupSHA256)||
    !safeDigest.MatchString(evt.SourceDigestSHA256)||evt.Hash!=eventHash(evt) {
   return nil,errors.New("journal event integrity invalid")
  }
  if i==0 {
   if evt.Stage!=Prepared||evt.PreviousSHA256!="" {return nil,errors.New("journal lacks prepared event")}
  } else {
   previous:=events[len(events)-1]
   if evt.ProjectID!=previous.ProjectID||evt.BackupSHA256!=previous.BackupSHA256||
      evt.SourceDigestSHA256!=previous.SourceDigestSHA256||
      evt.PreviousSHA256!=previous.Hash||!allowed(previous.Stage,evt.Stage){
    return nil,errors.New("journal chain or transition invalid")
   }
  }
  events=append(events,evt)
 }
 return events,nil
}

func verifyLocalBackup(dataDir,projectID,backupPath,expectedSHA string)error{
 root:=filepath.Join(dataDir,"migration-backups")
 absolute,err:=filepath.Abs(backupPath);if err!=nil{return err}
 if filepath.Dir(absolute)!=filepath.Clean(root)||
  !strings.HasPrefix(filepath.Base(absolute),"legacy-intelligence-")||
  !strings.HasSuffix(filepath.Base(absolute),".json"){
  return errors.New("backup must be in Nexus-owned migration-backups directory")
 }
 parent,err:=os.Lstat(root)
 if err!=nil||!parent.IsDir()||parent.Mode()&os.ModeSymlink!=0 {
  return errors.New("unsafe backup directory")
 }
 info,err:=os.Lstat(absolute)
 if err!=nil||!info.Mode().IsRegular()||info.Size()<1||info.Size()>maxBackupBytes{
  return errors.New("invalid local backup")
 }
 if runtime.GOOS!="windows"&&info.Mode().Perm()&0077!=0{
  return errors.New("backup file permissions too broad")
 }
 f,err:=os.Open(absolute);if err!=nil{return err}
 defer f.Close()
 h:=sha256.New()
 // Bound the actual read, not just the initial stat, against file growth.
 raw,err:=io.ReadAll(io.LimitReader(io.TeeReader(f,h),maxBackupBytes+1))
 if err!=nil{return err}
 if len(raw)>maxBackupBytes{return errors.New("backup exceeded size limit while reading")}
 if hex.EncodeToString(h.Sum(nil))!=expectedSHA{return errors.New("backup checksum mismatch")}
 var identity struct {
  SchemaVersion int `json:"schema_version"`
  ProjectID string `json:"project_id"`
  ProjectPath string `json:"project_path"`
 }
 if err=json.Unmarshal(raw,&identity);err!=nil{return errors.New("invalid backup JSON")}
 if identity.SchemaVersion!=1||identity.ProjectID!=projectID||identity.ProjectPath==""{
  return errors.New("backup belongs to a different project or schema")
 }
 return nil
}

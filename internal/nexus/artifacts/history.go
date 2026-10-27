package artifacts

import (
 "crypto/sha256"
 "encoding/hex"
 "encoding/json"
 "errors"
 "fmt"
 "io"
 "net/http"
 "os"
 "path/filepath"
 "sort"
 "strconv"
 "sync"
 "time"
)

const maxSnapshotSize=int64(24<<20)
const maxSnapshotProjectBytes=int64(256<<20)
const maxRevisions=5

var ledgerMu sync.Mutex

type Revision struct{
 Number int `json:"number"`
 SHA256 string `json:"sha256"`
 Size int64 `json:"size"`
 Modified time.Time `json:"modified"`
 Snapshot bool `json:"snapshot"`
}
type Record struct{
 Path string `json:"path"`
 AgentID string `json:"agent_id,omitempty"`
 RuntimeID string `json:"runtime_id,omitempty"`
 Origin string `json:"origin"`
 Versions []Revision `json:"versions"`
}
type ledger struct{Records map[string]Record `json:"records"`}
func ledgerDirectory(dataDir,projectID string)string{
 h:=sha256.Sum256([]byte(projectID))
 return filepath.Join(dataDir,"outputs","projects",hex.EncodeToString(h[:16]))
}
func ledgerPath(dataDir,projectID string)string{return filepath.Join(ledgerDirectory(dataDir,projectID),"index.json")}
func openLedger(dataDir,projectID string)(ledger,error){
 state:=ledger{Records:map[string]Record{}}
 body,err:=os.ReadFile(ledgerPath(dataDir,projectID))
 if errors.Is(err,os.ErrNotExist){return state,nil}
 if err!=nil{return state,err}
 if err:=json.Unmarshal(body,&state);err!=nil{return state,err}
 if state.Records==nil{state.Records=map[string]Record{}}
 return state,nil
}
func persistLedger(dataDir,projectID string,state ledger)error{
 dir:=ledgerDirectory(dataDir,projectID)
 if err:=os.MkdirAll(dir,0700);err!=nil{return err}
 b,err:=json.MarshalIndent(state,"","  ");if err!=nil{return err}
 tmp,err:=os.CreateTemp(dir,"index-*.tmp");if err!=nil{return err}
 _=tmp.Chmod(0600)
 if _,err=tmp.Write(b);err!=nil{_ =tmp.Close();_ =os.Remove(tmp.Name());return err}
 if err=tmp.Close();err!=nil{_ =os.Remove(tmp.Name());return err}
 if err=os.Rename(tmp.Name(),ledgerPath(dataDir,projectID));err!=nil{_ =os.Remove(tmp.Name());return err}
 return nil
}
func snapshotFilename(itemID string,rev int)string{return fmt.Sprintf("%s-v%d.bin",itemID,rev)}
func snapshotDirectory(dataDir,projectID string)string{return filepath.Join(ledgerDirectory(dataDir,projectID),"versions")}
func trackedBytes(state ledger)int64{
 var size int64
 for _,record:=range state.Records{for _,rev:=range record.Versions{if rev.Snapshot{size+=rev.Size}}}
 return size
}
func snapshotCurrent(root *os.Root,dataDir,projectID string,item Item,revision int)bool{
 if item.Size<0||item.Size>maxSnapshotSize||item.SHA256==""{return false}
 file,info,_,_,err:=inspected(root,item.Path);if err!=nil{return false};defer file.Close()
 if info.Size()!=item.Size||!info.ModTime().Equal(item.Modified){return false}
 dir:=snapshotDirectory(dataDir,projectID)
 if os.MkdirAll(dir,0700)!=nil{return false}
 tmp,err:=os.CreateTemp(dir,"version-*.tmp");if err!=nil{return false}
 _=tmp.Chmod(0600)
 h:=sha256.New()
 count,err:=io.Copy(io.MultiWriter(tmp,h),io.LimitReader(file,maxSnapshotSize+1))
 _=tmp.Close()
 if err!=nil||count!=item.Size||hex.EncodeToString(h.Sum(nil))!=item.SHA256{_ =os.Remove(tmp.Name());return false}
 dst:=filepath.Join(dir,snapshotFilename(item.ID,revision))
 if err:=os.Rename(tmp.Name(),dst);err!=nil{_ =os.Remove(tmp.Name());return false}
 return true
}
// Observe records deterministic revisions of discovered project outputs.
// Called on selected scan results. The project-scoped cache has strict bounds;
// no original is deleted or rewritten. Agent provenance is only assigned when
// the caller authenticates a runtime-specific publisher.
func Observe(root *os.Root,dataDir,projectID string,item Item,origin,agentID,runtimeID string)(Item,error){
 if projectID==""||item.Path==""{return item,ErrForbidden}
 ledgerMu.Lock();defer ledgerMu.Unlock()
 state,err:=openLedger(dataDir,projectID);if err!=nil{return item,err}
 old:=state.Records[item.Path]
 if origin==""{origin="filesystem"}
 latestHash:=""
 if len(old.Versions)>0 {latestHash=old.Versions[len(old.Versions)-1].SHA256}
 sameContent:=len(old.Versions)>0 && ((item.SHA256!="" && item.SHA256==latestHash) || (item.SHA256=="" && old.Versions[len(old.Versions)-1].Size==item.Size && old.Versions[len(old.Versions)-1].Modified.Equal(item.Modified)))
 if sameContent {
  if agentID==""{agentID=old.AgentID}
  if runtimeID==""{runtimeID=old.RuntimeID}
  if old.Origin=="agent"&&origin=="filesystem"{origin=old.Origin}
 }
 latest:=Revision{}
 if len(old.Versions)>0{latest=old.Versions[len(old.Versions)-1]}
 if sameContent && latest.Size==item.Size{
  if old.AgentID!=agentID||old.RuntimeID!=runtimeID||old.Origin!=origin{
   old.AgentID=agentID;old.RuntimeID=runtimeID;old.Origin=origin
   state.Records[item.Path]=old
   if err=persistLedger(dataDir,projectID,state);err!=nil{return item,err}
  }
  item.Origin=old.Origin;item.AgentID=old.AgentID;item.RuntimeID=old.RuntimeID
  item.Revision=latest.Number;item.Revisions=len(old.Versions)
  return item,nil
 }
 rev:=latest.Number+1
 if rev==1{rev=1}
 isSnapshot:=false
 if trackedBytes(state)+item.Size<=maxSnapshotProjectBytes{isSnapshot=snapshotCurrent(root,dataDir,projectID,item,rev)}
 next:=Revision{Number:rev,SHA256:item.SHA256,Size:item.Size,Modified:item.Modified,Snapshot:isSnapshot}
 versions:=append(old.Versions,next)
 if len(versions)>maxRevisions{
  removed:=versions[:len(versions)-maxRevisions]
  for _,v:=range removed{if v.Snapshot{_ =os.Remove(filepath.Join(snapshotDirectory(dataDir,projectID),snapshotFilename(item.ID,v.Number)))}}
  versions=versions[len(versions)-maxRevisions:]
 }
 state.Records[item.Path]=Record{Path:item.Path,AgentID:agentID,RuntimeID:runtimeID,Origin:origin,Versions:versions}
 if err=persistLedger(dataDir,projectID,state);err!=nil{return item,err}
 item.Origin=origin;item.AgentID=agentID;item.RuntimeID=runtimeID
 item.Revision=rev;item.Revisions=len(versions)
 return item,nil
}
func History(dataDir,projectID,rel string)(Record,error){
 if _,err:=SafeRelative(rel);err!=nil{return Record{},err}
 ledgerMu.Lock();defer ledgerMu.Unlock()
 state,err:=openLedger(dataDir,projectID);if err!=nil{return Record{},err}
 rec,ok:=state.Records[rel];if !ok{return Record{},os.ErrNotExist}
 return rec,nil
}
func ServeRevision(w http.ResponseWriter,r *http.Request,dataDir,projectID,rel string,number int)error{
 if number<1{return ErrForbidden}
 safe,err:=SafeRelative(rel);if err!=nil||safe=="."{return ErrForbidden}
 ledgerMu.Lock()
 state,err:=openLedger(dataDir,projectID)
 if err!=nil{ledgerMu.Unlock();return err}
 record,ok:=state.Records[safe]
 var revision Revision
 if ok{for _,v:=range record.Versions{if v.Number==number{revision=v;break}}}
 ledgerMu.Unlock()
 if revision.Number!=number||!revision.Snapshot{return os.ErrNotExist}
 id:=sha256.Sum256([]byte(safe));fileID:=hex.EncodeToString(id[:12])
 dir:=snapshotDirectory(dataDir,projectID)
 // Use a second Root for the cache to prevent traversal or symlink attacks.
 root,err:=os.OpenRoot(dir);if err!=nil{return err};defer root.Close()
 f,err:=root.Open(snapshotFilename(fileID,number));if err!=nil{return err};defer f.Close()
 st,err:=f.Stat();if err!=nil||!st.Mode().IsRegular()||st.Size()!=revision.Size{return ErrForbidden}
 h:=sha256.New();if _,err:=io.Copy(h,f);err!=nil||hex.EncodeToString(h.Sum(nil))!=revision.SHA256{return ErrForbidden}
 _,_ =f.Seek(0,io.SeekStart)
 mime:="application/octet-stream";if _,m:=class(filepath.Base(safe));m!=""{mime=m}
 w.Header().Set("Content-Type",mime)
 w.Header().Set("Cache-Control","private, no-store")
 w.Header().Set("X-Content-Type-Options","nosniff")
 w.Header().Set("Content-Security-Policy","sandbox")
 w.Header().Set("Content-Disposition","attachment; filename="+strconv.Quote(filepath.Base(safe)))
 http.ServeContent(w,r,filepath.Base(safe),revision.Modified,f)
 return nil
}
func ListRevisions(dataDir,projectID,rel string)([]Revision,error){
 rec,err:=History(dataDir,projectID,rel);if err!=nil{return nil,err}
 out:=append([]Revision(nil),rec.Versions...)
 sort.Slice(out,func(i,j int)bool{return out[i].Number>out[j].Number})
 return out,nil
}

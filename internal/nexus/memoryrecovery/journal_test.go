package memoryrecovery

import (
 "crypto/sha256"
 "encoding/hex"
 "encoding/json"
 "os"
 "path/filepath"
 "strings"
 "testing"
)

func journalFixture(t *testing.T)(string,string,string){
 t.Helper()
 root:=t.TempDir()
 dir:=filepath.Join(root,"migration-backups")
 if err:=os.MkdirAll(dir,0700);err!=nil{t.Fatal(err)}
 name:=filepath.Join(dir,"legacy-intelligence-unit.json")
 body:=[]byte(`{"schema_version":1,"project_id":"prj_test","project_path":"/repo/test","install":[],"project":[]}`)
 if err:=os.WriteFile(name,body,0600);err!=nil{t.Fatal(err)}
 sum:=sha256.Sum256(body)
 return root,name,hex.EncodeToString(sum[:])
}

func TestJournalStaysMetadataOnlyAndReplaysValidHistory(t *testing.T){
 root,backup,hash:=journalFixture(t)
 j,err:=Begin(root,"prj_test",backup,hash,strings.Repeat("a",64))
 if err!=nil{t.Fatal(err)}
 initial,err:=Inspect(root)
 if err!=nil{t.Fatal(err)}
 if !initial.Blocked||!initial.LockPresent||len(initial.Operations)!=1||
  initial.Operations[0].Stage!=Prepared{t.Fatalf("bad initial journal %+v",initial)}
 if err=j.Advance(ProjectCommitted);err!=nil{t.Fatal(err)}
 if err=j.Advance(InstallCommitted);err!=nil{t.Fatal(err)}
 if err=j.Advance(Verified);err!=nil{t.Fatal(err)}
 if err=j.Close();err!=nil{t.Fatal(err)}
 result,err:=Inspect(root)
 if err!=nil{t.Fatal(err)}
 if result.Blocked||result.LockPresent||len(result.Operations)!=1||
  result.Operations[0].Stage!=Verified||result.Operations[0].Events!=4{
  t.Fatalf("final journal not verified: %+v",result)
 }
 events,err:=readEvents(filepath.Join(root,"memory-recovery-journal",result.Operations[0].ID),result.Operations[0].ID)
 if err!=nil{t.Fatal(err)}
 if events[3].PreviousSHA256!=events[2].Hash{t.Fatal("hash-chain link lost")}
 raw,err:=os.ReadFile(filepath.Join(root,"memory-recovery-journal",result.Operations[0].ID,"0001.json"))
 if err!=nil{t.Fatal(err)}
 for _,private:=range []string{"/repo/test","legacy-intelligence-unit.json","install\":","project\":"}{
  if strings.Contains(string(raw),private){t.Fatalf("journal exposed sensitive backup contents: %s",private)}
 }
}

func TestJournalRejectsInvalidTransitionsAndAllowsPreMutationAbort(t *testing.T){
 root,backup,hash:=journalFixture(t)
 j,err:=Begin(root,"prj_test",backup,hash,strings.Repeat("b",64))
 if err!=nil{t.Fatal(err)}
 if err=j.Advance(Verified);err==nil{t.Fatal("skipped stages accepted")}
 if err=j.Advance(Aborted);err!=nil{t.Fatal(err)}
 if err=j.Advance(ProjectCommitted);err==nil{t.Fatal("terminal journal was advanced")}
 if err=j.Close();err!=nil{t.Fatal(err)}
 if status,err:=Inspect(root);err!=nil||status.Blocked{t.Fatalf("clean abortion blocked: %+v %v",status,err)}
}

func TestJournalCrashLeavesLockAndBlocksNewOperations(t *testing.T){
 root,backup,hash:=journalFixture(t)
 j,err:=Begin(root,"prj_test",backup,hash,strings.Repeat("c",64))
 if err!=nil{t.Fatal(err)}
 if err:=j.Advance(ProjectCommitted);err!=nil{t.Fatal(err)}
 if err:=j.Close();err==nil{t.Fatal("incomplete operation did not retain lock")}
 state,err:=Inspect(root)
 if err!=nil||!state.Blocked||!state.LockPresent{t.Fatalf("missing recovery lock: %+v %v",state,err)}
 if _,err:=Begin(root,"prj_test",backup,hash,strings.Repeat("c",64));err==nil{
  t.Fatal("second operation bypassed unfinished restore")
 }
}

func TestJournalRejectsTamperingEvenWhenLockAbsent(t *testing.T){
 root,backup,hash:=journalFixture(t)
 j,err:=Begin(root,"prj_test",backup,hash,strings.Repeat("d",64))
 if err!=nil{t.Fatal(err)}
 if err:=j.Advance(Aborted);err!=nil{t.Fatal(err)}
 id:=j.last.OperationID
 if err:=j.Close();err!=nil{t.Fatal(err)}
 file:=filepath.Join(root,"memory-recovery-journal",id,"0001.json")
 data,err:=os.ReadFile(file)
 if err!=nil{t.Fatal(err)}
 var payload map[string]any
 if err:=json.Unmarshal(data,&payload);err!=nil{t.Fatal(err)}
 payload["stage"]="VERIFIED"
 modified,err:=json.Marshal(payload)
 if err!=nil{t.Fatal(err)}
 if err:=os.WriteFile(file,modified,0600);err!=nil{t.Fatal(err)}
 state,err:=Inspect(root)
 if err!=nil||!state.Blocked||len(state.Problems)!=1{t.Fatalf("corrupt journal accepted: %+v %v",state,err)}
}

func TestJournalBackupIsBoundToProjectAndContentHash(t *testing.T){
 root,backup,hash:=journalFixture(t)
 for _,input:=range []struct{project,path,digest string}{
  {"prj_other",backup,hash},
  {"prj_test",backup,strings.Repeat("0",64)},
  {"../project",backup,hash},
  {"prj_test",filepath.Join(t.TempDir(),"legacy-intelligence-unit.json"),hash},
 }{
  if _,err:=Begin(root,input.project,input.path,input.digest,strings.Repeat("e",64));err==nil{
   t.Fatalf("unsafe backup accepted: %+v",input)
  }
 }
 status,err:=Inspect(root)
 if err!=nil||status.Blocked||len(status.Operations)!=0{t.Fatalf("rejected preflight made journal: %+v %v",status,err)}
}

func TestJournalRejectsUnknownEntries(t *testing.T){
 root,_,_:=journalFixture(t)
 dir:=filepath.Join(root,"memory-recovery-journal")
 if err:=os.MkdirAll(dir,0700);err!=nil{t.Fatal(err)}
 if err:=os.WriteFile(filepath.Join(dir,"unknown-entry"),[]byte("bad"),0600);err!=nil{t.Fatal(err)}
 info,err:=Inspect(root)
 if err!=nil||!info.Blocked||len(info.Problems)!=1{t.Fatalf("unsafe recovery ignored: %+v %v",info,err)}
}

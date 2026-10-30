package store

import (
 "errors"
 "os"
 "path/filepath"
 "strings"
 "testing"

 "github.com/IAPro-Community/nexus/internal/core/memoryrecovery"
)

func TestProjectMemoryWritersRespectRecoveryFence(t *testing.T) {
 dir:=t.TempDir()
 s,err:=Open(filepath.Join(dir,"nexus.db"))
 if err!=nil{t.Fatal(err)}
 defer s.Close()
 project,err:=s.CreateProject(Project{Name:"Fence test",CanonicalPath:t.TempDir()})
 if err!=nil{t.Fatal(err)}
 original,err:=s.UpsertIntelligenceMemory(IntelligenceMemory{
  ProjectID:project.ID,Key:"quality.testing",Value:"run go test",Source:"explicit",
 })
 if err!=nil{t.Fatal(err)}

 root:=filepath.Join(dir,"memory-recovery-journal")
 if err:=os.MkdirAll(root,0700);err!=nil{t.Fatal(err)}
 lock:=filepath.Join(root,".active.lock")
 if err:=os.WriteFile(lock,[]byte(strings.Repeat("a",32)),0600);err!=nil{t.Fatal(err)}
 if _,err:=s.UpsertIntelligenceMemory(IntelligenceMemory{
  ProjectID:project.ID,Key:"quality.testing",Value:"unsafe overwrite",
 });!errors.Is(err,memoryrecovery.ErrLegacyWriteFenced){
  t.Fatalf("fence did not block upsert: %v",err)
 }
 if err:=s.DeleteIntelligenceMemory(project.ID,original.ID);!errors.Is(err,memoryrecovery.ErrLegacyWriteFenced){
  t.Fatalf("fence did not block delete: %v",err)
 }
 if err:=s.DeleteIntelligenceMemoryByKey(project.ID,original.Key);!errors.Is(err,memoryrecovery.ErrLegacyWriteFenced){
  t.Fatalf("fence did not block delete by key: %v",err)
 }
 row,err:=s.GetIntelligenceMemoryByKey(project.ID,original.Key)
 if err!=nil || row.Value!=original.Value {t.Fatalf("fenced project memory mutated: %+v %v",row,err)}
 if err:=os.Remove(lock);err!=nil{t.Fatal(err)}
 if err:=s.DeleteIntelligenceMemory(project.ID,original.ID);err!=nil{t.Fatal(err)}
 if _,err:=s.GetIntelligenceMemoryByKey(project.ID,original.Key);err==nil{
  t.Fatal("delete after fence release did not run")
 }
}

func TestProjectMemoryUsesOwnDatabaseDirectoryForRecoveryFence(t *testing.T) {
 dir:=t.TempDir()
 t.Setenv("NEXUS_DATA_DIR",t.TempDir())
 s,err:=Open(filepath.Join(dir,"nexus.db"))
 if err!=nil{t.Fatal(err)}
 defer s.Close()
 project,err:=s.CreateProject(Project{Name:"Scoped store",CanonicalPath:t.TempDir()})
 if err!=nil{t.Fatal(err)}
 if _,err:=s.UpsertIntelligenceMemory(IntelligenceMemory{
  ProjectID:project.ID,Key:"quality.testing",Value:"go test",
 });err!=nil{t.Fatal(err)}
 info,err:=os.Stat(filepath.Join(dir,"memory-recovery-journal"))
 if err!=nil||!info.IsDir(){t.Fatalf("fence not co-located with store: %v",err)}
}

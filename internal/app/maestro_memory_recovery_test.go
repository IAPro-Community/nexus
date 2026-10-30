package app

import (
 "os"
 "path/filepath"
 "strings"
 "testing"

 "github.com/IAPro-Community/nexus/internal/nexus/memoryrecovery"
)

func TestMaestroRecoveryStatusIsReadOnlyAndNoJournalIsClean(t *testing.T) {
 dir:=t.TempDir()
 t.Setenv("NEXUS_DATA_DIR",dir)
 if err:=maestroMemoryRecoveryStatusCmd([]string{"--json"});err!=nil{t.Fatal(err)}
 if _,err:=os.Stat(filepath.Join(dir,"memory-recovery-journal"));!os.IsNotExist(err){
  t.Fatalf("read-only status created a journal directory: %v",err)
 }
}

func TestMaestroRecoveryStatusRejectsInvalidParameters(t *testing.T) {
 t.Setenv("NEXUS_DATA_DIR",t.TempDir())
 for _,args:=range [][]string{
  {"--unknown"}, {"--json","--json"}, {"--project-id","prj_one"},
  {"--unlock"}, {"--apply"}, {"start"},
 }{
  if err:=maestroMemoryRecoveryStatusCmd(args);err==nil{
   t.Fatalf("status unexpectedly accepted %q",args)
  }
 }
}

func TestMaestroRecoveryStatusRejectsCorruptJournalWithoutDeleting(t *testing.T) {
 dir:=t.TempDir()
 t.Setenv("NEXUS_DATA_DIR",dir)
 root:=filepath.Join(dir,"memory-recovery-journal")
 if err:=os.MkdirAll(root,0700);err!=nil{t.Fatal(err)}
 file:=filepath.Join(root,"corrupt-record")
 if err:=os.WriteFile(file,[]byte("not a journal"),0600);err!=nil{t.Fatal(err)}
 if err:=maestroMemoryRecoveryStatusCmd([]string{"--json"});err!=nil{t.Fatal(err)}
 report,err:=memoryrecovery.Inspect(dir)
 if err!=nil||!report.Blocked||!strings.Contains(strings.Join(report.Problems,","),"UNKNOWN_JOURNAL_ENTRY"){
  t.Fatalf("corrupt records must fail closed without deletion: %+v %v",report,err)
 }
 if _,err:=os.Stat(file);err!=nil{t.Fatalf("status deleted corrupt evidence: %v",err)}
}

package memoryrecovery

import (
 "errors"
 "fmt"
 "os"
 "os/exec"
 "path/filepath"
 "strings"
 "sync"
 "testing"
 "time"
)

func TestWriterFenceSerializesTwoIndependentWriters(t *testing.T) {
 root:=t.TempDir()
 entered:=make(chan struct{})
 release:=make(chan struct{})
 done:=make(chan error,1)
 go func() {
  done<-WithLegacyWriter(root,func()error{close(entered);<-release;return nil})
 }()
 <-entered
 second:=WithLegacyWriter(root,func()error{
  return errors.New("second callback should not run while lock held")
 })
 if !errors.Is(second,ErrLegacyWriteFenced) {t.Fatalf("simultaneous writer was admitted: %v",second)}
 close(release)
 if err:=<-done;err!=nil{t.Fatal(err)}
 if err:=WithLegacyWriter(root,func()error{return nil});err!=nil{
  t.Fatalf("writer lock not released: %v",err)
 }
}

func TestWriterFenceBlocksWhileRestoreJournalIsOpen(t *testing.T) {
 root,backup,hash:=journalFixture(t)
 j,err:=Begin(root,"prj_test",backup,hash,strings.Repeat("d",64))
 if err!=nil{t.Fatal(err)}
 called:=false
 err=WithLegacyWriter(root,func()error{called=true;return nil})
 if !errors.Is(err,ErrLegacyWriteFenced)||called{
  t.Fatalf("writer bypassed prepared recovery journal: %v",err)
 }
 if err:=j.Advance(Aborted);err!=nil{t.Fatal(err)}
 if err:=j.Close();err!=nil{t.Fatal(err)}
 if err:=WithLegacyWriter(root,func()error{called=true;return nil});err!=nil||!called{
  t.Fatalf("writer did not recover after safe abort: %v",err)
 }
}

func TestWriterFenceFailClosedWhenJournalIsUnfinishedWithoutLock(t *testing.T) {
 root,backup,hash:=journalFixture(t)
 j,err:=Begin(root,"prj_test",backup,hash,strings.Repeat("e",64))
 if err!=nil{t.Fatal(err)}
 if err:=j.Advance(ProjectCommitted);err!=nil{t.Fatal(err)}
 if err:=j.Close();err==nil{t.Fatal("unfinished journal erroneously released")}
 lock:=filepath.Join(root,"memory-recovery-journal",".active.lock")
 // Simulates an external operator deleting the lock without repairing
 // a still-incomplete journal: the persisted event chain must still block.
 if err:=os.Remove(lock);err!=nil{t.Fatal(err)}
 called:=false
 err=WithLegacyWriter(root,func()error{called=true;return nil})
 if !errors.Is(err,ErrLegacyWriteFenced)||called{
  t.Fatalf("incomplete journal allowed mutation: %v",err)
 }
}

func TestWriterFenceDoesNotAutomaticallyClearOrphanLock(t *testing.T) {
 root:=t.TempDir()
 dir:=filepath.Join(root,"memory-recovery-journal")
 if err:=os.MkdirAll(dir,0700);err!=nil{t.Fatal(err)}
 lock:=filepath.Join(dir,".active.lock")
 if err:=os.WriteFile(lock,[]byte(strings.Repeat("a",32)),0600);err!=nil{t.Fatal(err)}
 before,_:=os.Stat(lock)
 err:=WithLegacyWriter(root,func()error{return nil})
 if !errors.Is(err,ErrLegacyWriteFenced){t.Fatalf("stale lock was silently reclaimed: %v",err)}
 after,e:=os.Stat(lock)
 if e!=nil||after.ModTime()!=before.ModTime(){t.Fatal("stale lock modified without owner consent")}
}

func TestWriterFenceCallbackErrorStillReleasesOwnedLock(t *testing.T) {
 root:=t.TempDir()
 want:=errors.New("intentional write failure")
 if err:=WithLegacyWriter(root,func()error{return want});!errors.Is(err,want){t.Fatalf("callback failure lost: %v",err)}
 if err:=WithLegacyWriter(root,func()error{return nil});err!=nil{t.Fatal(err)}
}

func TestWriterFenceRejectsCorruptedJournal(t *testing.T) {
 root:=t.TempDir()
 dir:=filepath.Join(root,"memory-recovery-journal")
 if err:=os.MkdirAll(dir,0700);err!=nil{t.Fatal(err)}
 if err:=os.WriteFile(filepath.Join(dir,"unknown"),[]byte("broken"),0600);err!=nil{t.Fatal(err)}
 called:=false
 if err:=WithLegacyWriter(root,func()error{called=true;return nil});!errors.Is(err,ErrLegacyWriteFenced)||called{
  t.Fatalf("corrupt journal admitted a writer: %v",err)
 }
}

func TestWriterFenceCrossProcessStyleConcurrentEntrants(t *testing.T){
 root:=t.TempDir()
 const workers=8
 var group sync.WaitGroup
 entered:=make(chan struct{},workers)
 release:=make(chan struct{})
 for i:=0;i<workers;i++{
  group.Add(1)
  go func(){
   defer group.Done()
   _=WithLegacyWriter(root,func()error{
    entered<-struct{}{}
    <-release
    return nil
   })
  }()
 }
 <-entered
 select{
 case <-entered: t.Fatal("concurrent writers entered same exclusive section")
 case <-time.After(70*time.Millisecond):
 }
 close(release)
 group.Wait()
}

func TestWriterFenceSubprocessHelper(t *testing.T){
 if os.Getenv("NEXUS_FENCE_SUBPROCESS_HELPER")!="1"{return}
 dir:=os.Getenv("NEXUS_FENCE_SUBPROCESS_DIR")
 if dir==""{t.Fatal("missing helper directory")}
 if err:=WithLegacyWriter(dir,func()error{return nil});err!=nil{
  t.Fatalf("subprocess fence refused: %v",err)
 }
}

func TestWriterFenceHoldsAcrossIndependentOSProcesses(t *testing.T){
 if os.Getenv("NEXUS_FENCE_SUBPROCESS_HELPER")=="1"{return}
 root:=t.TempDir()
 helper:=func()error{
  cmd:=exec.Command(os.Args[0],"-test.run=^TestWriterFenceSubprocessHelper$")
  cmd.Env=append(os.Environ(),"NEXUS_FENCE_SUBPROCESS_HELPER=1","NEXUS_FENCE_SUBPROCESS_DIR="+root)
  output,err:=cmd.CombinedOutput()
  if err!=nil {return fmt.Errorf("child exit: %w (%s)",err,output)}
  return nil
 }
 held:=WithLegacyWriter(root,func()error{
  if err:=helper();err==nil{t.Fatal("independent process bypassed lock")}
  return nil
 })
 if held!=nil{t.Fatal(held)}
 if err:=helper();err!=nil{t.Fatalf("independent process cannot write after unlock: %v",err)}
}

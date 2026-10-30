package config

import (
 "errors"
 "os"
 "path/filepath"
 "strings"
 "testing"

 "github.com/IAPro-Community/nexus/internal/core/memoryrecovery"
)

func TestSaveConfigRefusesActiveRecoveryFenceWithoutChangingSettings(t *testing.T){
 configDir:=t.TempDir()
 dataDir:=t.TempDir()
 t.Setenv("NEXUS_CONFIG_DIR",configDir)
 t.Setenv("NEXUS_DATA_DIR",dataDir)

 original:=NewDefaultConfig()
 original.Language="pt-BR"
 if err:=SaveConfig(original);err!=nil{t.Fatal(err)}
 root:=filepath.Join(dataDir,"memory-recovery-journal")
 if err:=os.MkdirAll(root,0700);err!=nil{t.Fatal(err)}
 lock:=filepath.Join(root,".active.lock")
 if err:=os.WriteFile(lock,[]byte(strings.Repeat("a",32)),0600);err!=nil{t.Fatal(err)}
 modified:=original
 modified.Language="en"
 err:=SaveConfig(modified)
 if !errors.Is(err,memoryrecovery.ErrLegacyWriteFenced) {
  t.Fatalf("config write bypassed recovery fence: %v",err)
 }
 loaded,err:=LoadConfig()
 if err!=nil || loaded.Language!="pt-BR" {
  t.Fatalf("fenced write modified config: %q %v",loaded.Language,err)
 }
 if err:=os.Remove(lock);err!=nil{t.Fatal(err)}
 if err:=SaveConfig(modified);err!=nil{t.Fatal(err)}
 loaded,err=LoadConfig()
 if err!=nil || loaded.Language!="en" {
  t.Fatalf("released fence did not permit config write: %q %v",loaded.Language,err)
 }
}

func TestSaveConfigBlocksOnCorruptRecoveryJournalWithoutLock(t *testing.T){
 configDir:=t.TempDir()
 dataDir:=t.TempDir()
 t.Setenv("NEXUS_CONFIG_DIR",configDir)
 t.Setenv("NEXUS_DATA_DIR",dataDir)
 cfg:=NewDefaultConfig()
 if err:=SaveConfig(cfg);err!=nil{t.Fatal(err)}
 root:=filepath.Join(dataDir,"memory-recovery-journal")
 if err:=os.WriteFile(filepath.Join(root,"corrupt"),[]byte("unsafe"),0600);err!=nil{t.Fatal(err)}
 cfg.Language="es"
 err:=SaveConfig(cfg)
 if !errors.Is(err,memoryrecovery.ErrLegacyWriteFenced){
  t.Fatalf("corrupt journal did not block config write: %v",err)
 }
}

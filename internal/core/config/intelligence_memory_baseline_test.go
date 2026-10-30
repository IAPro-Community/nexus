package config

import (
 "strings"
 "testing"
)

func TestSaveConfigRejectsStaleMemoryAndKeepsRecoveredValue(t *testing.T) {
 t.Setenv("NEXUS_CONFIG_DIR",t.TempDir())
 t.Setenv("NEXUS_DATA_DIR",t.TempDir())

 base:=NewDefaultConfig()
 base.IntelligenceMemories=[]IntelligenceMemory{{
  ID:"imem_1",Key:"quality.testing",Value:"original",Source:"user",
 }}
 if err:=SaveConfig(base);err!=nil{t.Fatal(err)}

 stale,err:=LoadConfig()
 if err!=nil{t.Fatal(err)}
 fresh,err:=LoadConfig()
 if err!=nil{t.Fatal(err)}
 fresh.IntelligenceMemories[0].Value="restored"
 if err:=SaveConfig(fresh);err!=nil{t.Fatal(err)}
 stale.Language="es"
 err=SaveConfig(stale)
 if err==nil || !strings.Contains(err.Error(),"CAS precondition failed"){
  t.Fatalf("stale pre-restore config was accepted: %v",err)
 }
 current,err:=LoadConfig()
 if err!=nil{t.Fatal(err)}
 if current.IntelligenceMemories[0].Value!="restored"{
  t.Fatalf("stale caller overwrote recovered memory: %+v",current.IntelligenceMemories)
 }
}

func TestSaveConfigRejectsUnbasedOverwriteOfExistingMemories(t *testing.T){
 t.Setenv("NEXUS_CONFIG_DIR",t.TempDir())
 t.Setenv("NEXUS_DATA_DIR",t.TempDir())
 cfg:=NewDefaultConfig()
 cfg.IntelligenceMemories=[]IntelligenceMemory{{ID:"imem_1",Key:"quality.testing",Value:"keep"}}
 if err:=SaveConfig(cfg);err!=nil{t.Fatal(err)}
 unbased:=NewDefaultConfig()
 unbased.IntelligenceMemories=[]IntelligenceMemory{{ID:"imem_1",Key:"quality.testing",Value:"clobber"}}
 if err:=SaveConfig(unbased);err==nil{t.Fatal("unbased replacement of memory accepted")}
 raw,err:=LoadConfig()
 if err!=nil||raw.IntelligenceMemories[0].Value!="keep"{t.Fatalf("unbased write clobbered memory: %v %+v",err,raw)}
}

func TestSaveConfigCanEditNonMemorySettingsWithUnchangedMemoryBaseline(t *testing.T){
 t.Setenv("NEXUS_CONFIG_DIR",t.TempDir())
 t.Setenv("NEXUS_DATA_DIR",t.TempDir())
 cfg:=NewDefaultConfig()
 cfg.IntelligenceMemories=[]IntelligenceMemory{{ID:"imem_1",Key:"quality.testing",Value:"persist"}}
 if err:=SaveConfig(cfg);err!=nil{t.Fatal(err)}
 loaded,err:=LoadConfig()
 if err!=nil{t.Fatal(err)}
 loaded.Language="pt-BR"
 if err:=SaveConfig(loaded);err!=nil{t.Fatal(err)}
 result,err:=LoadConfig()
 if err!=nil||result.Language!="pt-BR"||result.IntelligenceMemories[0].Value!="persist"{
  t.Fatalf("nonmemory settings blocked: %v %+v",err,result)
 }
}

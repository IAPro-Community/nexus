package config

import (
 "crypto/sha256"
 "encoding/hex"
 "encoding/json"
 "errors"
 "fmt"
 "os"
 "sort"
)

// installMemoryBaseline records what a caller of LoadConfig actually saw.
// It is not persisted. The compare-and-swap check prevents a stale config
// read from overwriting a restored or concurrently updated INSTALL memory.
func installMemoryBaseline(rows []IntelligenceMemory) string {
 sorted:=append([]IntelligenceMemory{},rows...)
 sort.Slice(sorted,func(i,j int)bool{
  if sorted[i].ID!=sorted[j].ID{return sorted[i].ID<sorted[j].ID}
  return sorted[i].Key<sorted[j].Key
 })
 encoded,_:=json.Marshal(sorted)
 sum:=sha256.Sum256(encoded)
 return hex.EncodeToString(sum[:])
}

func checkInstallMemoryBaseline(configPath string, cfg Config) error {
 var current []IntelligenceMemory
 bytes,err:=os.ReadFile(configPath)
 if err!=nil && !os.IsNotExist(err){return err}
 if err==nil {
  var doc map[string]json.RawMessage
  if e:=json.Unmarshal(bytes,&doc);e!=nil||doc==nil{
   return errors.New("existing configuration is not a valid object")
  }
  if field,ok:=doc["intelligence_memories"];ok && string(field)!="null" {
   if e:=json.Unmarshal(field,&current);e!=nil{
    return errors.New("existing INSTALL memory is malformed")
   }
  }
 }
 live:=installMemoryBaseline(current)
 if cfg.installMemoryBaseline != "" {
  if cfg.installMemoryBaseline!=live {
   return fmt.Errorf("INSTALL memories changed since config was read (CAS precondition failed)")
  }
  return nil
 }
 // Configs created without LoadConfig carry no snapshot. Do not let them
 // silently delete or replace existing memory. An explicit unchanged set
 // may still be saved for backward compatibility with configuration APIs.
 if len(current)>0 && installMemoryBaseline(cfg.IntelligenceMemories)!=live {
  return errors.New("unbased configuration cannot overwrite existing INSTALL memories")
 }
 return nil
}

package app

import (
 "encoding/json"
 "errors"
 "fmt"

 "github.com/IAPro-Community/nexus/internal/core/config"
 "github.com/IAPro-Community/nexus/internal/core/memoryrecovery"
)

// Only status inspection is exposed. Starting, advancing, unlocking and
// applying a restore are deliberately NOT CLI commands in this phase.
func maestroMemoryRecoveryStatusCmd(args []string) error {
 asJSON := false
 for _, arg := range args {
  switch arg {
  case "--json":
   if asJSON { return errors.New("duplicate --json") }
   asJSON = true
  default:
   return fmt.Errorf("unknown memory-recovery-status option %q",arg)
  }
 }
 dataDir,err:=config.DataDir()
 if err!=nil{return err}
 report,err:=memoryrecovery.Inspect(dataDir)
 if err!=nil{return err}
 if asJSON {
  bytes,err:=json.MarshalIndent(report,"","  ")
  if err!=nil{return err}
  fmt.Println(string(bytes))
  if report.Blocked { return errors.New("recovery journal requires manual review") }
  return nil
 }
 fmt.Printf("Nexus recovery journal: %d operation(s)\n",len(report.Operations))
 if report.Blocked {
  fmt.Println("Recovery blocked: unfinished, locked or corrupted journal requires manual review")
 } else {
  fmt.Println("Recovery journal: no unresolved operations")
 }
 for _,op:=range report.Operations{
  fmt.Printf("  %s: %s (%d events)\n",op.ID,op.Stage,op.Events)
 }
 for _,problem:=range report.Problems{
  fmt.Printf("  integrity problem: %s\n",problem)
 }
 if report.Blocked { return errors.New("recovery journal requires manual review") }
 return nil
}

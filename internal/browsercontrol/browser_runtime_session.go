package browsercontrol

import (
 "context"
 "errors"
)

// Each runtime has its own browser; browsers aren't created when a CLI is
// launched but when it first calls browser_open. The separate runtimeMu
// serializes provisioning without holding Manager.mu across Chrome startup.
func (m *Manager) FindRuntimeBrowser(runtimeID string)(*Session,bool){
 m.mu.RLock()
 defer m.mu.RUnlock()
 for _,s:=range m.sessions{
  s.mu.Lock()
  belongs:=!s.closed&&s.info.OwnerRuntimeID==runtimeID
  s.mu.Unlock()
  if belongs{return s,true}
 }
 return nil,false
}
func (m *Manager) OpenRuntimeBrowser(ctx context.Context,cap RuntimeCapability,url string)(Info,error){
 if cap.RuntimeID==""||cap.ProjectID==""{return Info{},errors.New("browser runtime capability invalid")}
 m.runtimeMu.Lock()
 defer m.runtimeMu.Unlock()
 if s,ok:=m.FindRuntimeBrowser(cap.RuntimeID);ok{return s.Info(),nil}
 if url==""{url="http://localhost:3000"}
 info,err:=m.Create(ctx,cap.ProjectID,url)
 if err!=nil{return Info{},err}
 s,ok:=m.Get(info.ID)
 if !ok{return Info{},errors.New("browser disappeared during provisioning")}
 s.mu.Lock()
 s.info.OwnerRuntimeID=cap.RuntimeID
 info=s.info
 s.mu.Unlock()
 return info,nil
}

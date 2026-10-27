package web

import (
 "context"
 "encoding/json"
 "net/http"
 "strings"
 "time"

 "github.com/IAPro-Community/nexus/internal/browsercontrol"
)

// Browser routes reuse the Nexus authentication, CSRF and tunnel security boundary.
// DevTools sockets and Chrome debugging credentials are never exposed to clients.
func (s *Server) handleBrowserSessions(w http.ResponseWriter, r *http.Request) {
 if s.browserSessions == nil { writeError(w,http.StatusServiceUnavailable,"browser service unavailable"); return }
 w.Header().Set("Cache-Control","no-store")
 switch r.Method {
 case http.MethodGet:
  writeJSON(w,http.StatusOK,s.browserSessions.List(r.URL.Query().Get("project_id")))
 case http.MethodPost:
  var req struct{ProjectID string `json:"project_id"`; URL string `json:"url"`}
  r.Body=http.MaxBytesReader(w,r.Body,8192)
  d:=json.NewDecoder(r.Body); d.DisallowUnknownFields()
  if err:=d.Decode(&req); err!=nil { writeError(w,http.StatusBadRequest,"invalid browser session request"); return }
  if strings.TrimSpace(req.ProjectID)=="" { writeError(w,http.StatusBadRequest,"project_id required"); return }
  if req.URL!="" { if err:=browsercontrol.ValidURL(req.URL); err!=nil { writeError(w,http.StatusBadRequest,err.Error()); return } }
  ctx,cancel:=contextWithTimeout(r,20*time.Second); defer cancel()
  info,err:=s.browserSessions.Create(ctx,req.ProjectID,req.URL)
  if err!=nil { writeError(w,http.StatusServiceUnavailable,err.Error()); return }
  writeJSON(w,http.StatusCreated,info)
 default:
  w.Header().Set("Allow","GET, POST")
  writeError(w,http.StatusMethodNotAllowed,"method not allowed")
 }
}

// contextWithTimeout provides a bounded lifecycle for Chromium calls.
func contextWithTimeout(r *http.Request,d time.Duration)(context.Context,context.CancelFunc){
 return context.WithTimeout(r.Context(),d)
}

func (s *Server) handleBrowserSession(w http.ResponseWriter, r *http.Request) {
 if s.browserSessions==nil { writeError(w,http.StatusServiceUnavailable,"browser service unavailable"); return }
 w.Header().Set("Cache-Control","no-store")
 path:=strings.Trim(strings.TrimPrefix(r.URL.Path,"/api/v1/browser/sessions/"),"/")
 parts:=strings.Split(path,"/")
 if len(parts)<1 || len(parts)>2 || len(parts[0])!=32 { writeError(w,http.StatusNotFound,"browser session not found"); return }
 session,ok:=s.browserSessions.Get(parts[0])
 if !ok { writeError(w,http.StatusNotFound,"browser session not found"); return }
 action:=""
 if len(parts)==2 { action=parts[1] }
 ctx,cancel:=contextWithTimeout(r,12*time.Second)
 defer cancel()
 switch {
 case action=="" && r.Method==http.MethodGet:
  writeJSON(w,http.StatusOK,session.Info())
 case action=="" && r.Method==http.MethodDelete:
  s.browserSessions.Close(parts[0]);writeJSON(w,http.StatusOK,map[string]string{"status":"closed"})
 case action=="frame" && r.Method==http.MethodGet:
  img,err:=session.Screenshot(ctx)
  if err!=nil { writeError(w,http.StatusBadGateway,"browser screenshot unavailable");return }
  w.Header().Set("Content-Type","image/jpeg")
  w.Header().Set("X-Content-Type-Options","nosniff")
  _,_=w.Write(img)
 case action=="snapshot" && r.Method==http.MethodGet:
  snapshot,err:=session.Snapshot(ctx)
  if err!=nil { writeError(w,http.StatusBadGateway,"browser snapshot unavailable");return }
  writeJSON(w,http.StatusOK,snapshot)
 case action=="action" && r.Method==http.MethodPost:
  r.Body=http.MaxBytesReader(w,r.Body,16384)
  d:=json.NewDecoder(r.Body);d.DisallowUnknownFields()
  var input browsercontrol.Action
  if err:=d.Decode(&input);err!=nil {writeError(w,http.StatusBadRequest,"invalid browser action");return}
  info,err:=session.Do(ctx,input)
  if err!=nil {writeError(w,http.StatusBadRequest,err.Error());return}
  writeJSON(w,http.StatusOK,info)
 default:
  writeError(w,http.StatusMethodNotAllowed,"method not allowed")
 }
}

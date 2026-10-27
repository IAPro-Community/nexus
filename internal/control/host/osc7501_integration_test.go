package host

import (
 "encoding/base64"
 "testing"

 "github.com/IAPro-Community/nexus/internal/control/registry"
)

func TestNativeAGYQuestionPreemptsPTYHeuristic(t *testing.T) {
 detector:=NewAttentionDetector("rt-agy-native","agy","default","/workspace/demo",nil)
 msg:=base64.StdEncoding.EncodeToString([]byte("May I run migration?"))
 status:="\x1b]7501;state=blocked:kind=permission:app=agy:msg="+msg+"\x1b\\"
 detector.ProcessChunk([]byte(status[:12]))
 detector.ProcessChunk([]byte(status[12:]))
 if detector.lastReason!="APPROVAL"||detector.lastState!=registry.StateWaiting||
 detector.lastPrompt!="native_status"||detector.lastKind!=AttentionNeedsUser {
  t.Fatalf("native status lost: reason=%s state=%s prompt=%s kind=%s",
   detector.lastReason,detector.lastState,detector.lastPrompt,detector.lastKind)
 }
 detector.ProcessChunk([]byte("\r\nTask completed: quoted in docs\r\n"))
 if detector.lastKind!=AttentionNeedsUser {t.Fatal("stale PTY narration replaced native blocker")}
 detector.ProcessChunk([]byte("\x1b]7501;state=working:app=agy\x07"))
 if detector.lastKind!=AttentionWorking||detector.lastReason!="WORKING"{
  t.Fatal("provider working status did not dismiss old attention")
 }
 detector.ProcessChunk([]byte("\x1b]7501;state=done:app=agy\x07"))
 if detector.lastKind!=AttentionCompleted {t.Fatal("native completion not detected")}
}

func TestNativeSignalHasNoRemoteApprovalContract(t *testing.T){
 detector:=NewAttentionDetector("rt-native-confirm","agy","default","/workspace/demo",nil)
 detector.ProcessChunk([]byte("\x1b]7501;state=blocked:kind=auth:app=agy\x07"))
 if detector.lastReason!="AUTH"||detector.lastPrompt!="native_status"||
 detector.lastFingerprint=="" {t.Fatal("auth needs terminal attention without remote reply")}
}

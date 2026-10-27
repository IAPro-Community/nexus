package host

import (
	"encoding/base64"
	"testing"
)

func TestOSC7501ProgramStatus(t *testing.T) {
	msg := base64.StdEncoding.EncodeToString([]byte("Approve migration?"))
	seq := "\x1b]7501;state=blocked:kind=permission:app=agy:msg=" + msg + "\x1b\\"
	v, ok := extractOSC7501("before" + seq)
	if !ok || v.State != "blocked" || v.Kind != "permission" || v.Message != "Approve migration?" {
		t.Fatalf("bad status %+v %v", v, ok)
	}
	_, ok = extractOSC7501("\x1b]7501;?\x1b\\")
	if ok {
		t.Fatal("query must not look like a status")
	}
	_, ok = extractOSC7501("\x1b]7501;state=blocked:kind=garbage\x07")
	if ok {
		t.Fatal("unknown blocker")
	}
}

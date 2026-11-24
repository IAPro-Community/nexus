package web

import (
	"testing"
	"time"
)

func TestRemoteDevicesListAndRevoke(t *testing.T) {
	auth, _, err := NewAuthManager("127.0.0.1", "0")
	if err != nil {
		t.Fatalf("NewAuthManager: %v", err)
	}
	// A public tunnel makes the manager remote-exposed; only then are paired
	// sessions tracked as remote devices.
	auth.SetTunnelActive(true)

	remote, err := auth.CreateSession()
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	deviceID, ok := auth.AnnotateDevice(remote.ID, "Chrome · Android", "mobile")
	if !ok || deviceID == "" {
		t.Fatal("AnnotateDevice must mint an opaque device handle")
	}
	if deviceID == remote.ID {
		t.Fatal("device handle must never be the bearer session ID")
	}

	local, err := auth.CreateSession()
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	devices := auth.RemoteDevices()
	if len(devices) != 1 {
		t.Fatalf("expected only the paired remote device, got %d", len(devices))
	}
	if devices[0].DeviceID != deviceID || devices[0].Label != "Chrome · Android" || devices[0].Kind != "mobile" {
		t.Fatalf("unexpected device metadata: %+v", devices[0])
	}

	if auth.RevokeDevice(local.ID) {
		t.Fatal("a local session must not be revocable through the device handle path")
	}
	if !auth.SessionActive(local.ID) {
		t.Fatal("rejected revoke must not touch the local session")
	}

	if !auth.RevokeDevice(deviceID) {
		t.Fatal("RevokeDevice must revoke the paired device")
	}
	if auth.SessionActive(remote.ID) {
		t.Fatal("revoked device session must be inactive")
	}
	if len(auth.RemoteDevices()) != 0 {
		t.Fatal("revoked device must disappear from the list")
	}
	if auth.RevokeDevice(deviceID) {
		t.Fatal("revoking an unknown handle must be refused")
	}
}

func TestRemoteDeviceRevocationClosesOnlyThatDeviceSession(t *testing.T) {
	auth, _, err := NewAuthManager("127.0.0.1", "0")
	if err != nil {
		t.Fatalf("NewAuthManager: %v", err)
	}
	auth.SetTunnelActive(true)

	phoneA, err := auth.CreateSession()
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	deviceA, ok := auth.AnnotateDevice(phoneA.ID, "Chrome · Android", "mobile")
	if !ok {
		t.Fatal("AnnotateDevice must mint a device handle")
	}
	phoneB, err := auth.CreateSession()
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	deviceB, ok := auth.AnnotateDevice(phoneB.ID, "Safari · iOS", "mobile")
	if !ok {
		t.Fatal("AnnotateDevice must mint a device handle")
	}

	revokedA, stopA, activeA := auth.WatchSession(phoneA.ID)
	defer stopA()
	revokedB, stopB, activeB := auth.WatchSession(phoneB.ID)
	defer stopB()
	if !activeA || !activeB {
		t.Fatal("paired devices must start with live session watchers")
	}

	if !auth.RevokeDevice(deviceA) {
		t.Fatal("RevokeDevice must revoke the paired device")
	}
	select {
	case <-revokedA:
	case <-time.After(2 * time.Second):
		t.Fatal("revoking a device must immediately close its session watcher")
	}
	select {
	case <-revokedB:
		t.Fatal("revoking one device must not close another device session")
	default:
	}
	if !auth.SessionActive(phoneB.ID) {
		t.Fatal("the second paired device must stay active")
	}
	if !auth.RevokeDevice(deviceB) {
		t.Fatal("the second device must remain independently revocable")
	}
}

func TestDeviceLabelFromUserAgent(t *testing.T) {
	cases := []struct {
		ua    string
		label string
		kind  string
	}{
		{"Mozilla/5.0 (Linux; Android 14) AppleWebKit/537.36 Chrome/126.0 Mobile Safari/537.36", "Chrome · Android", "mobile"},
		{"Mozilla/5.0 (iPhone; CPU iPhone OS 17_0) AppleWebKit/605.1.15 Version/17.0 Mobile Safari/604.1", "Safari · iOS", "mobile"},
		{"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/126.0 Safari/537.36 Edg/126.0", "Edge · Windows", "desktop"},
	}
	for _, tc := range cases {
		label, kind := DeviceLabelFromUserAgent(tc.ua)
		if label != tc.label || kind != tc.kind {
			t.Fatalf("ua %q: got (%q,%q), want (%q,%q)", tc.ua, label, kind, tc.label, tc.kind)
		}
	}
}

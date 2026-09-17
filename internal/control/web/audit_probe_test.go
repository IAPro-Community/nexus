package web

import (
	"bytes"
	"encoding/base64"
	"strings"
	"testing"
)

// Auditor's extra probes (throwaway): polyglot tails must not survive
// normalization, non-allowlisted formats must be rejected.

func TestAuditProbeTrailingScriptAfterIEND(t *testing.T) {
	pngBytes := encodeTestPNG(t, 4, 4)
	evil := append(append([]byte{}, pngBytes...), []byte("<script>alert(document.domain)</script>")...)
	out, mime, err := normalizeAttachmentImage(evil)
	if err != nil {
		return // strict rejection of markup-tailed PNG is safe
	}
	if mime != "image/png" {
		t.Fatalf("mime=%q", mime)
	}
	if bytes.Contains(out, []byte("<script")) {
		t.Fatal("P0: trailing script survived normalization")
	}
}

func TestAuditProbeCorruptWebP(t *testing.T) {
	raw, _ := base64.StdEncoding.DecodeString("UklGRiIAAABXRUJQVlA4IBYAAAAwAQCdASoBAAEADsD+JaQAA3AAAAAA")
	corrupt := append(raw[:20], 0xFF, 0xFF, 0xFF, 0xFF)
	if _, _, err := normalizeAttachmentImage(corrupt); err == nil {
		t.Fatal("P0: corrupt webp accepted")
	}
	if _, _, err := normalizeAttachmentImage(raw[:10]); err == nil {
		t.Fatal("P0: truncated webp accepted")
	}
}

func TestAuditProbeGifAndBmpRejected(t *testing.T) {
	gif := []byte("GIF89a\x01\x00\x01\x00\x80\x00\x00\xff\xff\xff\x00\x00\x00!\xf9\x04\x01\x00\x00\x00\x00,\x00\x00\x00\x00\x01\x00\x01\x00\x00\x02\x02D\x01\x00;")
	if _, _, err := normalizeAttachmentImage(gif); err == nil {
		t.Fatal("P0: gif accepted despite allowlist")
	}
	bmp := append([]byte("BM"), bytes.Repeat([]byte{0}, 100)...)
	if _, _, err := normalizeAttachmentImage(bmp); err == nil {
		t.Fatal("P0: bmp accepted despite allowlist")
	}
}

func TestAuditProbeJpegTailScriptRejected(t *testing.T) {
	jpegBytes := encodeTestJPEG(t)
	withTail := append(append([]byte{}, jpegBytes...), []byte("XMP-evil<script>alert(1)</script>")...)
	out, _, err := normalizeAttachmentImage(withTail)
	if err != nil {
		return // strict rejection of markup-tailed JPEG is safe
	}
	if strings.Contains(string(out), "XMP-evil") {
		t.Fatal("P0: jpeg trailing metadata survived re-encode")
	}
}

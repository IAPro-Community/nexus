package web

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"hash/crc32"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IAPro-Community/nexus/internal/control/ids"
	"github.com/IAPro-Community/nexus/internal/nexus"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

// ── pure validation units ──────────────────────────────────────────────

func TestValidAttachmentScope(t *testing.T) {
	for _, ok := range []string{"cmp_01JABCDEF", "abc-123_XYZ", "s"} {
		if !validAttachmentScope(ok) {
			t.Errorf("scope %q must be valid", ok)
		}
	}
	for _, bad := range []string{"", "../x", "a/b", `a\b`, "a b", "a/b", strings.Repeat("a", 65), ".", "x/../y"} {
		if validAttachmentScope(bad) {
			t.Errorf("scope %q must be rejected", bad)
		}
	}
}

func TestValidAttachmentID(t *testing.T) {
	good := "att_" + ids.NewULID()
	if !validAttachmentID(good) {
		t.Fatalf("id %q must be valid", good)
	}
	for _, bad := range []string{
		"", "att_short", "att_" + strings.ToLower(good[4:]),
		"att_" + strings.Repeat("A", 25), "att_" + strings.Repeat("A", 27),
		"../att_" + strings.Repeat("A", 26), "att_01K7QZXWPR9N3VJ8KDZ6W4M0Y2*",
		"att_01K7QZXWPR9N3VJ8KDZ6W4M0IL", // I and L are not Crockford
		"/api/v1/attachments/" + good,
	} {
		if validAttachmentID(bad) {
			t.Errorf("id %q must be rejected", bad)
		}
	}
}

func encodeTestPNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{uint8(x), uint8(y), 128, 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func encodeTestJPEG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 80}); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// well-known 1x1 transparent WebP, base64 to keep the test readable.
func decodeTestWebP(t *testing.T) []byte {
	t.Helper()
	raw, err := base64.StdEncoding.DecodeString("UklGRiIAAABXRUJQVlA4IBYAAAAwAQCdASoBAAEADsD+JaQAA3AAAAAA")
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestNormalizeAcceptsRealImages(t *testing.T) {
	pngBytes := encodeTestPNG(t, 4, 4)
	out, mime, err := normalizeAttachmentImage(pngBytes)
	if err != nil {
		t.Fatalf("png: %v", err)
	}
	if mime != "image/png" || len(out) == 0 {
		t.Fatalf("png: mime=%q len=%d", mime, len(out))
	}
	jpegBytes := encodeTestJPEG(t)
	out, mime, err = normalizeAttachmentImage(jpegBytes)
	if err != nil {
		t.Fatalf("jpeg: %v", err)
	}
	if mime != "image/jpeg" {
		t.Fatalf("jpeg: mime=%q", mime)
	}
	webpBytes := decodeTestWebP(t)
	if got := sniffImageContentType(webpBytes); got != "image/webp" {
		t.Fatalf("webp fixture sniffed as %q", got)
	}
	out, mime, err = normalizeAttachmentImage(webpBytes)
	if err != nil {
		t.Fatalf("webp: %v", err)
	}
	// WebP is normalized to server-encoded PNG.
	if mime != "image/png" {
		t.Fatalf("webp: mime=%q, want image/png", mime)
	}
}

func TestNormalizeRejectsPolyglots(t *testing.T) {
	html := append([]byte("<!DOCTYPE html><html><script>alert(1)</script>"), encodeTestPNG(t, 2, 2)...)
	svg := []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`)
	js := []byte(`alert(document.cookie);//` + strings.Repeat("\x00", 600))
	for name, body := range map[string][]byte{
		"html-as-png": html,
		"svg":         svg,
		"js":          js,
		"text":        []byte("just some text, not an image"),
		"empty":       {},
		"truncated":   encodeTestPNG(t, 8, 8)[:40],
	} {
		if _, _, err := normalizeAttachmentImage(body); err == nil {
			t.Errorf("%s must be rejected", name)
		}
	}
}

// giantDimensionsPNG builds a structurally headed PNG whose IHDR claims
// 9000x9000, with a valid CRC so the decoder reaches the dimension gate.
func giantDimensionsPNG() []byte {
	buf := bytes.NewBuffer([]byte{137, 80, 78, 71, 13, 10, 26, 10})
	ihdr := []byte{
		0, 0, 0, 13, 'I', 'H', 'D', 'R',
		0, 0, 0x23, 0x28, // width 9000
		0, 0, 0x23, 0x28, // height 9000
		8, 2, 0, 0, 0,
	}
	crc := crc32.ChecksumIEEE(ihdr[4:])
	buf.Write(ihdr)
	buf.Write([]byte{byte(crc >> 24), byte(crc >> 16), byte(crc >> 8), byte(crc)})
	return buf.Bytes()
}

func TestNormalizeRejectsGiantDimensions(t *testing.T) {
	if _, _, err := normalizeAttachmentImage(giantDimensionsPNG()); err == nil {
		t.Fatal("9000x9000 IHDR must be rejected before full decode")
	}
}

func TestAttachmentPathConfinement(t *testing.T) {
	dir := t.TempDir()
	inside := filepath.Join(dir, "attachments", "cmp_x", "att_ABC")
	if !attachmentPathInsideDataDir(dir, inside) {
		t.Fatal("inside path must be accepted")
	}
	for _, outside := range []string{
		filepath.Join(dir, "other", "x"),
		filepath.Join(dir, "attachments", "..", "nexus.db"),
		"/etc/passwd",
		"",
	} {
		if attachmentPathInsideDataDir(dir, outside) {
			t.Errorf("path %q must be rejected", outside)
		}
	}
}

// ── HTTP-level adversarial tests ───────────────────────────────────────

func attachmentTestSession(t *testing.T, client *http.Client, base, csrf, dataDir string) (projectID, sessionID string) {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"name": "Attachments", "path": t.TempDir()})
	req, _ := http.NewRequest(http.MethodPost, base+"/api/v1/projects", bytes.NewReader(body))
	req.Header.Set("X-CSRF-Token", csrf)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		raw, _ := io.ReadAll(resp.Body)
		t.Fatalf("create project status %d: %s", resp.StatusCode, raw)
	}
	var proj store.Project
	if err := json.NewDecoder(resp.Body).Decode(&proj); err != nil {
		t.Fatal(err)
	}
	// The handler serves nexus.Default(): a process singleton whose store is
	// opened once. Insert the session through that same handle instead of a
	// second SQLite connection so setup always lands in the serving DB.
	st, err := nexus.Default().OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	sess, err := st.CreateComposerSession(store.ComposerSession{ProjectID: proj.ID, Title: "attacks"})
	if err != nil {
		t.Fatal(err)
	}
	return proj.ID, sess.ID
}

func uploadAttachment(t *testing.T, client *http.Client, url, csrf, filename, partMIME string, content []byte) *http.Response {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	hdr := textproto.MIMEHeader{}
	hdr.Set("Content-Disposition", `form-data; name="file"; filename="`+filename+`"`)
	hdr.Set("Content-Type", partMIME)
	part, err := mw.CreatePart(hdr)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	req, _ := http.NewRequest(http.MethodPost, url, &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("X-CSRF-Token", csrf)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func TestAttachmentUploadRoundTrip(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("NEXUS_DATA_DIR", dataDir)
	client, srv, csrf := csrfClient(t)
	base := srv.URL()
	_, sessionID := attachmentTestSession(t, client, base, csrf, dataDir)
	uploadURL := base + "/api/v1/composer-sessions/" + sessionID + "/attachments"

	// Client claims octet-stream for a real PNG: the server must sniff and win.
	pngBytes := encodeTestPNG(t, 6, 5)
	resp := uploadAttachment(t, client, uploadURL, csrf, "../../evil.png", "application/octet-stream", pngBytes)
	raw, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("upload status %d: %s", resp.StatusCode, raw)
	}
	var created struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		MimeType string `json:"mime_type"`
		Size     int64  `json:"size"`
		URL      string `json:"url"`
	}
	if err := json.Unmarshal(raw, &created); err != nil {
		t.Fatal(err)
	}
	if !validAttachmentID(created.ID) {
		t.Fatalf("server id %q is not a safe att_ ULID", created.ID)
	}
	if created.MimeType != "image/png" {
		t.Fatalf("mime=%q, want server-sniffed image/png", created.MimeType)
	}
	if strings.Contains(created.Name, "/") || strings.Contains(created.Name, "..") {
		t.Fatalf("display name %q leaked path", created.Name)
	}
	if !strings.HasPrefix(created.URL, "/api/v1/attachments/") {
		t.Fatalf("url %q", created.URL)
	}
	if bytes.Contains(raw, []byte("storage_ref")) {
		t.Fatal("upload response leaks storage_ref server path")
	}

	// List must not leak server paths either.
	resp, err := client.Get(uploadURL)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list status %d", resp.StatusCode)
	}
	if bytes.Contains(raw, []byte("storage_ref")) {
		t.Fatal("list response leaks storage_ref server path")
	}

	// Authorized GET serves normalized bytes with safe headers.
	resp, err = client.Get(base + created.URL)
	if err != nil {
		t.Fatal(err)
	}
	served, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get status %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "image/png" {
		t.Fatalf("Content-Type=%q, want image/png", ct)
	}
	if v := resp.Header.Get("X-Content-Type-Options"); v != "nosniff" {
		t.Fatalf("nosniff=%q", v)
	}
	if cd := resp.Header.Get("Content-Disposition"); !strings.HasPrefix(cd, "inline;") || strings.Contains(cd, "..") {
		t.Fatalf("Content-Disposition=%q", cd)
	}
	if cc := resp.Header.Get("Cache-Control"); cc == "" {
		t.Fatal("missing Cache-Control")
	}
	if sniffImageContentType(served) != "image/png" {
		t.Fatal("served bytes are not a real PNG")
	}

	// Authorized DELETE removes bytes and record.
	req, _ := http.NewRequest(http.MethodDelete, base+created.URL, nil)
	req.Header.Set("X-CSRF-Token", csrf)
	resp, err = client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("delete status %d", resp.StatusCode)
	}
	resp, err = client.Get(base + created.URL)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("get after delete = %d, want 404", resp.StatusCode)
	}
}

func TestAttachmentAdversarialUploads(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("NEXUS_DATA_DIR", dataDir)
	client, srv, csrf := csrfClient(t)
	base := srv.URL()
	_, sessionID := attachmentTestSession(t, client, base, csrf, dataDir)
	uploadURL := base + "/api/v1/composer-sessions/" + sessionID + "/attachments"

	cases := []struct {
		name     string
		filename string
		mime     string
		body     []byte
	}{
		{"html disguised as png", "x.png", "image/png", append([]byte("<html><script>alert(1)</script>"), encodeTestPNG(t, 2, 2)...)},
		{"svg", "x.svg", "image/svg+xml", []byte(`<svg xmlns="http://www.w3.org/2000/svg"/>`)},
		{"js with png mime", "x.png", "image/png", []byte(`alert(1);`)},
		{"fake mime text", "x.png", "image/png", []byte("not an image at all")},
		{"truncated png", "x.png", "image/png", encodeTestPNG(t, 8, 8)[:40]},
		{"giant dimensions", "x.png", "image/png", giantDimensionsPNG()},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := uploadAttachment(t, client, uploadURL, csrf, tc.filename, tc.mime, tc.body)
			resp.Body.Close()
			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("status=%d, want 400", resp.StatusCode)
			}
		})
	}

	t.Run("over per-file limit", func(t *testing.T) {
		big := bytes.Repeat([]byte{137, 80, 78, 71}, attachmentMaxFileBytes/4+1)
		resp := uploadAttachment(t, client, uploadURL, csrf, "big.png", "image/png", big)
		resp.Body.Close()
		if resp.StatusCode != http.StatusRequestEntityTooLarge {
			t.Fatalf("status=%d, want 413", resp.StatusCode)
		}
	})

	t.Run("oversized request body", func(t *testing.T) {
		big := bytes.Repeat([]byte("A"), attachmentMaxRequestBytes+1024)
		resp := uploadAttachment(t, client, uploadURL, csrf, "big.png", "image/png", big)
		resp.Body.Close()
		if resp.StatusCode != http.StatusRequestEntityTooLarge {
			t.Fatalf("status=%d, want 413", resp.StatusCode)
		}
	})

	t.Run("missing file field", func(t *testing.T) {
		var body bytes.Buffer
		mw := multipart.NewWriter(&body)
		_ = mw.WriteField("note", "no file here")
		_ = mw.Close()
		req, _ := http.NewRequest(http.MethodPost, uploadURL, &body)
		req.Header.Set("Content-Type", mw.FormDataContentType())
		req.Header.Set("X-CSRF-Token", csrf)
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("status=%d, want 400", resp.StatusCode)
		}
	})

	t.Run("unknown session", func(t *testing.T) {
		resp := uploadAttachment(t, client, base+"/api/v1/composer-sessions/cmp_DOESNOTEXIST/attachments", csrf, "x.png", "image/png", encodeTestPNG(t, 2, 2))
		resp.Body.Close()
		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("status=%d, want 404", resp.StatusCode)
		}
	})

	t.Run("traversal session", func(t *testing.T) {
		resp := uploadAttachment(t, client, base+"/api/v1/composer-sessions/../evil/attachments", csrf, "x.png", "image/png", encodeTestPNG(t, 2, 2))
		raw, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode == http.StatusCreated {
			t.Fatalf("traversal upload accepted: %s", raw)
		}
	})

	t.Run("missing csrf on mutation", func(t *testing.T) {
		resp := uploadAttachment(t, client, uploadURL, "", "x.png", "image/png", encodeTestPNG(t, 2, 2))
		resp.Body.Close()
		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("status=%d, want 403", resp.StatusCode)
		}
	})

	t.Run("unauthenticated upload", func(t *testing.T) {
		plain := &http.Client{}
		resp := uploadAttachment(t, plain, uploadURL, csrf, "x.png", "image/png", encodeTestPNG(t, 2, 2))
		resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("status=%d, want 401", resp.StatusCode)
		}
	})

	t.Run("unauthenticated get", func(t *testing.T) {
		plain := &http.Client{}
		resp, err := plain.Get(base + "/api/v1/attachments/att_" + strings.Repeat("A", 26))
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("status=%d, want 401", resp.StatusCode)
		}
	})

	t.Run("malformed id", func(t *testing.T) {
		for _, id := range []string{"nope", "att_short"} {
			resp, err := client.Get(base + "/api/v1/attachments/" + id)
			if err != nil {
				t.Fatal(err)
			}
			_, _ = io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				t.Fatalf("id %q served 200", id)
			}
		}
	})

	t.Run("traversal never discloses files", func(t *testing.T) {
		// Do not follow redirects: net/http cleans dot-segments into a
		// 301 toward the SPA fallback. Either a 301 or a 404 is safe;
		// serving the target file (200 + passwd content) is not.
		noRedirect := &http.Client{
			Jar: client.Jar,
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
		resp, err := noRedirect.Get(base + "/api/v1/attachments/%2e%2e%2fetc%2fpasswd")
		if err != nil {
			t.Fatal(err)
		}
		raw, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			t.Fatalf("traversal id served 200: %q", raw)
		}
		if bytes.Contains(raw, []byte("root:x:")) {
			t.Fatal("traversal disclosed /etc/passwd content")
		}
	})

	t.Run("wellformed unknown id", func(t *testing.T) {
		resp, err := client.Get(base + "/api/v1/attachments/att_" + strings.Repeat("B", 26))
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("status=%d, want 404", resp.StatusCode)
		}
	})

	t.Run("unauthenticated delete needs auth", func(t *testing.T) {
		plain := &http.Client{}
		req, _ := http.NewRequest(http.MethodDelete, base+"/api/v1/attachments/att_"+strings.Repeat("C", 26), nil)
		resp, err := plain.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("status=%d, want 401", resp.StatusCode)
		}
	})
}

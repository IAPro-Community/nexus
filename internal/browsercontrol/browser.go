// Package browsercontrol runs isolated Chromium sessions for the Nexus workspace.
// Chromium never exposes its DevTools endpoint outside the loopback interface.
package browsercontrol

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const MaxSessions = 4

type Info struct {
	ID        string `json:"id"`
	ProjectID string `json:"project_id"`
	URL       string `json:"url"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
}

type Action struct {
	Type      string  `json:"type"`
	URL       string  `json:"url,omitempty"`
	Text      string  `json:"text,omitempty"`
	Selector  string  `json:"selector,omitempty"`
	TimeoutMS int     `json:"timeout_ms,omitempty"`
	Width     int     `json:"width,omitempty"`
	Height    int     `json:"height,omitempty"`
	Key       string  `json:"key,omitempty"`
	X         float64 `json:"x,omitempty"`
	Y         float64 `json:"y,omitempty"`
	DeltaY    float64 `json:"delta_y,omitempty"`
}

type Session struct {
	mu              sync.Mutex
	info            Info
	wsURL           string
	cmd             *exec.Cmd
	done            chan struct{}
	directory       string
	closed          bool
	cdpConn         *websocket.Conn
	cdpNextID       int
	controller      string
	controllerUntil time.Time
	handoff Handoff
	streamMu        sync.Mutex
	streamReaders   map[chan []byte]struct{}
	streamCancel    context.CancelFunc
}

type Manager struct {
	mu       sync.RWMutex
	sessions map[string]*Session
	pending  int
	closing  bool
	grants map[[32]byte]grantEntry
}

func NewManager() *Manager { return &Manager{sessions: make(map[string]*Session)} }

func ValidURL(raw string) error {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u == nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil {
		return errors.New("browser URL must be an HTTP(S) address without embedded credentials")
	}
	host := strings.ToLower(u.Hostname())
	if host == "metadata.google.internal" || host == "metadata" {
		return errors.New("cloud metadata destinations are blocked")
	}
	if ip := net.ParseIP(host); ip != nil && (ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() || ip.IsMulticast()) {
		return errors.New("link-local, multicast and metadata addresses are blocked")
	}
	return nil
}

func chromeExecutable() (string, error) {
	if p := os.Getenv("NEXUS_CHROMIUM_PATH"); p != "" {
		if _, err := os.Stat(p); err != nil {
			return "", fmt.Errorf("NEXUS_CHROMIUM_PATH: %w", err)
		}
		return p, nil
	}
	for _, name := range []string{"google-chrome", "chromium", "chromium-browser", "microsoft-edge", "msedge", "chrome", "chrome.exe", "msedge.exe"} {
		if p, err := exec.LookPath(name); err == nil {
			return p, nil
		}
	}
	candidates := []string{}
	if runtime.GOOS == "darwin" {
		candidates = append(candidates, "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome", "/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge")
	}
	if runtime.GOOS == "windows" {
		for _, root := range []string{os.Getenv("PROGRAMFILES"), os.Getenv("PROGRAMFILES(X86)"), os.Getenv("LOCALAPPDATA")} {
			if root != "" {
				candidates = append(candidates, filepath.Join(root, "Google", "Chrome", "Application", "chrome.exe"), filepath.Join(root, "Microsoft", "Edge", "Application", "msedge.exe"))
			}
		}
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", errors.New("Chromium/Chrome/Edge not found; install a Chromium browser or set NEXUS_CHROMIUM_PATH")
}

func randomID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

type target struct {
	Type                 string `json:"type"`
	WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
}

func findPage(port string) (string, error) {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get("http://127.0.0.1:" + port + "/json/list")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("DevTools returned %d", resp.StatusCode)
	}
	var pages []target
	if err := json.NewDecoder(resp.Body).Decode(&pages); err != nil {
		return "", err
	}
	for _, page := range pages {
		if page.Type == "page" && strings.HasPrefix(page.WebSocketDebuggerURL, "ws://127.0.0.1:") {
			return page.WebSocketDebuggerURL, nil
		}
		// Chromium sometimes publishes a localhost URL rather than an IPv4 address.
		if page.Type == "page" && strings.HasPrefix(page.WebSocketDebuggerURL, "ws://localhost:") {
			return page.WebSocketDebuggerURL, nil
		}
	}
	return "", errors.New("Chromium has no debuggable page")
}

func (m *Manager) Create(ctx context.Context, projectID, initialURL string) (Info, error) {
	if strings.TrimSpace(projectID) == "" {
		return Info{}, errors.New("project_id required")
	}
	if initialURL == "" {
		initialURL = "https://example.org"
	}
	if err := ValidURL(initialURL); err != nil {
		return Info{}, err
	}
	m.mu.Lock()
	if m.closing {
		m.mu.Unlock()
		return Info{}, errors.New("browser manager is shutting down")
	}
	if len(m.sessions)+m.pending >= MaxSessions {
		m.mu.Unlock()
		return Info{}, fmt.Errorf("maximum %d browser sessions", MaxSessions)
	}
	m.pending++
	m.mu.Unlock()
	defer func() {
		m.mu.Lock()
		m.pending--
		m.mu.Unlock()
	}()
	binary, err := chromeExecutable()
	if err != nil {
		return Info{}, err
	}
	id, err := randomID()
	if err != nil {
		return Info{}, err
	}
	directory, err := os.MkdirTemp("", "nexus-chromium-")
	if err != nil {
		return Info{}, err
	}
	if err := os.Chmod(directory, 0700); err != nil {
		_ = os.RemoveAll(directory)
		return Info{}, err
	}
	cmd := exec.Command(binary,
		"--headless=new", "--remote-debugging-port=0", "--remote-debugging-address=127.0.0.1",
		"--user-data-dir="+directory, "--no-first-run", "--no-default-browser-check",
		"--disable-extensions", "--disable-background-networking", "--window-size=1280,800",
		"about:blank")
	if err := cmd.Start(); err != nil {
		_ = os.RemoveAll(directory)
		return Info{}, err
	}
	done := make(chan struct{})
	go func() { _ = cmd.Wait(); close(done) }()
	cleanup := func() { _ = cmd.Process.Kill(); <-done; _ = os.RemoveAll(directory) }
	deadline := time.NewTimer(25 * time.Second)
	defer deadline.Stop()
	var port, ws string
	start := time.NewTicker(100 * time.Millisecond)
	defer start.Stop()
	for ws == "" {
		select {
		case <-ctx.Done():
			cleanup()
			return Info{}, ctx.Err()
		case <-done:
			_ = os.RemoveAll(directory)
			return Info{}, errors.New("Chromium exited before DevTools became ready")
		case <-deadline.C:
			cleanup()
			return Info{}, errors.New("Chromium DevTools startup timeout")
		case <-start.C:
			b, err := os.ReadFile(filepath.Join(directory, "DevToolsActivePort"))
			if err != nil {
				continue
			}
			line := strings.SplitN(string(b), "\n", 2)[0]
			n, parseErr := strconv.Atoi(strings.TrimSpace(line))
			if parseErr != nil || n < 1 || n > 65535 {
				continue
			}
			port = strconv.Itoa(n)
			ws, _ = findPage(port)
		}
	}
	s := &Session{info: Info{ID: id, ProjectID: projectID, URL: initialURL, Width: 1280, Height: 800}, wsURL: ws, cmd: cmd, done: done, directory: directory}
	if err := s.navigate(ctx, initialURL); err != nil && !isNetworkNavigationError(err) {
		cleanup()
		return Info{}, err
	}
	m.mu.Lock()
	if m.closing {
		m.mu.Unlock()
		cleanup()
		return Info{}, errors.New("browser manager is shutting down")
	}
	m.sessions[id] = s
	m.mu.Unlock()
	return s.info, nil
}

func (m *Manager) List(projectID string) []Info {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]Info, 0)
	for _, s := range m.sessions {
		s.mu.Lock()
		if projectID == "" || s.info.ProjectID == projectID {
			result = append(result, s.info)
		}
		s.mu.Unlock()
	}
	return result
}

func (m *Manager) Get(id string) (*Session, bool) {
	m.mu.RLock()
	s, ok := m.sessions[id]
	m.mu.RUnlock()
	return s, ok
}

func (m *Manager) Close(id string) bool {
	m.mu.Lock()
	s, exists := m.sessions[id]
	if exists {
		delete(m.sessions, id)
	}
	m.mu.Unlock()
	if !exists {
		return false
	}
	// Immediately revoke every credential scoped to this browser.
	m.mu.Lock()
	for digest, entry := range m.grants {
		if entry.SessionID == id { delete(m.grants, digest) }
	}
	m.mu.Unlock()
	s.stopStream()
	s.mu.Lock()
	s.closed = true
	if s.cdpConn != nil {
		_ = s.cdpConn.Close()
		s.cdpConn = nil
	}
	_ = s.cmd.Process.Kill()
	<-s.done
	_ = os.RemoveAll(s.directory)
	s.mu.Unlock()
	return true
}

func (m *Manager) Shutdown() {
	m.mu.Lock()
	m.closing = true
	ids := make([]string, 0, len(m.sessions))
	for id := range m.sessions {
		ids = append(ids, id)
	}
	m.mu.Unlock()
	for _, id := range ids {
		m.Close(id)
	}
}

// call is serialized by Session.mu. One CDP connection is reused for commands
// and snapshots. A failed connection is discarded and dialed again next time.
func (s *Session) call(ctx context.Context, method string, params any, into any) error {
	if s.closed {
		return errors.New("browser session closed")
	}
	timeout := 8 * time.Second
	if deadline, ok := ctx.Deadline(); ok {
		if remaining := time.Until(deadline); remaining < timeout {
			timeout = remaining
		}
	}
	if timeout <= 0 {
		return context.DeadlineExceeded
	}
	if s.cdpConn == nil {
		dialer := websocket.Dialer{HandshakeTimeout: timeout}
		conn, _, err := dialer.DialContext(ctx, s.wsURL, nil)
		if err != nil {
			return fmt.Errorf("DevTools connect: %w", err)
		}
		s.cdpConn = conn
	}
	c := s.cdpConn
	_ = c.SetReadDeadline(time.Now().Add(timeout))
	_ = c.SetWriteDeadline(time.Now().Add(timeout))
	s.cdpNextID++
	id := s.cdpNextID
	fail := func(err error) error {
		_ = c.Close()
		s.cdpConn = nil
		return err
	}
	if err := c.WriteJSON(map[string]any{"id": id, "method": method, "params": params}); err != nil {
		return fail(err)
	}
	for {
		var msg struct {
			ID     int             `json:"id"`
			Result json.RawMessage `json:"result"`
			Error  *struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := c.ReadJSON(&msg); err != nil {
			return fail(err)
		}
		if msg.ID != id {
			continue
		}
		if msg.Error != nil {
			return errors.New(msg.Error.Message)
		}
		if into != nil {
			return json.Unmarshal(msg.Result, into)
		}
		return nil
	}
}

func (s *Session) navigate(ctx context.Context, raw string) error {
	if err := ValidURL(raw); err != nil {
		return err
	}
	var result struct {
		ErrorText string `json:"errorText"`
	}
	if err := s.call(ctx, "Page.navigate", map[string]any{"url": raw}, &result); err != nil {
		return err
	}
	s.info.URL = raw
	if result.ErrorText != "" {
		return &NavigationError{URL: raw, Reason: result.ErrorText}
	}
	return nil
}

func (s *Session) Do(ctx context.Context, a Action) (Info, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.doLocked(ctx, a)
}

func (s *Session) doLocked(ctx context.Context, a Action) (Info, error) {
	if s.closed {
		return Info{}, errors.New("browser session closed")
	}
	var err error
	switch a.Type {
	case "navigate":
		err = s.navigate(ctx, a.URL)
	case "click_selector":
		err = s.clickSelector(ctx, a.Selector)
	case "fill_selector":
		err = s.fillSelector(ctx, a.Selector, a.Text)
	case "wait_selector":
		err = s.waitSelector(ctx, a.Selector, a.TimeoutMS)
	case "viewport":
		if a.Width < 320 || a.Width > 1920 || a.Height < 240 || a.Height > 1200 {
			return Info{}, errors.New("viewport dimensions out of range")
		}
		err = s.call(ctx, "Emulation.setDeviceMetricsOverride", map[string]any{
			"width": a.Width, "height": a.Height, "deviceScaleFactor": 1, "mobile": a.Width < 600,
		}, nil)
		if err == nil {
			s.info.Width = a.Width
			s.info.Height = a.Height
		}
	case "reload":
		err = s.call(ctx, "Page.reload", map[string]any{"ignoreCache": false}, nil)
	case "back", "forward":
		var history struct {
			CurrentIndex int `json:"currentIndex"`
			Entries      []struct {
				ID int `json:"id"`
			} `json:"entries"`
		}
		err = s.call(ctx, "Page.getNavigationHistory", map[string]any{}, &history)
		if err == nil {
			target := history.CurrentIndex - 1
			if a.Type == "forward" {
				target = history.CurrentIndex + 1
			}
			if target >= 0 && target < len(history.Entries) {
				err = s.call(ctx, "Page.navigateToHistoryEntry", map[string]any{"entryId": history.Entries[target].ID}, nil)
			}
		}
	case "click":
		if a.X < 0 || a.Y < 0 || a.X >= float64(s.info.Width) || a.Y >= float64(s.info.Height) {
			return Info{}, errors.New("click outside viewport")
		}
		p := map[string]any{"x": a.X, "y": a.Y, "button": "left", "clickCount": 1}
		p["type"] = "mousePressed"
		err = s.call(ctx, "Input.dispatchMouseEvent", p, nil)
		if err == nil {
			p["type"] = "mouseReleased"
			err = s.call(ctx, "Input.dispatchMouseEvent", p, nil)
		}
	case "scroll":
		err = s.call(ctx, "Input.dispatchMouseEvent", map[string]any{"type": "mouseWheel", "x": a.X, "y": a.Y, "deltaX": 0, "deltaY": a.DeltaY}, nil)
	case "type":
		if len(a.Text) > 8192 {
			return Info{}, errors.New("text too long")
		}
		err = s.call(ctx, "Input.insertText", map[string]any{"text": a.Text}, nil)
	case "key":
		codes := map[string]int{"Enter": 13, "Tab": 9, "Escape": 27, "Backspace": 8, "ArrowUp": 38, "ArrowDown": 40, "ArrowLeft": 37, "ArrowRight": 39, "Delete": 46}
		code, ok := codes[a.Key]
		if !ok {
			return Info{}, errors.New("unsupported key")
		}
		p := map[string]any{"type": "keyDown", "key": a.Key, "windowsVirtualKeyCode": code, "nativeVirtualKeyCode": code}
		err = s.call(ctx, "Input.dispatchKeyEvent", p, nil)
		if err == nil {
			p["type"] = "keyUp"
			err = s.call(ctx, "Input.dispatchKeyEvent", p, nil)
		}
	default:
		return Info{}, errors.New("unsupported browser action")
	}
	if err != nil {
		return Info{}, err
	}
	return s.info, nil
}

func (s *Session) Screenshot(ctx context.Context) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.screenshotLocked(ctx)
}

func (s *Session) screenshotLocked(ctx context.Context) ([]byte, error) {
	var out struct {
		Data string `json:"data"`
	}
	if err := s.call(ctx, "Page.captureScreenshot", map[string]any{"format": "jpeg", "quality": 65, "fromSurface": true}, &out); err != nil {
		return nil, err
	}
	return base64.StdEncoding.DecodeString(out.Data)
}

func (s *Session) Snapshot(ctx context.Context) (json.RawMessage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.snapshotLocked(ctx)
}

func (s *Session) snapshotLocked(ctx context.Context) (json.RawMessage, error) {
	// Do not expose values of password fields or arbitrary script evaluation.
	const expression = `(()=>{
		const path=(el)=>{
			const result=[];
			for(let node=el;node&&node.nodeType===1&&result.length<10;node=node.parentElement){
				if(node.id){result.unshift('#'+CSS.escape(node.id));break;}
				let part=node.tagName.toLowerCase();
				const testid=node.getAttribute('data-testid');
				if(testid){part+='[data-testid="'+CSS.escape(testid)+'"]';result.unshift(part);break;}
				if(node.parentElement){
					const peers=Array.from(node.parentElement.children).filter(e=>e.tagName===node.tagName);
					if(peers.length>1)part+=':nth-of-type('+(peers.indexOf(node)+1)+')';
				}
				result.unshift(part);
			}
			return result.join(' > ');
		};
		return {url:location.href,title:document.title,untrusted:true,
			text:(document.body?.innerText||'').slice(0,12000),
			elements:Array.from(document.querySelectorAll('a,button,input,textarea,select,[role="button"]')).slice(0,80).map(e=>({
				selector:path(e),tag:e.tagName,role:e.getAttribute('role'),
				label:e.getAttribute('aria-label')||e.getAttribute('placeholder')||'',
				text:(e.innerText||'').slice(0,100)
			}))};
	})()`
	var out struct {
		Result struct {
			Value json.RawMessage `json:"value"`
		} `json:"result"`
	}
	if err := s.call(ctx, "Runtime.evaluate", map[string]any{"expression": expression, "returnByValue": true, "awaitPromise": false}, &out); err != nil {
		return nil, err
	}
	if len(out.Result.Value) == 0 {
		return json.RawMessage("{}"), nil
	}
	var meta struct {
		URL string `json:"url"`
	}
	if json.Unmarshal(out.Result.Value, &meta) == nil && meta.URL != "" {
		s.info.URL = meta.URL
	}
	return out.Result.Value, nil
}

func (s *Session) Info() Info {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.info
}

type NavigationError struct {
	URL    string
	Reason string
}

func (e *NavigationError) Error() string { return "browser navigation failed: " + e.Reason }

// A session on an unreachable development URL must still be usable. The
// Chromium error page remains interactive and the address can be corrected.
func isNetworkNavigationError(err error) bool {
	var nav *NavigationError
	return errors.As(err, &nav)
}

// AcquireControl gives one authenticated Nexus session the input lease.
// An idle controller expires, enabling remote takeover without two devices
// typing into the same page. Reading frames/snapshots requires no lease.
func (s *Session) AcquireControl(owner string) (bool, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if owner == "" || s.closed {
		return false, ""
	}
	now := time.Now()
	if s.handoff.Status == "requested" || (s.handoff.Status == "human_active" && s.controller != owner) {
		return false, "browser paused for human intervention"
	}
	if s.controller != "" && s.controller != owner && now.Before(s.controllerUntil) {
		return false, "another device controls this browser"
	}
	s.controller = owner
	s.controllerUntil = now.Add(25 * time.Second)
	return true, ""
}
func (s *Session) ReleaseControl(owner string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if owner != "" && s.controller == owner {
		s.controller = ""
		s.controllerUntil = time.Time{}
	}
}
func (s *Session) CanControl(owner string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if owner == "" || s.closed || s.controller != owner {
		return false
	}
	if time.Now().After(s.controllerUntil) {
		s.controller = ""
		return false
	}
	return true
}

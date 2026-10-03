package browser

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/gorilla/websocket"
)

const (
	designModeProtocolVersion = 1
	maxDesignHTMLBytes        = 12 * 1024
	maxDesignTextBytes        = 4 * 1024
	maxDesignScreenshotEdge   = 2048.0
)

var devToolsListeningPattern = regexp.MustCompile(`DevTools listening on (ws://[^\s]+)`)

type UIElementBounds struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

type UIContextArtifact struct {
	Version        int               `json:"version"`
	URL            string            `json:"url"`
	Title          string            `json:"title,omitempty"`
	Selector       string            `json:"selector"`
	HTML           string            `json:"html"`
	Neighborhood   string            `json:"neighborhood,omitempty"`
	Text           string            `json:"text,omitempty"`
	ComputedStyles map[string]string `json:"computed_styles,omitempty"`
	Bounds         UIElementBounds   `json:"bounds"`
	SourceHints    map[string]string `json:"source_hints,omitempty"`
	ScreenshotPath string            `json:"screenshot_path,omitempty"`
	CapturedAt     time.Time         `json:"captured_at"`
}

type DesignBrowser struct {
	mu         sync.Mutex
	cmd        *exec.Cmd
	done       chan error
	profileDir string
	client     *cdpClient
	sessionID  string
	closed     bool
}

func StartDesignBrowser(ctx context.Context, rawURL string) (*DesignBrowser, error) {
	if err := validateDesignURL(rawURL); err != nil {
		return nil, err
	}
	binary := ChromiumExecutable()
	if binary == "" {
		return nil, fmt.Errorf("no Chromium-family browser with DevTools support was found")
	}
	profileDir, err := os.MkdirTemp("", "nexus-design-browser-*")
	if err != nil {
		return nil, fmt.Errorf("create Design Mode browser profile: %w", err)
	}
	_ = os.Chmod(profileDir, 0o700)

	cmd := exec.Command(binary,
		"--remote-debugging-address=127.0.0.1",
		"--remote-debugging-port=0",
		"--no-first-run",
		"--no-default-browser-check",
		"--disable-sync",
		"--disable-background-networking",
		"--disable-component-update",
		"--disable-default-apps",
		"--user-data-dir="+profileDir,
		rawURL,
	)
	stderr, err := cmd.StderrPipe()
	if err != nil {
		_ = os.RemoveAll(profileDir)
		return nil, err
	}
	cmd.Stdout = io.Discard
	if err := cmd.Start(); err != nil {
		_ = os.RemoveAll(profileDir)
		return nil, fmt.Errorf("start Design Mode Chromium: %w", err)
	}
	browser := &DesignBrowser{cmd: cmd, done: make(chan error, 1), profileDir: profileDir}
	go func() { browser.done <- cmd.Wait() }()

	endpointCh := make(chan string, 1)
	go scanDevToolsEndpoint(stderr, endpointCh)
	startupTimer := time.NewTimer(12 * time.Second)
	defer startupTimer.Stop()

	var endpoint string
	select {
	case endpoint = <-endpointCh:
		if endpoint == "" {
			_ = browser.Close()
			return nil, fmt.Errorf("Chromium exited before exposing DevTools")
		}
	case err := <-browser.done:
		_ = os.RemoveAll(profileDir)
		return nil, fmt.Errorf("Design Mode Chromium exited during startup: %w", err)
	case <-startupTimer.C:
		_ = browser.Close()
		return nil, fmt.Errorf("timed out waiting for Chromium DevTools endpoint")
	case <-ctx.Done():
		_ = browser.Close()
		return nil, ctx.Err()
	}

	client, err := dialCDP(ctx, endpoint)
	if err != nil {
		_ = browser.Close()
		return nil, err
	}
	browser.client = client
	if err := browser.attachPage(ctx, rawURL); err != nil {
		_ = browser.Close()
		return nil, err
	}
	return browser, nil
}

func scanDevToolsEndpoint(stderr io.Reader, out chan<- string) {
	scanner := bufio.NewScanner(stderr)
	for scanner.Scan() {
		match := devToolsListeningPattern.FindStringSubmatch(scanner.Text())
		if len(match) == 2 {
			out <- match[1]
			return
		}
	}
	out <- ""
}

func validateDesignURL(raw string) error {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return fmt.Errorf("invalid Design Mode URL: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("Design Mode only supports http/https URLs")
	}
	if parsed.Host == "" {
		return fmt.Errorf("Design Mode URL requires a host")
	}
	return nil
}

func (b *DesignBrowser) attachPage(ctx context.Context, wantedURL string) error {
	deadline := time.Now().Add(10 * time.Second)
	wanted, _ := url.Parse(wantedURL)
	for {
		var targets struct {
			TargetInfos []struct {
				TargetID string `json:"targetId"`
				Type     string `json:"type"`
				URL      string `json:"url"`
			} `json:"targetInfos"`
		}
		if err := b.client.call(ctx, "", "Target.getTargets", map[string]any{}, &targets); err != nil {
			return err
		}
		selected := ""
		fallback := ""
		for _, target := range targets.TargetInfos {
			if target.Type != "page" || strings.HasPrefix(target.URL, "chrome://") {
				continue
			}
			if fallback == "" {
				fallback = target.TargetID
			}
			parsed, _ := url.Parse(target.URL)
			if wanted != nil && parsed != nil && strings.EqualFold(parsed.Host, wanted.Host) {
				selected = target.TargetID
				break
			}
		}
		if selected == "" {
			selected = fallback
		}
		if selected != "" {
			var attached struct {
				SessionID string `json:"sessionId"`
			}
			if err := b.client.call(ctx, "", "Target.attachToTarget", map[string]any{
				"targetId": selected,
				"flatten":  true,
			}, &attached); err != nil {
				return err
			}
			if attached.SessionID == "" {
				return fmt.Errorf("Chromium did not return a Design Mode page session")
			}
			b.sessionID = attached.SessionID
			return b.waitDocumentReady(ctx)
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("no Chromium page target became available")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}
}

func (b *DesignBrowser) waitDocumentReady(ctx context.Context) error {
	deadline := time.Now().Add(10 * time.Second)
	for {
		var value string
		if err := b.evaluateString(ctx, "document.readyState", &value); err == nil && (value == "interactive" || value == "complete") {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("Design Mode page did not become ready")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}
}


type DesignPickerStatus struct {
	Active    bool   `json:"active"`
	Selector  string `json:"selector,omitempty"`
	Cancelled bool   `json:"cancelled,omitempty"`
}

func (b *DesignBrowser) BeginElementPicker(ctx context.Context) (*DesignPickerStatus, error) {
	expression := `(() => {
  const previous = window.__nexusDesignPicker;
  if (previous && typeof previous.cleanup === 'function') previous.cleanup(true);
  const state = { selector: '', cancelled: false, active: true };
  const overlay = document.createElement('div');
  overlay.setAttribute('data-nexus-design-picker', 'true');
  Object.assign(overlay.style, {
    position: 'fixed',
    pointerEvents: 'none',
    zIndex: '2147483647',
    border: '2px solid #38bdf8',
    background: 'rgba(56,189,248,0.12)',
    display: 'none',
    boxSizing: 'border-box'
  });
  (document.documentElement || document.body).appendChild(overlay);

  const selectorFor = (node) => {
    if (!(node instanceof Element)) return '';
    if (node.id) return '#' + CSS.escape(node.id);
    const parts = [];
    let current = node;
    while (current && current.nodeType === 1 && parts.length < 7) {
      let part = current.tagName.toLowerCase();
      const testid = current.getAttribute('data-testid');
      if (testid) {
        parts.unshift('[data-testid="' + CSS.escape(testid) + '"]');
        break;
      }
      const parent = current.parentElement;
      if (parent) {
        const peers = Array.from(parent.children).filter((child) => child.tagName === current.tagName);
        if (peers.length > 1) part += ':nth-of-type(' + (peers.indexOf(current) + 1) + ')';
      }
      parts.unshift(part);
      current = parent;
    }
    return parts.join(' > ');
  };

  const onMove = (event) => {
    const target = event.target;
    if (!(target instanceof Element) || target === overlay) return;
    const rect = target.getBoundingClientRect();
    Object.assign(overlay.style, {
      display: 'block',
      left: rect.left + 'px',
      top: rect.top + 'px',
      width: rect.width + 'px',
      height: rect.height + 'px'
    });
  };
  const cleanup = (removeState) => {
    document.removeEventListener('mousemove', onMove, true);
    document.removeEventListener('click', onClick, true);
    document.removeEventListener('keydown', onKey, true);
    overlay.remove();
    state.active = false;
    if (removeState) delete window.__nexusDesignPicker;
  };
  const onClick = (event) => {
    if (!(event.target instanceof Element) || event.target === overlay) return;
    event.preventDefault();
    event.stopPropagation();
    event.stopImmediatePropagation();
    state.selector = selectorFor(event.target);
    cleanup(false);
  };
  const onKey = (event) => {
    if (event.key !== 'Escape') return;
    event.preventDefault();
    state.cancelled = true;
    cleanup(false);
  };
  state.cleanup = cleanup;
  window.__nexusDesignPicker = state;
  document.addEventListener('mousemove', onMove, true);
  document.addEventListener('click', onClick, true);
  document.addEventListener('keydown', onKey, true);
  return JSON.stringify({active:true});
})()`
	var serialized string
	if err := b.evaluateString(ctx, expression, &serialized); err != nil {
		return nil, err
	}
	var status DesignPickerStatus
	if err := json.Unmarshal([]byte(serialized), &status); err != nil {
		return nil, fmt.Errorf("decode Design Mode picker start: %w", err)
	}
	return &status, nil
}

func (b *DesignBrowser) PickerStatus(ctx context.Context) (*DesignPickerStatus, error) {
	expression := `(() => {
  const state = window.__nexusDesignPicker;
  if (!state) return JSON.stringify({active:false});
  return JSON.stringify({
    active: Boolean(state.active),
    selector: String(state.selector || ''),
    cancelled: Boolean(state.cancelled)
  });
})()`
	var serialized string
	if err := b.evaluateString(ctx, expression, &serialized); err != nil {
		return nil, err
	}
	var status DesignPickerStatus
	if err := json.Unmarshal([]byte(serialized), &status); err != nil {
		return nil, fmt.Errorf("decode Design Mode picker status: %w", err)
	}
	return &status, nil
}

func (b *DesignBrowser) ClearPicker(ctx context.Context) error {
	var ignored string
	return b.evaluateString(ctx, `(() => {
  const state = window.__nexusDesignPicker;
  if (state && typeof state.cleanup === 'function') state.cleanup(true);
  delete window.__nexusDesignPicker;
  return 'ok';
})()`, &ignored)
}

type designElementSnapshot struct {
	Found          bool
	Restricted     bool              `json:"found"`
	URL            string            `json:"url"`
	Title          string            `json:"title"`
	HTML           string            `json:"html"`
	Neighborhood   string            `json:"neighborhood"`
	Text           string            `json:"text"`
	ComputedStyles map[string]string `json:"computedStyles"`
	Bounds         UIElementBounds   `json:"bounds"`
	SourceHints    map[string]string `json:"sourceHints"`
}

func (b *DesignBrowser) InspectSelector(ctx context.Context, selector string) (*UIContextArtifact, error) {
	selector = strings.TrimSpace(selector)
	if selector == "" {
		return nil, fmt.Errorf("Design Mode selector is required")
	}
	literal, _ := json.Marshal(selector)
	expression := fmt.Sprintf(`(() => {
  const selector = %s;
  const el = document.querySelector(selector);
  if (!el) return JSON.stringify({found:false});
  if (el.matches('input[type="password"], [data-nexus-sensitive="true"]')) {
    return JSON.stringify({found:false, restricted:true});
  }
  const cs = getComputedStyle(el);
  const rect = el.getBoundingClientRect();
  const keys = ['display','position','color','backgroundColor','fontFamily','fontSize','fontWeight','lineHeight','letterSpacing','margin','padding','gap','width','height','border','borderRadius','boxShadow','opacity','overflow','alignItems','justifyContent'];
  const styles = {};
  for (const key of keys) styles[key] = String(cs[key] ?? '');
  const sourceHints = {};
  for (const attr of ['data-source','data-source-file','data-source-line','data-component','data-testid']) {
    const value = el.getAttribute(attr);
    if (value) sourceHints[attr] = value.slice(0, 512);
  }
  if (el.id) sourceHints.id = el.id.slice(0, 512);
  if (typeof el.className === 'string' && el.className) sourceHints.class = el.className.slice(0, 1024);
  return JSON.stringify({
    found:true,
    url:location.href,
    title:document.title,
    html:el.outerHTML.slice(0,%d),
    neighborhood:(el.parentElement?.outerHTML || '').slice(0,%d),
    text:(el.textContent || '').trim().slice(0,%d),
    computedStyles:styles,
    bounds:{x:rect.left+scrollX,y:rect.top+scrollY,width:rect.width,height:rect.height},
    sourceHints
  });
})()`, string(literal), maxDesignHTMLBytes, maxDesignHTMLBytes, maxDesignTextBytes)

	var serialized string
	if err := b.evaluateString(ctx, expression, &serialized); err != nil {
		return nil, err
	}
	var snapshot designElementSnapshot
	if err := json.Unmarshal([]byte(serialized), &snapshot); err != nil {
		return nil, fmt.Errorf("decode Design Mode element snapshot: %w", err)
	}
	if snapshot.Restricted {
		return nil, fmt.Errorf("Design Mode will not capture sensitive elements")
	}
	if !snapshot.Found {
		return nil, fmt.Errorf("Design Mode selector %q matched no element", selector)
	}
	artifact := &UIContextArtifact{
		Version:        designModeProtocolVersion,
		URL:            snapshot.URL,
		Title:          snapshot.Title,
		Selector:       selector,
		HTML:           snapshot.HTML,
		Neighborhood:   snapshot.Neighborhood,
		Text:           snapshot.Text,
		ComputedStyles: snapshot.ComputedStyles,
		Bounds:         snapshot.Bounds,
		SourceHints:    snapshot.SourceHints,
		CapturedAt:     time.Now().UTC(),
	}
	if snapshot.Bounds.Width > 0 && snapshot.Bounds.Height > 0 {
		path, err := b.captureElementScreenshot(ctx, selector, snapshot.Bounds)
		if err == nil {
			artifact.ScreenshotPath = path
		}
	}
	return artifact, nil
}

func (b *DesignBrowser) evaluateString(ctx context.Context, expression string, out *string) error {
	var evaluated struct {
		Result struct {
			Type  string `json:"type"`
			Value any    `json:"value"`
		} `json:"result"`
		ExceptionDetails json.RawMessage `json:"exceptionDetails,omitempty"`
	}
	if err := b.client.call(ctx, b.sessionID, "Runtime.evaluate", map[string]any{
		"expression":    expression,
		"returnByValue": true,
		"awaitPromise":  true,
	}, &evaluated); err != nil {
		return err
	}
	if len(evaluated.ExceptionDetails) > 0 && string(evaluated.ExceptionDetails) != "null" {
		return fmt.Errorf("Design Mode page evaluation failed")
	}
	value, ok := evaluated.Result.Value.(string)
	if !ok {
		return fmt.Errorf("Design Mode evaluation returned %T, expected string", evaluated.Result.Value)
	}
	*out = value
	return nil
}

func (b *DesignBrowser) captureElementScreenshot(ctx context.Context, selector string, bounds UIElementBounds) (string, error) {
	width := bounds.Width
	height := bounds.Height
	if width > maxDesignScreenshotEdge {
		width = maxDesignScreenshotEdge
	}
	if height > maxDesignScreenshotEdge {
		height = maxDesignScreenshotEdge
	}
	var captured struct {
		Data string `json:"data"`
	}
	if err := b.client.call(ctx, b.sessionID, "Page.captureScreenshot", map[string]any{
		"format":      "png",
		"fromSurface": true,
		"captureBeyondViewport": true,
		"clip": map[string]any{
			"x": bounds.X, "y": bounds.Y, "width": width, "height": height, "scale": 1,
		},
	}, &captured); err != nil {
		return "", err
	}
	pngBytes, err := base64.StdEncoding.DecodeString(captured.Data)
	if err != nil {
		return "", fmt.Errorf("decode Design Mode screenshot: %w", err)
	}
	dataDir, err := config.DataDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(dataDir, "design-mode")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s\x00%s\x00%d", selector, b.sessionID, time.Now().UnixNano())))
	path := filepath.Join(dir, "element-"+hex.EncodeToString(sum[:10])+".png")
	if err := os.WriteFile(path, pngBytes, 0o600); err != nil {
		return "", err
	}
	return path, nil
}

func (b *DesignBrowser) Close() error {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return nil
	}
	b.closed = true
	client := b.client
	cmd := b.cmd
	profileDir := b.profileDir
	done := b.done
	b.mu.Unlock()

	if client != nil {
		_ = client.close()
	}
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
	if done != nil {
		select {
		case <-done:
		case <-time.After(2 * time.Second):
		}
	}
	if profileDir != "" {
		_ = os.RemoveAll(profileDir)
	}
	return nil
}

type cdpClient struct {
	mu     sync.Mutex
	conn   *websocket.Conn
	nextID int64
}

func dialCDP(ctx context.Context, endpoint string) (*cdpClient, error) {
	dialer := *websocket.DefaultDialer
	conn, _, err := dialer.DialContext(ctx, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("connect Chromium DevTools: %w", err)
	}
	return &cdpClient{conn: conn}, nil
}

func (c *cdpClient) call(ctx context.Context, sessionID, method string, params any, out any) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.nextID++
	id := c.nextID
	request := map[string]any{"id": id, "method": method}
	if params != nil {
		request["params"] = params
	}
	if sessionID != "" {
		request["sessionId"] = sessionID
	}
	if deadline, ok := ctx.Deadline(); ok {
		_ = c.conn.SetWriteDeadline(deadline)
		_ = c.conn.SetReadDeadline(deadline)
	} else {
		_ = c.conn.SetWriteDeadline(time.Time{})
		_ = c.conn.SetReadDeadline(time.Time{})
	}
	if err := c.conn.WriteJSON(request); err != nil {
		return err
	}
	for {
		var response struct {
			ID     int64           `json:"id"`
			Result json.RawMessage `json:"result"`
			Error  *struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			} `json:"error,omitempty"`
		}
		if err := c.conn.ReadJSON(&response); err != nil {
			return err
		}
		if response.ID != id {
			continue
		}
		if response.Error != nil {
			return fmt.Errorf("CDP %s failed (%d): %s", method, response.Error.Code, response.Error.Message)
		}
		if out != nil && len(response.Result) > 0 {
			if err := json.Unmarshal(response.Result, out); err != nil {
				return fmt.Errorf("decode CDP %s result: %w", method, err)
			}
		}
		return nil
	}
}

func (c *cdpClient) close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

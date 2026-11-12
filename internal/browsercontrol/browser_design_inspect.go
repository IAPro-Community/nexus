package browsercontrol

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"math"
	"strings"
)

// DesignInspection contains a bounded, read-only view of one selected DOM
// element. The crop is JPEG encoded after sensitive regions have been masked.
type DesignInspection struct {
	Selector       string            `json:"selector"`
	Tag            string            `json:"tag"`
	Role           string            `json:"role,omitempty"`
	Name           string            `json:"name,omitempty"`
	Text           string            `json:"text,omitempty"`
	Attributes     map[string]string `json:"attributes,omitempty"`
	ComputedStyles map[string]string `json:"computed_styles"`
	MaskingApplied bool              `json:"masking_applied"`
	CropJPEG       []byte            `json:"crop_jpeg,omitempty"`
}

type viewportRect struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

type inspectPayload struct {
	Selector       string            `json:"selector"`
	Tag            string            `json:"tag"`
	Role           string            `json:"role"`
	Name           string            `json:"name"`
	Text           string            `json:"text"`
	Attributes     map[string]string `json:"attributes"`
	ComputedStyles map[string]string `json:"computedStyles"`
	Rect           viewportRect      `json:"rect"`
	Masks          []viewportRect    `json:"masks"`
}

// Inspect selects exactly one element by CSS selector, reads a deliberately
// limited DOM/style summary, then captures only that element's viewport crop.
// It never clicks, fills, submits, or returns input values/outerHTML.
func (s *Session) Inspect(ctx context.Context, selector string) (DesignInspection, error) {
	if strings.TrimSpace(selector) == "" || len(selector) > 2048 {
		return DesignInspection{}, errors.New("selector must be 1–2048 characters")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	quoted, _ := json.Marshal(selector)
	const expression = `(()=>{
  const selector=SELECTOR;
  let el;
  try { el=document.querySelector(selector); } catch { return {error:'invalid selector'}; }
  if(!el) return {error:'selector not found'};
  el.scrollIntoView({block:'center',inline:'center',behavior:'instant'});
  const r=el.getBoundingClientRect();
  const rect={x:r.x,y:r.y,width:r.width,height:r.height};
  const style=getComputedStyle(el);
  const styleKeys=['display','position','visibility','opacity','z-index','box-sizing','width','height','min-width','min-height','max-width','max-height','margin','padding','gap','color','background-color','font-family','font-size','font-weight','line-height','text-align','border','border-radius','overflow','flex-direction','grid-template-columns'];
  const computedStyles={};
  for(const key of styleKeys) computedStyles[key]=style.getPropertyValue(key);
  const attributes={};
  for(const name of ['id','class','role','aria-label','aria-describedby','data-testid']) {
    const value=el.getAttribute(name); if(value) attributes[name]=value.slice(0,500);
  }
  const sensitive=(node)=>node.matches('input,textarea,select,[contenteditable="true"]') ||
    node.matches('input[type="password"],input[type="email"],[autocomplete*="password" i],[autocomplete*="one-time-code" i],[autocomplete*="cc-" i]');
  const masks=[];
  for(const node of document.querySelectorAll('input,textarea,select,[contenteditable="true"],[autocomplete*="password" i],[autocomplete*="one-time-code" i],[autocomplete*="cc-" i]')) {
    const b=node.getBoundingClientRect();
    if(b.width>0&&b.height>0) masks.push({x:b.x,y:b.y,width:b.width,height:b.height});
  }
  // Mask text runs that contain common credential/PII patterns. The browser
  // crop is not treated as safe merely because the selected node is harmless.
  const pattern=/(?:bearer\s+\S+|(?:api[_ -]?key|access[_ -]?token|refresh[_ -]?token|token|secret|password|authorization|cookie|session[_ -]?id)\s*[:=]\s*\S+|[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}|(?:[A-Z0-9_-]{16,}\.){2}[A-Z0-9_-]{16,}|\b(?:sk-[A-Za-z0-9_-]{12,}|gh[pousr]_[A-Za-z0-9_]{16,}|xox[baprs]-[A-Za-z0-9-]{12,}|AIza[A-Za-z0-9_-]{20,})\b)/i;
  const walker=document.createTreeWalker(document.body||document,NodeFilter.SHOW_TEXT);
  let textNode; let count=0;
  while((textNode=walker.nextNode())&&count<500) {
    count++;
    if(!pattern.test(textNode.nodeValue||'')) continue;
    const range=document.createRange(); range.selectNodeContents(textNode);
    for(const b of range.getClientRects()) if(b.width>0&&b.height>0) masks.push({x:b.x,y:b.y,width:b.width,height:b.height});
  }
  const text=sensitive(el)?'':(el.innerText||el.textContent||'').slice(0,2000);
  const name=sensitive(el)?(el.getAttribute('aria-label')||''):((el.getAttribute('aria-label')||el.getAttribute('alt')||'').slice(0,300));
  return {selector,tag:el.tagName.toLowerCase(),role:el.getAttribute('role')||'',name,text,attributes,computedStyles,rect,masks};
})()`
	script := strings.Replace(expression, "SELECTOR", string(quoted), 1)
	var evaluation struct {
		Result struct {
			Value json.RawMessage `json:"value"`
		} `json:"result"`
		ExceptionDetails json.RawMessage `json:"exceptionDetails"`
	}
	if err := s.call(ctx, "Runtime.evaluate", map[string]any{"expression": script, "returnByValue": true, "awaitPromise": false}, &evaluation); err != nil {
		return DesignInspection{}, err
	}
	if len(evaluation.ExceptionDetails) > 0 || len(evaluation.Result.Value) == 0 {
		return DesignInspection{}, errors.New("design inspection failed")
	}
	var payload inspectPayload
	if err := json.Unmarshal(evaluation.Result.Value, &payload); err != nil {
		return DesignInspection{}, errors.New("design inspection response was invalid")
	}
	if payload.Selector == "" {
		var failure struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(evaluation.Result.Value, &failure)
		if failure.Error != "" {
			return DesignInspection{}, errors.New(failure.Error)
		}
		return DesignInspection{}, errors.New("selected element was unavailable")
	}
	r := payload.Rect
	if !finiteRect(r) || r.Width < 1 || r.Height < 1 {
		return DesignInspection{}, errors.New("selected element has no visible crop")
	}
	if r.X < 0 {
		r.Width += r.X
		r.X = 0
	}
	if r.Y < 0 {
		r.Height += r.Y
		r.Y = 0
	}
	if r.Width > 1280 {
		r.Width = 1280
	}
	if r.Height > 800 {
		r.Height = 800
	}
	if r.Width < 1 || r.Height < 1 {
		return DesignInspection{}, errors.New("selected element is outside the viewport")
	}
	scale := math.Min(1, math.Min(1280/r.Width, 800/r.Height))
	var shot struct {
		Data string `json:"data"`
	}
	if err := s.call(ctx, "Page.captureScreenshot", map[string]any{
		"format": "jpeg", "quality": 60, "fromSurface": true,
		"clip": map[string]any{"x": r.X, "y": r.Y, "width": r.Width, "height": r.Height, "scale": scale},
	}, &shot); err != nil {
		return DesignInspection{}, err
	}
	encoded, err := base64.StdEncoding.DecodeString(shot.Data)
	if err != nil {
		return DesignInspection{}, errors.New("could not decode inspected crop")
	}
	masked, err := maskDesignCrop(encoded, r, payload.Masks, scale)
	if err != nil {
		return DesignInspection{}, err
	}
	clean := DesignInspection{
		Selector: scrubBrowserText(payload.Selector), Tag: payload.Tag, Role: scrubBrowserText(payload.Role),
		Name: scrubBrowserText(payload.Name), Text: scrubBrowserText(payload.Text),
		Attributes: make(map[string]string), ComputedStyles: make(map[string]string), MaskingApplied: true, CropJPEG: masked,
	}
	for key, value := range payload.Attributes {
		clean.Attributes[key] = scrubBrowserText(value)
	}
	for key, value := range payload.ComputedStyles {
		clean.ComputedStyles[key] = scrubBrowserText(value)
	}
	return clean, nil
}

func finiteRect(r viewportRect) bool {
	return !math.IsNaN(r.X) && !math.IsNaN(r.Y) && !math.IsNaN(r.Width) && !math.IsNaN(r.Height) &&
		!math.IsInf(r.X, 0) && !math.IsInf(r.Y, 0) && !math.IsInf(r.Width, 0) && !math.IsInf(r.Height, 0)
}

func maskDesignCrop(encoded []byte, crop viewportRect, masks []viewportRect, scale float64) ([]byte, error) {
	decoded, err := jpeg.Decode(bytes.NewReader(encoded))
	if err != nil {
		return nil, fmt.Errorf("invalid inspected crop: %w", err)
	}
	bounds := decoded.Bounds()
	if bounds.Empty() || bounds.Dx() > 1280 || bounds.Dy() > 800 {
		return nil, errors.New("inspected crop dimensions exceed limit")
	}
	canvas := image.NewRGBA(bounds)
	draw.Draw(canvas, bounds, decoded, bounds.Min, draw.Src)
	maskColor := color.RGBA{R: 32, G: 36, B: 42, A: 255}
	for _, rect := range masks {
		if !finiteRect(rect) || rect.Width <= 0 || rect.Height <= 0 {
			continue
		}
		left := int(math.Floor((rect.X - crop.X) * scale))
		top := int(math.Floor((rect.Y - crop.Y) * scale))
		right := int(math.Ceil((rect.X + rect.Width - crop.X) * scale))
		bottom := int(math.Ceil((rect.Y + rect.Height - crop.Y) * scale))
		masked := image.Rect(left, top, right, bottom).Intersect(bounds)
		if !masked.Empty() {
			draw.Draw(canvas, masked, &image.Uniform{C: maskColor}, image.Point{}, draw.Src)
		}
	}
	var out bytes.Buffer
	if err := jpeg.Encode(&out, canvas, &jpeg.Options{Quality: 65}); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

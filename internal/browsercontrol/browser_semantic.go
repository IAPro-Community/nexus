package browsercontrol

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// querySelector resolves a CSS selector in the controlled Chrome page.
// All selectors are CDP data, never arbitrary script text.
func (s *Session) querySelector(ctx context.Context, selector string) (int, error) {
	if selector == "" || len(selector) > 2048 {
		return 0, errors.New("selector must be 1–2048 characters")
	}
	var doc struct {
		Root struct {
			NodeID int `json:"nodeId"`
		} `json:"root"`
	}
	if err := s.call(ctx, "DOM.getDocument", map[string]any{"depth": 0}, &doc); err != nil {
		return 0, err
	}
	var node struct {
		NodeID int `json:"nodeId"`
	}
	if err := s.call(ctx, "DOM.querySelector", map[string]any{"nodeId": doc.Root.NodeID, "selector": selector}, &node); err != nil {
		return 0, err
	}
	if node.NodeID == 0 {
		return 0, errors.New("selector not found")
	}
	return node.NodeID, nil
}

// clickSelector uses a real mouse event instead of HTMLElement.click().
// It improves compatibility with pointer-driven web applications.
func (s *Session) clickSelector(ctx context.Context, selector string) error {
	nodeID, err := s.querySelector(ctx, selector)
	if err != nil {
		return err
	}
	if err := s.call(ctx, "DOM.scrollIntoViewIfNeeded", map[string]any{"nodeId": nodeID}, nil); err != nil {
		return err
	}
	var box struct {
		Model struct {
			Content []float64 `json:"content"`
		} `json:"model"`
	}
	if err := s.call(ctx, "DOM.getBoxModel", map[string]any{"nodeId": nodeID}, &box); err != nil {
		return err
	}
	if len(box.Model.Content) != 8 {
		return errors.New("element has no clickable box")
	}
	var x, y float64
	for i := 0; i < 8; i += 2 {
		x += box.Model.Content[i]
		y += box.Model.Content[i+1]
	}
	x /= 4
	y /= 4
	button := map[string]any{"type": "mousePressed", "button": "left", "clickCount": 1, "x": x, "y": y}
	if err := s.call(ctx, "Input.dispatchMouseEvent", button, nil); err != nil {
		return err
	}
	button["type"] = "mouseReleased"
	return s.call(ctx, "Input.dispatchMouseEvent", button, nil)
}

// fillSelector uses a fixed, narrowly scoped expression with JSON-encoded
// arguments and dispatches input/change events for controlled React inputs.
func (s *Session) fillSelector(ctx context.Context, selector, text string) error {
	if selector == "" || len(selector) > 2048 || len(text) > 8192 {
		return errors.New("invalid selector or text size")
	}
	q, _ := json.Marshal(selector)
	t, _ := json.Marshal(text)
	const expression = `(()=>{
  const element=document.querySelector(SELECTOR);
  if(!element) return {ok:false,error:'selector not found'};
  if(!(element instanceof HTMLInputElement || element instanceof HTMLTextAreaElement))
   return {ok:false,error:'element is not an input or textarea'};
  if(element.disabled || element.readOnly) return {ok:false,error:'element cannot be edited'};
  element.focus();
  const prototype=element instanceof HTMLTextAreaElement ? HTMLTextAreaElement.prototype : HTMLInputElement.prototype;
  const setter=Object.getOwnPropertyDescriptor(prototype,'value')?.set;
  if(!setter) return {ok:false,error:'native input setter unavailable'};
  setter.call(element,TEXT);
  element.dispatchEvent(new Event('input',{bubbles:true}));
  element.dispatchEvent(new Event('change',{bubbles:true}));
  return {ok:true};
 })()`
	script := strings.ReplaceAll(strings.ReplaceAll(expression, "SELECTOR", string(q)), "TEXT", string(t))
	var result struct {
		Result struct {
			Value struct {
				OK    bool   `json:"ok"`
				Error string `json:"error"`
			} `json:"value"`
		} `json:"result"`
		ExceptionDetails json.RawMessage `json:"exceptionDetails"`
	}
	if err := s.call(ctx, "Runtime.evaluate", map[string]any{"expression": script, "returnByValue": true}, &result); err != nil {
		return err
	}
	if len(result.ExceptionDetails) > 0 {
		return errors.New("page rejected input")
	}
	if !result.Result.Value.OK {
		return fmt.Errorf("browser fill: %s", result.Result.Value.Error)
	}
	return nil
}

// waitSelector caps polling to five seconds and respects caller cancellation.
func (s *Session) waitSelector(ctx context.Context, selector string, timeoutMS int) error {
	if timeoutMS <= 0 || timeoutMS > 5000 {
		timeoutMS = 5000
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMS)*time.Millisecond)
	defer cancel()
	for {
		_, err := s.querySelector(ctx, selector)
		if err == nil {
			return nil
		}
		if !strings.Contains(err.Error(), "selector not found") {
			return err
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("selector timed out: %w", ctx.Err())
		case <-time.After(125 * time.Millisecond):
		}
	}
}

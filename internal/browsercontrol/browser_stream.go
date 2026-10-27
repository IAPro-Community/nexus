package browsercontrol

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/gorilla/websocket"
)

// SubscribeFrames shares one Chromium screencast across all current viewers.
// Frames are lossy by design: a slow mobile client skips old images rather
// than building an unbounded backlog. The stream does not own input control.
func (s *Session) SubscribeFrames() (<-chan []byte, func(), error) {
	s.mu.Lock()
	closed := s.closed
	s.mu.Unlock()
	if closed {
		return nil, nil, errors.New("browser session closed")
	}
	s.streamMu.Lock()
	if s.streamReaders == nil {
		s.streamReaders = make(map[chan []byte]struct{})
	}
	ch := make(chan []byte, 2)
	s.streamReaders[ch] = struct{}{}
	if s.streamCancel == nil {
		ctx, cancel := context.WithCancel(context.Background())
		s.streamCancel = cancel
		go s.streamLoop(ctx)
	}
	s.streamMu.Unlock()
	unsubscribe := func() {
		s.streamMu.Lock()
		if _, ok := s.streamReaders[ch]; ok {
			delete(s.streamReaders, ch)
			close(ch)
		}
		if len(s.streamReaders) == 0 && s.streamCancel != nil {
			s.streamCancel()
			s.streamCancel = nil
		}
		s.streamMu.Unlock()
	}
	return ch, unsubscribe, nil
}

func (s *Session) sendFrame(frame []byte) {
	s.streamMu.Lock()
	defer s.streamMu.Unlock()
	for ch := range s.streamReaders {
		select {
		case ch <- frame:
		default:
			// Drop the oldest frame. Never block the CDP read/ack loop.
			select {
			case <-ch:
			default:
			}
			select {
			case ch <- frame:
			default:
			}
		}
	}
}

func (s *Session) stopStream() {
	s.streamMu.Lock()
	if s.streamCancel != nil {
		s.streamCancel()
		s.streamCancel = nil
	}
	s.streamMu.Unlock()
}

// streamLoop reconnects when the DevTools socket is interrupted. A new
// stream producer is created only when the previous one has stopped.
// Browser shutdown cancels the loop before killing Chromium.
func (s *Session) streamLoop(ctx context.Context) {
	for {
		if ctx.Err() != nil {
			return
		}
		err := s.streamOnce(ctx)
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			// Make transient Chrome/DevTools interruptions recoverable.
			select {
			case <-ctx.Done():
				return
			case <-time.After(650 * time.Millisecond):
			}
		}
	}
}

func (s *Session) streamOnce(ctx context.Context) error {
	dialer := websocket.Dialer{HandshakeTimeout: 5 * time.Second}
	conn, _, err := dialer.DialContext(ctx, s.wsURL, nil)
	if err != nil {
		return fmt.Errorf("screencast CDP connection: %w", err)
	}
	defer conn.Close()
	stopClose := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			_ = conn.Close()
		case <-stopClose:
		}
	}()
	defer close(stopClose)
	if err := conn.WriteJSON(map[string]any{
		"id": 1, "method": "Page.startScreencast",
		"params": map[string]any{"format": "jpeg", "quality": 65, "everyNthFrame": 1},
	}); err != nil {
		return err
	}

	// Chromium sends Page.screencastFrame events while the page changes.
	// Ack is mandatory; without it Chrome throttles or stops the stream.
	nextID := 2
	lastPublished := time.Time{}
	for {
		var message struct {
			Method string `json:"method"`
			Params struct {
				Data      string `json:"data"`
				SessionID int    `json:"sessionId"`
			} `json:"params"`
			Error *struct {
				Message string `json:"message"`
			} `json:"error"`
			ID int `json:"id"`
		}
		if err := conn.ReadJSON(&message); err != nil {
			return err
		}
		if message.ID == 1 && message.Error != nil {
			return errors.New(message.Error.Message)
		}
		if message.Method != "Page.screencastFrame" {
			continue
		}
		if message.Params.Data != "" && (lastPublished.IsZero() || time.Since(lastPublished) >= 100*time.Millisecond) {
			// Limit remote delivery to 10fps without blocking CDP frame acks.
			// The CDP payload is already JPEG; decode base64 on clients.
			s.sendFrame([]byte(message.Params.Data))
			lastPublished = time.Now()
		}
		if err := conn.WriteJSON(map[string]any{
			"id": nextID, "method": "Page.screencastFrameAck",
			"params": map[string]any{"sessionId": message.Params.SessionID},
		}); err != nil {
			return err
		}
		nextID++
	}
}

// EncodeFrameEnvelope keeps a forward-compatible wire shape.
func EncodeFrameEnvelope(data []byte) ([]byte, error) {
	return json.Marshal(map[string]any{"type": "frame", "data": string(data)})
}

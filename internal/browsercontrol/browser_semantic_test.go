package browsercontrol

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gorilla/websocket"
)

func TestQuerySelectorRetriesWhenNavigationReplacesDocument(t *testing.T) {
	var documentReads atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := (&websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}).Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		for {
			var request struct {
				ID     int            `json:"id"`
				Method string         `json:"method"`
				Params map[string]any `json:"params"`
			}
			if err := conn.ReadJSON(&request); err != nil {
				return
			}
			response := map[string]any{"id": request.ID, "result": map[string]any{}}
			switch request.Method {
			case "DOM.getDocument":
				rootID := documentReads.Add(1)
				response["result"] = map[string]any{"root": map[string]any{"nodeId": rootID}}
			case "DOM.querySelector":
				rootID, ok := request.Params["nodeId"].(float64)
				if !ok {
					return
				}
				if int32(rootID) == 1 {
					response["error"] = map[string]string{"message": "Could not find node with given id"}
				} else {
					response["result"] = map[string]any{"nodeId": 42}
				}
			default:
				return
			}
			if err := conn.WriteJSON(response); err != nil {
				return
			}
		}
	}))
	defer server.Close()

	session := &Session{wsURL: "ws" + strings.TrimPrefix(server.URL, "http")}
	nodeID, err := session.querySelector(context.Background(), "#action")
	if err != nil {
		t.Fatalf("querySelector after replaced document: %v", err)
	}
	if nodeID != 42 {
		t.Fatalf("node ID=%d, want 42", nodeID)
	}
	if got := documentReads.Load(); got != 2 {
		t.Fatalf("document reads=%d, want 2", got)
	}
	if session.cdpConn != nil {
		_ = session.cdpConn.Close()
	}
}

package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/IAPro-Community/nexus/internal/browsercontrol"
)

func TestBrowserMCPRejectsUnauthenticatedAndRemoteRequests(t *testing.T) {
	srv := &Server{browsers: browsercontrol.NewManager()}
	request := func(host, bearer string, origin string, forwarded bool) *http.Request {
		req := httptest.NewRequest(http.MethodPost,
			"http://127.0.0.1:13000/api/v1/browser/mcp",
			strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`))
		req.Host = host
		req.RemoteAddr = "127.0.0.1:45678"
		if bearer != "" {
			req.Header.Set("Authorization", "Bearer "+bearer)
		}
		if origin != "" {
			req.Header.Set("Origin", origin)
		}
		if forwarded {
			req.Header.Set("X-Forwarded-Host", "remote.example")
		}
		return req
	}
	cases := []struct {
		Name    string
		Request *http.Request
		Status  int
	}{
		{"missing bearer", request("127.0.0.1:13000", "", "", false), http.StatusUnauthorized},
		{"bad bearer", request("127.0.0.1:13000", "nb_invalid", "", false), http.StatusUnauthorized},
		{"internet hostname", request("nexus.remote.example", "nb_invalid", "", false), http.StatusForbidden},
		{"cross origin", request("127.0.0.1:13000", "nb_invalid", "https://evil.example", false), http.StatusForbidden},
		{"proxied", request("127.0.0.1:13000", "nb_invalid", "", true), http.StatusForbidden},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			srv.handleBrowserMCP(recorder, tt.Request)
			if recorder.Code != tt.Status {
				t.Fatalf("got HTTP %d expected %d: %s", recorder.Code, tt.Status, recorder.Body.String())
			}
		})
	}
}

func TestBrowserMCPToolSchemas(t *testing.T) {
	tools := browserMCPTools()
	names := make(map[string]bool)
	for _, tool := range tools {
		name, ok := tool["name"].(string)
		if !ok {
			t.Fatal("tool without name")
		}
		if names[name] {
			t.Fatalf("duplicate tool %s", name)
		}
		names[name] = true
		schema, ok := tool["inputSchema"].(map[string]any)
		if !ok {
			t.Fatalf("missing schema for %s", name)
		}
		if _, ok := schema["required"].([]string); !ok {
			t.Fatalf("invalid required list for %s", name)
		}
	}
	for _, name := range []string{"browser_request_human", "browser_wait_for_human", "browser_snapshot", "browser_screenshot", "browser_request_control", "browser_fill"} {
		if !names[name] {
			t.Fatalf("missing required capability %s", name)
		}
	}
}

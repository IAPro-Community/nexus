package main

import (
	"fmt"
	"context"
	"github.com/IAPro-Community/nexus/internal/browsercontrol"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/IAPro-Community/nexus/internal/app"
	"github.com/IAPro-Community/nexus/internal/browser"
)

func main() {
	// Enable debug logging when NEXUS_DEBUG=1 or NEXUS_AGY_DEBUG=1
	if os.Getenv("NEXUS_DEBUG") == "1" || os.Getenv("NEXUS_AGY_DEBUG") == "1" {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})))
	}

	// Private MCP subprocess invoked by registered provider drivers. This
	// is not exposed as a user shell command and writes JSON-RPC to stdout.
	if len(os.Args) == 2 && os.Args[1] == "__browser-mcp-stdio" {
		if err := browsercontrol.ServeBrowserMCPStdio(context.Background(), os.Stdin, os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, "Nexus Browser MCP:", err)
			os.Exit(1)
		}
		return
	}
	base := filepath.Base(os.Args[0])
	helperBase := strings.TrimSuffix(strings.ToLower(base), ".exe")
	if helperBase == "nexus-browser" || helperBase == "ai-browser" || helperBase == "xdg-open" {
		// Set env var so child processes and browser.Open() can detect
		// recursion without relying solely on binary name heuristics.
		os.Setenv("NEXUS_BROWSER_HELPER", "1")
		if err := browser.Open(os.Args[1:]); err != nil {
			fmt.Fprintln(os.Stderr, "Nexus browser helper:", err)
			os.Exit(1)
		}
		return
	}

	// Strip debug env vars from args that might confuse flag parsing
	var cleanArgs []string
	for _, arg := range os.Args[1:] {
		if !strings.HasPrefix(arg, "NEXUS_DEBUG=") && !strings.HasPrefix(arg, "NEXUS_AGY_DEBUG=") {
			cleanArgs = append(cleanArgs, arg)
		}
	}

	if err := app.Run(cleanArgs); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", base, err)
		os.Exit(1)
	}
}

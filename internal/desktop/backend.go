package desktop

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"time"

	"github.com/IAPro-Community/nexus/internal/app"
	"github.com/IAPro-Community/nexus/internal/control/web"
)

const sharedBackendReadyTimeout = 10 * time.Second

// SharedBackend represents the authenticated Core/API surface consumed by a
// native Wails presentation. It prefers attaching to an existing loopback Web
// Core and only starts an embedded Core when no reusable process is available.
type SharedBackend struct {
	Assets    fs.FS
	Handler   http.Handler
	ServerURL string
	Session   *web.Session

	core   *app.Core
	cancel context.CancelFunc
}

// OpenSharedBackend keeps Nexus Desktop and Nexus HUD on the same Core whenever
// possible. Starting an embedded Core is a lifecycle fallback, not a duplicate
// domain implementation; singleton leases in canonical services remain
// authoritative.
func OpenSharedBackend(surfaceName string) (*SharedBackend, error) {
	assets, err := web.EmbeddedDistFS()
	if err != nil {
		return nil, fmt.Errorf("access embedded web assets: %w", err)
	}

	if state, stateErr := web.ReadListenState(); stateErr == nil {
		if sess, attachErr := web.AttachLoopbackSession(state); attachErr == nil {
			if handler, proxyErr := web.NewLoopbackBackendHandler(state.URL, assets); proxyErr == nil {
				fmt.Fprintf(os.Stderr, "%s attached to existing Web Core at %s\n", surfaceName, state.URL)
				return &SharedBackend{
					Assets:    assets,
					Handler:   handler,
					ServerURL: state.URL,
					Session:   sess,
				}, nil
			}
		}
	}

	core, err := app.NewCore(app.CoreConfig{
		Host:   "127.0.0.1",
		Port:   0,
		NoOpen: true,
	})
	if err != nil {
		return nil, fmt.Errorf("initialize Nexus Core: %w", err)
	}

	coreCtx, cancel := context.WithCancel(context.Background())
	go func() {
		if runErr := core.Start(coreCtx); runErr != nil && coreCtx.Err() == nil {
			fmt.Fprintf(os.Stderr, "%s embedded Core exited: %v\n", surfaceName, runErr)
		}
	}()

	select {
	case <-core.Ready():
	case <-time.After(sharedBackendReadyTimeout):
		cancel()
		return nil, fmt.Errorf("timeout waiting for Nexus Core to be ready")
	}

	session, err := core.CreateDesktopSession()
	if err != nil {
		cancel()
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer shutdownCancel()
		_ = core.Stop(shutdownCtx)
		return nil, fmt.Errorf("provision desktop session: %w", err)
	}

	fmt.Fprintf(os.Stderr, "%s started an embedded Core at %s\n", surfaceName, core.URL())
	return &SharedBackend{
		Assets:    assets,
		Handler:   core.Handler(),
		ServerURL: core.URL(),
		Session:   session,
		core:      core,
		cancel:    cancel,
	}, nil
}

// Close only stops a Core owned by this presentation. Attached Web Core
// processes are never terminated by the Desktop/HUD surface.
func (b *SharedBackend) Close(ctx context.Context) error {
	if b == nil || b.core == nil {
		return nil
	}
	if b.cancel != nil {
		b.cancel()
	}
	return b.core.Stop(ctx)
}

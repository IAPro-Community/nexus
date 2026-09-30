package app

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/web"
	"github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/localization"
	"github.com/IAPro-Community/nexus/internal/nexus"
)

// CoreConfig encapsulates runtime options for the reusable Nexus Core.
type CoreConfig struct {
	Host   string
	Port   int
	NoOpen bool
	Remote bool
	// DisableHostsFile prevents embedded/native surfaces from requiring
	// administrator privileges just to register the optional nexus.dev alias.
	DisableHostsFile bool
	Language         string
	TunnelHost       string // Cloudflare Quick Tunnel public hostname
}

// Core encapsulates the entire Nexus backend and control server lifecycle.
// It is shared between the CLI ('nexus web' / 'nexus serve') and the native Desktop shell.
type Core struct {
	mu        sync.Mutex
	cfg       CoreConfig
	server    *web.Server
	readyChan chan struct{}
	stopChan  chan struct{}
	stopped   bool
}

// NewCore initializes the Nexus Core instance without starting background listeners.
func NewCore(cfg CoreConfig) (*Core, error) {
	initRegistry()
	appCfg, _ := config.LoadConfig()
	lang := cfg.Language
	if lang == "" {
		lang = appCfg.Language
	}
	localization.Set(localization.Resolve("", lang))

	if cfg.Host == "" {
		cfg.Host = "127.0.0.1"
	}

	return &Core{
		cfg:       cfg,
		readyChan: make(chan struct{}),
		stopChan:  make(chan struct{}),
	}, nil
}

// Start launches the Nexus Control API server and signals Ready() when listening.
func (c *Core) Start(ctx context.Context) error {
	nexus.Default().StartQuotaMonitor(ctx)
	nexus.Default().StartCapacityMonitor(ctx)
	c.mu.Lock()
	srv, err := web.NewServer(web.ServerOptions{
		Host:             c.cfg.Host,
		Port:             c.cfg.Port,
		NoOpen:           c.cfg.NoOpen,
		Remote:           c.cfg.Remote,
		DisableHostsFile: c.cfg.DisableHostsFile,
		TunnelHost:       c.cfg.TunnelHost,
	})
	if err != nil {
		c.mu.Unlock()
		return fmt.Errorf("failed to start Nexus Core server: %w", err)
	}
	c.server = srv
	close(c.readyChan)
	c.mu.Unlock()

	// If a context was provided, wait for context cancellation in background
	go func() {
		select {
		case <-ctx.Done():
			_ = c.Stop(context.Background())
		case <-c.stopChan:
		}
	}()

	return srv.Start()
}

// Ready returns a channel that is closed as soon as the Core server is bound and ready for requests.
func (c *Core) Ready() <-chan struct{} {
	return c.readyChan
}

// URL returns the base loopback URL of the running server.
func (c *Core) URL() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.server == nil {
		return ""
	}
	return c.server.URL()
}

// BootstrapURL returns the authenticated one-time bootstrap URL.
func (c *Core) BootstrapURL() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.server == nil {
		return ""
	}
	return c.server.BootstrapURL()
}

// Handler returns the HTTP handler of the underlying Nexus Core server.
func (c *Core) Handler() http.Handler {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.server == nil {
		return nil
	}
	return c.server.Handler()
}

// CreateDesktopSession provisions a pre-authenticated session for the native desktop shell.
func (c *Core) CreateDesktopSession() (*web.Session, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.server == nil {
		return nil, fmt.Errorf("nexus core server not initialized")
	}
	return c.server.CreateDesktopSession()
}

// Stop gracefully shuts down the Core server and clears loopback state.
func (c *Core) Stop(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.stopped {
		return nil
	}
	c.stopped = true
	close(c.stopChan)
	nexus.Default().StopQuotaMonitor()

	if c.server != nil {
		shutdownCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()
		if err := c.server.Shutdown(shutdownCtx); err != nil {
			return err
		}
	}

	// Shut down Nexus background services and close the SQLite store.
	nexus.Default().Shutdown()

	return nil
}

// HasActiveWork returns true if any agent or mission is actively running.
func (c *Core) HasActiveWork() (bool, string) {
	return false, ""
}

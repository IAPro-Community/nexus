package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/web"
)

// extractTunnelHost extracts the hostname from a tunnel URL like https://xxx.trycloudflare.com
func extractTunnelHost(tunnelURL string) string {
	if strings.HasPrefix(tunnelURL, "https://") {
		url := strings.TrimPrefix(tunnelURL, "https://")
		if idx := strings.Index(url, "/"); idx >= 0 {
			return url[:idx]
		}
		return url
	}
	return ""
}

func tunnelCmd(args []string) error {
	port := web.DefaultPort
	host := "127.0.0.1"

	for i := 0; i < len(args); i++ {
		switch {
		case (args[i] == "--port" || args[i] == "-p") && i+1 < len(args):
			p, err := strconv.Atoi(args[i+1])
			if err == nil && p > 0 && p <= 65535 {
				port = p
			}
			i++
		case (args[i] == "--host" || args[i] == "-l") && i+1 < len(args):
			host = args[i+1]
			i++
		case args[i] == "--help" || args[i] == "-h":
			tunnelHelp()
			return nil
		}
	}

	ctx := context.Background()

	// 1. Check cloudflared
	fmt.Fprintf(os.Stderr, "🔍 Verificando cloudflared...\n")
	binPath, err := web.EnsureCloudflared(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "✗ %v\n", err)
		fmt.Fprintf(os.Stderr, "\nInstale manualmente:\n")
		fmt.Fprintf(os.Stderr, "  https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/downloads/\n")
		return err
	}
	ver := web.CloudflaredVersion(binPath)
	if ver != "" {
		fmt.Fprintf(os.Stderr, "✓ cloudflared encontrado (%s)\n", ver)
	} else {
		fmt.Fprintf(os.Stderr, "✓ cloudflared encontrado em %s\n", binPath)
	}

	// 2. Start tunnel FIRST (to get the public URL)
	fmt.Fprintf(os.Stderr, "🔗 Iniciando Cloudflare Quick Tunnel...\n")
	tunnel, err := web.StartTunnel(ctx, port)
	if err != nil {
		return fmt.Errorf("failed to start tunnel: %w", err)
	}

	// Extract tunnel hostname for origin validation
	tunnelHost := extractTunnelHost(tunnel.URL)
	if tunnelHost == "" {
		_ = tunnel.Stop()
		return fmt.Errorf("failed to extract tunnel hostname from %s", tunnel.URL)
	}

	// 3. Start web server WITH tunnel host for origin validation
	fmt.Fprintf(os.Stderr, "🌐 Iniciando Nexus Web Server em %s:%d...\n", host, port)

	core, err := NewCore(CoreConfig{
		Host:       host,
		Port:       port,
		NoOpen:     true,
		Remote:     true,
		TunnelHost: tunnelHost,
	})
	if err != nil {
		_ = tunnel.Stop()
		return fmt.Errorf("failed to start Nexus Core: %w", err)
	}

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- core.Start(ctx)
	}()

	// Wait for server ready
	select {
	case <-core.Ready():
		fmt.Fprintf(os.Stderr, "✓ Nexus Web Server rodando em %s\n", core.URL())
	case err := <-serverErr:
		_ = tunnel.Stop()
		return fmt.Errorf("failed to start web server: %w", err)
	case <-time.After(10 * time.Second):
		_ = tunnel.Stop()
		return fmt.Errorf("timeout waiting for web server to start")
	}

	// 4. Wait for tunnel DNS propagation and connectivity (AFTER server is ready)
	fmt.Fprintf(os.Stderr, "⏳ Aguardando DNS propagar...\n")
	readyCtx, readyCancel := context.WithTimeout(ctx, 60*time.Second)
	defer readyCancel()
	if err := tunnel.WaitForTunnel(readyCtx); err != nil {
		_ = tunnel.Stop()
		_ = core.Stop(context.Background())
		return fmt.Errorf("tunnel URL not reachable: %w", err)
	}
	fmt.Fprintf(os.Stderr, "✓ Túnel conectado e acessível!\n")
	// CLI and Settings share the same registered tunnel generation.
	if err := core.AttachRemoteTunnel(tunnel); err != nil {
		_ = tunnel.Stop()
		_ = core.Stop(context.Background())
		return fmt.Errorf("failed to register tunnel for invitations: %w", err)
	}

	// 5. Print a fresh one-time invitation, valid for 15 minutes.
	remoteBootstrapURL, err := core.NewRemoteInvitationURL()
	if err != nil {
		_ = tunnel.Stop()
		_ = core.Stop(context.Background())
		return fmt.Errorf("failed to issue remote invitation: %w", err)
	}
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "╔═══════════════════════════════════════════════════════════════╗\n")
	fmt.Fprintf(os.Stderr, "║  URL pública:      %-44s ║\n", tunnel.URL)
	fmt.Fprintf(os.Stderr, "║  Convite único:     %-44s ║\n", remoteBootstrapURL)
	fmt.Fprintf(os.Stderr, "║  Duração: 15 minutos, um único acesso.                        ║\n")
	fmt.Fprintf(os.Stderr, "╚═══════════════════════════════════════════════════════════════╝\n")
	fmt.Fprintf(os.Stderr, "\nPressione Ctrl+C para encerrar.\n")

	// 5. Wait for Ctrl+C
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Fprintf(os.Stderr, "\nEncerrando tunnel...\n")

	// 6. Cleanup
	if err := tunnel.Stop(); err != nil {
		fmt.Fprintf(os.Stderr, "Aviso: erro ao parar tunnel: %v\n", err)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := core.Stop(shutdownCtx); err != nil {
		fmt.Fprintf(os.Stderr, "Aviso: erro ao parar servidor: %v\n", err)
	}

	fmt.Fprintf(os.Stderr, "✓ Encerrado.\n")
	return nil
}

func tunnelHelp() {
	p := progName()
	fmt.Fprintf(os.Stderr, `Nexus Tunnel — Acesso remoto gratuito via Cloudflare Quick Tunnel

USO:
  %s tunnel [flags]

.DESCRIÇÃO:
  Inicia o Nexus Web Server e cria um túnel temporário via Cloudflare,
  gerando uma URL pública (https://xxx.trycloudflare.com) acessível
  de qualquer dispositivo. Não requer conta, domínio ou configuração.

  O cloudflared é baixado automaticamente na primeira execução (~30MB).

FLAGS:
  --port, -p <port>    Porta do servidor local (padrão: %d)
  --host, -l <ip>      Endereço de escuta (padrão: 127.0.0.1)
  -h, --help           Mostrar esta ajuda

EXEMPLOS:
  %s tunnel
  %s tunnel --port 8080
  %s tunnel --port 3000

NOTAS:
  • A URL pública é temporária e muda a cada execução
  • O token de autenticação do Nexus continua obrigatório
  • Ctrl+C encerra o tunnel e o servidor
`, p, web.DefaultPort, p, p, p)
}

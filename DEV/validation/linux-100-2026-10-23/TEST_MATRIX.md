# Matriz de testes — Linux AMD64

Todos os testes abaixo referem-se ao HEAD remoto `bf6adf0c80ac3531d7291844512a14a017ab0002` mais o worktree local preservado (dirty). `SHA` é o prefixo desse commit-base; hashes dos artefatos instalados estão no relatório principal. Severidade descreve impacto no gate, não um ranking de CVE.

| ID | Área / comando ou procedimento | Esperado → observado | Status | Evidência | Sev. | SHA |
|---|---|---|---|---|---|---|
| SYNC-01 | `git fetch origin --prune`; comparar HEAD e tracking remoto | Branch release sincronizada → `HEAD` e remoto em `bf6adf0c…0002` | PASS | `logs/release-history-final.txt` | — | `bf6adf0c` |
| ENV-01 | `uname`, `/etc/os-release`, sessão gráfica, toolchains e RAM/CPU | Linux AMD64 e dependências identificadas → Ubuntu 25.10, GNOME/Wayland, GPU VMware sem 3D | PASS | `ENVIRONMENT.md` | — | `bf6adf0c` |
| BUILD-CLI | `GOTOOLCHAIN=go1.26.10 make build` | CLI compila → exit 0 | PASS | `logs/build-final-go1.26.10-bf6adf0c.log` | — | `bf6adf0c` |
| BUILD-DESKTOP | `GOTOOLCHAIN=go1.26.10 make build-desktop` | Desktop Wails compila → exit 0 | PASS | `logs/build-final-go1.26.10-bf6adf0c.log` | — | `bf6adf0c` |
| BUILD-HUD | `GOTOOLCHAIN=go1.26.10 make build-hud` | HUD compila → exit 0 | PASS | `logs/build-final-go1.26.10-bf6adf0c.log` | — | `bf6adf0c` |
| BUILD-ELF | `file`, `ldd` nos três artefatos | AMD64 e libs resolvidas → ELF x86-64, sem `not found` | PASS | `logs/install-final-go1.26.10-bf6adf0c.log`, `logs/native-libraries-bf6adf0c.log` | — | `bf6adf0c` |
| INSTALL-LOCAL | Instalação atômica nos dois prefixos locais | Os dois conjuntos iguais → três hashes coincidem em cada caminho | PASS | `logs/install-final-go1.26.10-bf6adf0c.log` | — | `bf6adf0c` |
| VERSION-01 | `nexus version --json` | Versão, OS, arquitetura e toolchain identificáveis → beta.10 / linux / amd64 / Go 1.26.10 | PASS | `logs/processes-final-go1.26.10-bf6adf0c.log` | — | `bf6adf0c` |
| GO-TEST | `GOTOOLCHAIN=go1.26.10 make quality-full` — suíte Go | Pacotes Go passam → suite completa exit 0 | PASS | `logs/quality-full-final-go1.26.10-bf6adf0c.log` | — | `bf6adf0c` |
| GO-RACE | `make quality-full` — `go test -race -tags=gtk3 ./...` | Sem data races → suite completa exit 0 | PASS | `logs/quality-full-final-go1.26.10-bf6adf0c.log` | — | `bf6adf0c` |
| GO-E2E | `GOTOOLCHAIN=go1.26.10 make test-e2e` | Testes E2E Go com race em terminal/protocol/host/Web → exit 0 | PASS | `logs/test-e2e-go1.26.10-bf6adf0c.log` | — | `bf6adf0c` |
| GO-VET | `GOTOOLCHAIN=go1.26.10 make vet` | `go vet` sem findings → exit 0 | PASS | `logs/vet-go1.26.10-bf6adf0c.log` | — | `bf6adf0c` |
| GO-SECURITY | `GOTOOLCHAIN=go1.26.10 make security` | Scanner não identifica vulnerabilidade alcançável → exit 0 após fix do fallback | PASS | `logs/security-make-go1.26.10-bf6adf0c.log` | — | `bf6adf0c` |
| WEB-QUALITY | `make quality-full` — format, lint, typecheck, testes e build | Frontend completo → 99 arquivos e 612 testes aprovados | PASS | `logs/quality-full-final-go1.26.10-bf6adf0c.log` | — | `bf6adf0c` |
| WEB-E2E-01 | `bun run --cwd web test:e2e` | Fluxos de hardening/responsividade → axe sem violações menores; 7 breakpoints sem obstrução; menus Escape | PASS | `logs/browser-e2e-hardening-bf6adf0c.log`, `screenshots/e2e/hardening/` | — | `bf6adf0c` |
| WEB-E2E-02 | `bun run --cwd web test:a11y` | Acessibilidade/teclado → PASS, axe 0 violações menores | PASS | `logs/browser-e2e-a11y-bf6adf0c.log` | — | `bf6adf0c` |
| WEB-E2E-03 | `bun run --cwd web test:visual` | Breakpoints/layout → PASS em 320–1440 px | PASS | `logs/browser-e2e-visual-bf6adf0c.log`, capturas em `screenshots/e2e/` | — | `bf6adf0c` |
| WEB-E2E-04 | `bun run --cwd web test:e2e:capacity` | Verificações sintéticas de capacidade → PASS | PASS | `logs/browser-e2e-capacity-bf6adf0c.log` | — | `bf6adf0c` |
| WEB-E2E-05 | `bun run --cwd web test:e2e:hud` | HUD Web responsivo → PASS | PASS | `logs/browser-e2e-hud-bf6adf0c.log`, `screenshots/e2e/hud/` | — | `bf6adf0c` |
| NATIVE-01 | Iniciar `/home/desenvolvedor/.local/bin/nexus-desktop` na sessão GNOME | Wails abre → Desktop PID 565592 ativo | PASS | `logs/processes-final-go1.26.10-bf6adf0c.log`; capturas nativas anteriores em `screenshots/native-main-*.png` | — | `bf6adf0c` |
| NATIVE-02 | `GET http://127.0.0.1:43253/api/v1/health` | Core embarcado responde → HTTP 200, `1.0.0-beta.10` | PASS | `logs/processes-final-go1.26.10-bf6adf0c.log` | — | `bf6adf0c` |
| NATIVE-03 | Iniciar `nexus web --port 0`; GET `/api/v1/health` | Servidor Web responde → HTTP 200, `1.0.0-beta.10` | PASS | `logs/processes-final-go1.26.10-bf6adf0c.log`; porta `44065` | — | `bf6adf0c` |
| AUTH-01 | Abrir diálogo nativo e copiar comando de login Codex | Um único comando correto → texto/clipboard `nexus login codex` | PASS | `screenshots/native-login-command-codex-xwayland-bf6adf0c.png` | — | `bf6adf0c` |
| AUTH-02 | Importação Codex em perfil temporário isolado | Cópia segura sem alterar origem → perfil autenticado, `auth.json` 0600, origem intacta | PASS | `RUNTIME_VALIDATION.md`, captura `screenshots/native-main-import-result-bf6adf0c.png` | — | `bf6adf0c` |
| UI-01 | Abrir o Project Hub real e verificar descrição/ações | Texto não duplicado e alinhado → descrição em linha própria | PASS | `screenshots/native-main-home-xwayland-9929bef2-after-ui-fix.png` | — | `bf6adf0c` |
| RUNTIME-01 | Parar 4 hosts e iniciar mesmos perfis com `--continue` | Sessões preservadas → quatro runtimes novos em RUNNING | PASS | `logs/runtimes-final-go1.26.10-bf6adf0c.tsv` | — | `bf6adf0c` |
| RUNTIME-02 | Comparar `/proc/PID/exe` dos hosts, Desktop, HUD e Web | Processos usam hashes instalados → todos coincidem | PASS | `logs/processes-final-go1.26.10-bf6adf0c.log` | — | `bf6adf0c` |
| HUD-01 | Iniciar HUD nativo e consultar Core | HUD abre → PID 610408, HTTP 200, beta.10 | PASS | `RUNTIME_VALIDATION.md`, `logs/processes-final-go1.26.10-bf6adf0c.log` | — | `bf6adf0c` |
| HUD-02 | Abrir preferências HUD nativas | Modos disponíveis visíveis → Barra, Mascote e Bandeja; aviso de indisponibilidade Wayland | PASS | `screenshots/native-hud-after-import-bf6adf0c.png` | — | `bf6adf0c` |
| DOC-01 | `docs-verify` em checkout limpo do SHA | Manifesto e referências válidos → 197 docs / 33 arquivos requeridos aprovados | PASS | `logs/docs-verify-clean-bf6adf0c.log` | — | `bf6adf0c` |
| E2E-COMPOSER | `bun run --cwd web test:e2e:composer` | Criar sessão e abrir Composer → timeout aguardando botão “Nova elaboração” ausente na UI | FAIL | `logs/browser-e2e-composer-bf6adf0c.log` | P2 | `bf6adf0c` |
| E2E-SURFACE | `bun run --cwd web test:e2e:surface` | Layout mantém aba de produto fixada → assertion de ownership falhou | FAIL | `logs/browser-e2e-surface-bf6adf0c.log`, `screenshots/e2e/surface/` | P2 | `bf6adf0c` |
| E2E-SPINE | `bun run --cwd web test:e2e:composer-spine` | Fluxo de execução Composer → falhou; surface/controle esperado não encontrado | FAIL | `logs/browser-e2e-spine-bf6adf0c.log` | P2 | `bf6adf0c` |
| E2E-VAGUE | `bun run --cwd web test:e2e:composer-vague-spine` | Fluxo de intenção vaga → timeout no placeholder do Composer | FAIL | `logs/browser-e2e-vague-bf6adf0c.log` | P2 | `bf6adf0c` |
| E2E-THEMES | `bun run --cwd web test:e2e:composer-themes` — Ember Studio e Nexus Dusk observados | Preset abre superfície Composer → ambos falharam aguardando `.nx-composer-surface`; interrompido após repetição do mesmo erro | FAIL | `logs/browser-e2e-themes-bf6adf0c.log` | P2 | `bf6adf0c` |
| COMP-01 | Inspecionar build e render real do Companion 2D da PR #55 | Companion integrado → só GuardianMascot antigo; sem NexusGuardian/modelo/atlas contextual | FAIL | `COMPANION_VALIDATION.md`, `logs/companion-source-audit.txt` | P1 | `bf6adf0c` |
| AUTH-UI | Observar resultado da importação no diálogo nativo | Confirmação visível → não apareceu; importação interna foi confirmada | FAIL | `FIXES_APPLIED.md`, `screenshots/native-main-import-result-bf6adf0c.png` | P2 | `bf6adf0c` |
| WAYLAND-01 | Interagir com popover independente via GNOME Wayland | Popover destacado operável → interface informa indisponibilidade nesse compositor | BLOCKED | `HUD_VALIDATION.md`, captura de preferências | P2 | `bf6adf0c` |
| WAYLAND-02 | Validar click-through/interação de janela HUD no Wayland | Clique atravessa áreas transparentes → sem ferramenta de interação nativa disponível | BLOCKED | `HUD_VALIDATION.md` | P2 | `bf6adf0c` |
| X11-01 | Repetir HUD em sessão X11 separada | Segunda sessão gráfica → host não disponibiliza X11 independente | BLOCKED | `ENVIRONMENT.md` | P2 | `bf6adf0c` |
| DIST-01 | Harness oficial GoReleaser/DEB | Pacote suportado do worktree → harness exige checkout limpo e `--clean`; `dist/` tinha artefatos preexistentes preservados | BLOCKED | `DISTRIBUTION_VALIDATION.md` | P1 | `bf6adf0c` |
| DIST-02 | Atualização GitHub Release A→B | Manifesto/artefato estável assinado → release estável v1.0.0 não disponível para teste | BLOCKED | `DISTRIBUTION_VALIDATION.md` | P1 | `bf6adf0c` |
| PROVIDER-01 | Requisições reais controladas aos provedores | Prova de roteamento externo → não executadas sobre as sessões reais em andamento/quota do usuário | BLOCKED | `RUNTIME_VALIDATION.md` | P2 | `bf6adf0c` |
| THEME-NATIVE | Matriz completa de temas aberta/reiniciada no Desktop nativo | Todos os temas em janela real → não concluída; E2E Composer falhou | NOT_RUN | `COMPANION_VALIDATION.md`, logs E2E de temas | P2 | `bf6adf0c` |
| UI-KEYBOARD | Redimensionamento, foco, teclado, tooltips e diálogos na janela nativa | Fluxos acessíveis em desktop → execução manual incompleta | NOT_RUN | `ENVIRONMENT.md` | P2 | `bf6adf0c` |
| UI-MONITORS | Escala DPI e segundo monitor | Layout em múltiplas escalas → apenas 1920×1080, escala 1 | NOT_RUN | `ENVIRONMENT.md` | P2 | `bf6adf0c` |
| HUD-MATRIX | Barra/Mascote/Bandeja, show/hide, auto-hide, foco e persistência | Matriz completa nativa → somente início e preferências observados | NOT_RUN | `HUD_VALIDATION.md` | P2 | `bf6adf0c` |
| ROUTING-01 | Exercitar modos DIRECT/STICKY_LEAD/ORCHESTRATED e decisões KEEP/DELEGATE/MISSION | Decisões reais respeitam especialidade/risco/continuidade → não executado | NOT_RUN | `RUNTIME_VALIDATION.md` | P1 | `bf6adf0c` |
| ROUTING-02 | Troca de agente, failover, retomada e cancelamento em tarefa sintética | Continuidade real sem órfãos → não executado para não interferir em sessões ativas | NOT_RUN | `RUNTIME_VALIDATION.md` | P1 | `bf6adf0c` |
| SKILLS-01 | Descoberta/invocação explícita/automática de skills na sessão | Skills isoladas e autorizadas → não exercitado com provedor real | NOT_RUN | `RUNTIME_VALIDATION.md` | P2 | `bf6adf0c` |
| TERMINAL-01 | Terminal nativo: ANSI, Unicode, resize, múltiplas sessões, longa duração | Sessões reais e cleanup corretos → matriz nativa não executada | NOT_RUN | `RUNTIME_VALIDATION.md` | P2 | `bf6adf0c` |
| UPDATE-01 | Download interrompido, integridade, rollback e preservação de configuração | Atualização recuperável → não executado sem release/artefato de teste | NOT_RUN | `DISTRIBUTION_VALIDATION.md` | P1 | `bf6adf0c` |
| RESILIENCE-01 | Crash, configuração corrompida, porta ocupada, permissões e instâncias | Recuperação sem processo órfão → matriz de falhas não executada | NOT_RUN | `RUNTIME_VALIDATION.md` | P2 | `bf6adf0c` |
| LOGS-01 | Inspecionar logs após falhas reais de autenticação/provedor | Sem token/segredo → campanha de redaction dinâmica não executada | NOT_RUN | `RUNTIME_VALIDATION.md` | P1 | `bf6adf0c` |
| PERF-01 | Amostrar CPU/memória em repouso e animação HUD | Uso de recursos dentro do orçamento → amostra comparável não coletada | NOT_RUN | `ENVIRONMENT.md` | P2 | `bf6adf0c` |

## Contagem

| PASS | FAIL | BLOCKED | NOT_RUN | Total |
|---:|---:|---:|---:|---:|
| 30 | 7 | 6 | 12 | 55 |

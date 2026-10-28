# Homologação Linux — alterações recentes do Nexus 1.0.0 (28/10/2026)

> **Status:** roteiro executável, **não é evidência de homologação**. Branch canônica: `release/v1.0.0`. As 48 obrigações em `docs/reference/agent-workspace-v1.json` são a fonte estruturada de escopo; não inventar funcionalidades ou marcar `PASS` com base apenas na presença de código.

## PROMPT PARA O AGENTE EXECUTOR

Você é o responsável pela auditoria e homologação **real**, em Linux amd64, de todas as alterações recentemente integradas ao Nexus 1.0.0. Trabalhe no repositório `IAPro-Community/nexus`, `release/v1.0.0`, e **não use SHAs antigos como se fossem o HEAD atual**.

### Regras obrigatórias

1. Leia `AGENTS.md`, `docs/product/agent-workspace-v1-acceptance.md`, `docs/reference/agent-workspace-v1.json`, as issues #94, #100, #102, #111, #115 e a documentação do Browser. Capture `git rev-parse HEAD`, distro/kernel, `uname -m`, sessão X11/Wayland, Go, Bun, Chromium, versões e binários dos CLIs, os workflows aplicáveis e os commits após a última release instalada.
2. Preserve mudanças locais, perfis, tokens e credenciais. Homologação no clone/worktree dedicado de SHA fixo; **não limpar diretórios do usuário**. Use serviços HTTP e dados sintéticos. Não fazer push, merge ou publicar release sem aprovação.
3. Empregue **dois agentes reais simultâneos de provedores diferentes** (prioridade OpenCode e AGY; testar Codex, Pi, OMP, dsh, mimo, Claude e Gemini quando efetivamente instalados e suportados), e **dois clientes independentes**: Cliente A desktop Linux, Cliente B celular com sessão/cookies independentes ou, se não disponível, perfil isolado em outro navegador. Não considerar duas abas do mesmo perfil como dois clientes.
4. Estados de evidência: `PASS` (executado e comprovado), `FAIL` (defeito reproduzido), `BLOCKED` (ambiente impede), `NOT_TESTED` (não executado), `PARTIAL` (parte verificada), `NOT_IMPLEMENTED` (funcionalidade inexistente ou explicitamente planejada). Um teste com mock não prova integração real, e instalar um CLI não comprova suporte nativo a ACP/RPC.
5. Não misture: processo vivo, turno em execução, pergunta pendente, sessão Browser, lease humano, snapshot de arquivo e status de Mission. Os respectivos proprietários são Nexus Core/driver e Maestro conforme o contrato de `executionOwner`.
6. Nunca registre em logs/evidências senhas, cookies, tokens MCP, OTP, dados de produção ou screenshots de login reais; redija previamente redaction. Bloqueie qualquer plano de compartilhar projeto com usuários não confiáveis enquanto RBAC do Nexus não estiver aprovado.

### Fase A — verificações automáticas exact-SHA (não substituir por inspeção textual)

Em checkout limpo:

```bash
git checkout release/v1.0.0
git pull --ff-only
export NEXUS_EXPECTED_SHA="$(git rev-parse HEAD)"
git status --short
uname -a
go version
bun --version
node --version
# Contratos de documentação e arquitetura:
node --test scripts/verify-agent-workspace-v1.test.mjs
node scripts/verify-agent-workspace-v1.mjs
bun scripts/docs-verify.mjs
# NEX / geração isolada de assets (depende de bun install no web):
bun install --cwd web --frozen-lockfile
node --test web/scripts/export-nex-art.test.mjs
# Backend / multi-provider / browser / outputs:
go test -race -count=1 ./internal/nexus/artifacts ./internal/browsercontrol ./internal/control/launcher ./internal/control/web
# UI, acessibilidade e builds:
cd web
bun run format:check
bun run typecheck
bun run lint
bun run lint:styles
bun run test
bun run build
cd ..
# Suite completa do beta Linux amd64, incluindo e2e e evidências:
NEXUS_EXPECTED_SHA="$(git rev-parse HEAD)" bash scripts/homologate-beta.sh
```

O script `homologate-beta.sh` exige toolchain versionada, assinatura/chave pública canônica e um workspace inicialmente limpo; **se um pré-requisito falhar, relatar `BLOCKED`**, não improvisar alteração em configuração pessoal. Registrar comandos, exit codes e logs sanitizados em `DEV/validation/local-beta/<VERSION>/<SHA>/linux-amd64/` ou no padrão da própria suíte. O build não pode modificar os SVGs versionados de `web/public/nex`.

### Fase B — matrizes reais (dois agentes + dois clientes)

| ID | Componente / mudança | Operação e evidência mínima |
|---|---|---|
| LNX-01 | Primeiro uso / sessão host | Core + Workspace abrem na build exata; login/túnel autorizado; dois clientes conectam separadamente |
| LNX-02 | Agentes e perfis | Iniciar OpenCode + AGY supervisionados simultaneamente; provedor, profile, modelo e terminal não mudam silenciosamente |
| LNX-03 | AgentWire ACP/RPC/PTy | Em cada provider disponível, verificar negociação de transporte, processo/turno separados, pergunta com ID, conclusão real e fallback declarado |
| LNX-04 | Attention Center universal | Duas perguntas **idênticas** de runtimes diferentes devem gerar ações distintas; pergunta, permissão, erro, conclusão e cancelamento; sem falsas resoluções |
| LNX-05 | Attention cross-project | Alerta do agente A chega com usuário no projeto B; clicar abre terminal certo; detach/reconnect/crash com estado `UNKNOWN` quando a fonte não confirma |
| LNX-06 | Terminal Studio | Dois terminais, split, rename, focus, Ctrl+C, resize, scrollback, reattach sem relançar processo, input lease único |
| LNX-07 | Browser MCP automático | `browser_open`, DOM/screenshot, fill/click, lease, `request_human`, retomar após MFA **sintético**, sessão selecionável nos dois clientes |
| LNX-08 | Isolamento de Browser | Dois agentes com dois Chromiums e projetos/worktrees isolados; negar token revogado, URL/ID de outro projeto, controle cruzado e comandos atrasados |
| LNX-09 | Outputs / Files | Gerar 5 PNGs + 1 PDF sintéticos no Linux; galeria e Files no desktop/celular, preview, thumbnail, listagem paginada e download original |
| LNX-10 | Histórico de artefatos | Substituir imagem, navegar revisões, baixar snapshot anterior, comparar SHA-256, testar arquivo parcial e reconexão |
| LNX-11 | Registro MCP de output | Chamar `outputs_publish` do runtime correto; associar autoria verificável, negar `../`, symlink externo, `.env`, MIME falso e arquivo de outro projeto |
| LNX-12 | Browser -> Outputs | Testar download/export e screenshot; **se não houver integração automática, registrar NOT_IMPLEMENTED**, jamais fingir que screenshot é original |
| LNX-13 | Composer | Criação, entrevista, clarificação, histórico, preview, falhas e recuperação no backend real; conferir status de funcionalidade conforme flags |
| LNX-14 | Flow / Mission | Editor, execução/handoff, retomada, review, `executionOwner`, Evidence Bundle; não iniciar trabalho sem autorização |
| LNX-15 | Tours contextuais | Composer início/sessão/offline; Flow biblioteca/editor/Mission; abrir, avançar, voltar, Esc, retomar e concluir sem chamadas de mutação |
| LNX-16 | Tour + layout | Testar tour em split panes e elementos ocultos; highlights apenas no painel de origem, teclas e foco corretos |
| LNX-17 | Workspace UX | Terminal, Browser, Outputs, Overview, Agents, Rails, comando Ctrl+K, tabs, split, popout, restauração de layout e mudança de projeto |
| LNX-18 | Maestro | Contexto e memória canônicos, política de migração/reversão, nenhuma duplicação pelo Nexus, falha explícita em vez de fallback silencioso |
| LNX-19 | Mobile Companion | Cliente B observa agentes/arquivos/status; intervenção remota idempotente, reattach, logout, perda de rede, expiração e negação cross-project |
| LNX-20 | NEX / HUD | Estados idle/working/warning/critical/offline, animações/reduced motion, contraste e click-through na janela nativa Linux quando disponível |
| LNX-21 | Release notices | Novas funcionalidades têm aviso/tour com estado correto, sem anunciar recursos não implementados/homologados |
| LNX-22 | Plataformas / release | Verificação do HEAD e de artefatos assinados, scripts de instalação/rollback, docs exact-SHA, sem promover release enquanto faltarem gates Windows/macOS |
| LNX-23 | 48 requisitos V1 | Enumerar todos os IDs do JSON e confrontar `present/partial/missing` com execução real, evidências e issues; não converter escopo em `PASS` |
| LNX-24 | Contratos negativos | Dois clientes não podem introduzir cookies, credenciais, artefatos, Browser/MCP nem input de outros projetos; RBAC ausente é impeditivo para terceiros |

### Fase C — variantes e falhas

- **Dois clientes:** A age, B observa, B desconecta/reconecta, B assume após liberação explícita, A não age simultaneamente. Repita em celular físico e viewport 390px/tablet/desktop; se só em emulador, informe.
- **Dois agentes:** ambos trabalhando simultaneamente; um solicita permissão, outro gera imagem; responder a um não resolve o outro. Encerrar um não invalida o segundo.
- **Estados reais:** subprocesso encerra, turno cancelado, pergunta pendente, resposta atrasada/replay, credencial expirada, provider offline, rede oscilando e Core reiniciando. Nenhuma mensagem “concluído” sem evento conclusivo autorizado.
- **Compatibilidade:** ACP, RPC e PTY apenas com binários instalados e versões registradas. Para fornecedor ausente, `BLOCKED — NOT_INSTALLED`, não `PASS`. Explicitamente verificar modos que alteram experiência TUI/GUI.
- **Segurança:** traversal, symlinks, MIME disfarçado, arquivo de outro projeto, sessões não autenticadas, CSRF, XSS no preview, URLs do navegador, tokens revogados, recursos excessivamente grandes, isolamento de snapshots.
- **UX:** PT-BR/EN/ES, teclado completo, 100–200% zoom, alto contraste, modo claro/escuro, toque e reduced motion; testar a versão distribuída, não só dev server.
- **Desempenho:** monitorar memória, CPU e processos órfãos de Go/Node/Chromium, separando medições reais de estimativas.

### Entrega obrigatória

Produza `DEV/validation/linux-full-v1/<SHA>/REPORT.md`, `matrix.csv`, `environment.txt`, `issues.md`, evidências redigidas e logs com exit code. Registre **SHA exato, data, agente, provedor, versão, cliente/dispositivo e resultado para cada caso**.

O relatório final deve ter: resumo GO/NO-GO para **Linux funcional**, **mobile/duas sessões**, **multiagent**, **release geral**; links às evidências; defeitos com passos de reprodução; correções de baixo risco feitas localmente e testes de regressão; tarefas bloqueadas por dispositivo, autenticação, Windows/macOS ou aprovação humana.

**Não encerrar issues e não declarar release homologada sem dois agentes reais, dois clientes independentes e evidência exact-SHA. Não fazer push sem permissão do usuário.**

## Rastreabilidade

- `scripts/homologate-beta.sh`: suite Linux principal, inclusive race/Browser/terminal.
- `docs/reference/agent-workspace-v1.json`: 48 obrigações de escopo; `scripts/verify-agent-workspace-v1.mjs`: verificador.
- `docs/product/nexus-outputs-project-files.md`: Outputs/Files, limites de autorização e histórico.
- `docs/product/nexus-browser-auto-mcp.md`: Browser MCP auto provisionado.
- `docs/design/universal-agent-harness-support.md`: transportes ACP/RPC/PTY e capacidades.
- `docs/product/agent-workspace-v1-acceptance.md`: critérios completos V1.
- Issues #94, #100, #102, #111, #115: backlog e gates ainda abertos.

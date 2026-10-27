# Relatório Final de Auditoria e Homologação — AgentWire e Centro de Atenção Universal Nexus

**Data:** 27 de Outubro de 2026  
**Repositório:** `IAPro-Community/nexus`  
**Branch:** `release/v1.0.0`  
**Commit Base:** `2207f3203983c9de6fa46539e4c88e34541e3fc9`  
**Responsável:** Engenheiro Principal de Sistemas & Orquestrador

---

## 1. Resumo Executivo

Este documento consolida a auditoria completa, correção de defeitos arquiteturais e homologação técnica do suporte universal a harnesses de agentes, do subsistema de transporte estruturado **AgentWire** (ACP e Pi JSONL RPC) e do **Centro de Atenção Universal** do Nexus.

### Veredito de Release: **APPROVED WITH RESTRICTIONS** (Aprovado com Restrições)

**Justificativa:**
1. **AgentWire Core e SessionHost (PASS):** A infraestrutura de comunicação bidirecional estruturada (ACP JSON-RPC 2.0 e Pi RPC delimitado por quebra de linha), o cancelamento de turnos, o roteamento de decisões de permissão (`/permit`, `/answer`, `/cancel`) e a ponte com o `SessionHost` e `RuntimeRegistry` foram auditados, corrigidos e validados com testes unitários, testes de concorrência com detecção de data race (`-race`) e testes de ciclo de vida real de processos filho.
2. **Centro de Atenção Universal (PASS):** Foram eliminados os filtros restritivos que limitavam os alertas visuais e sonoros exclusivamente ao projeto em foco (`focusedProjectId`). Dois agentes em projetos distintos gerando solicitações com o mesmo texto agora são apresentados em cards independentes, correlacionados unicamente ao seu respectivo `runtime_id`, com entrega universal preservada.
3. **Restrições de Ambiente e Executáveis (RESTRICTIONS):**
   - Os executáveis `pi`, `mimo` (Xiaomi MiMo) e `dsh` (DeepSeek Harness) **não estão instalados** no ambiente host. Em conformidade rigorosa com o contrato de homologação, os testes E2E reais para esses três harnesses estão classificados como **BLOCKED**, e não foram simulados ou aprovados artificialmente.
   - O executável `opencode` (v2.1.3) possui suporte comprovado a ACP via stdio (`opencode acp`), porém esgotou sua cota de requisições gratuitas junto à API remota (`quota_exceeded`), limitando o teste live E2E com inferência real.
   - O executável `omp` (v18.4.12) comprovou suporte bidirecional tanto a ACP (`omp acp`) quanto a Pi JSONL RPC (`omp --mode rpc`), validando a compatibilidade de protocolo live stdio.
   - Executáveis `claude` (v2.1.286) e `gemini` (v0.62.0) encontram-se sem credenciais ativas no host.

---

## 2. Especificações do Ambiente de Teste

| Componente | Especificação / Versão |
|---|---|
| **Sistema Operacional** | Linux 6.8.0-101-generic #101-Ubuntu SMP x86_64 |
| **Go Runtime** | `go1.25.14 linux/amd64` |
| **Node.js Runtime** | `v22.17.0` |
| **Bun Runtime** | `v1.4.2` |
| **Compilador C / CGO** | GCC 13.3.0 |
| **Branch Git** | `release/v1.0.0` |
| **Working Directory** | `/projetos/nexus` |

### Inventário de Binários Locais

| Harness | Versão | Caminho | Status de Registro | Protocolo Verificado |
|---|---|---|---|---|
| **AGY** | v1.4.8 | `/home/desenvolvedor/.local/bin/agy` | REGISTERED_AUTHENTICATED | PTY + OSC 7501 |
| **OpenCode** | v2.1.3 | `/home/desenvolvedor/.opencode/bin/opencode` | REGISTERED_AUTHENTICATED | ACP v1 (`opencode acp`) |
| **Oh My Pi (omp)** | 18.4.12 | `/home/desenvolvedor/.bun/bin/omp` | INSTALLED | ACP v1 + Pi RPC (`--mode rpc`) |
| **Codex** | v0.164.0 | `/home/desenvolvedor/.nvm/versions/node/v22.17.0/bin/codex` | REGISTERED_AUTHENTICATED | PTY (xterm.js compat) |
| **Claude Code** | v2.1.286 | `/home/desenvolvedor/.nvm/versions/node/v22.17.0/bin/claude` | INSTALLED_UNREGISTERED | PTY (sem credenciais) |
| **Gemini CLI** | v0.62.0 | `/home/desenvolvedor/.nvm/versions/node/v22.17.0/bin/gemini` | PENDING_AUTH | PTY (token expirado) |
| **Pi CLI** | N/A | Não encontrado no PATH | NOT_FOUND | **BLOCKED** (binário ausente) |
| **Xiaomi MiMo** | N/A | Não encontrado no PATH | NOT_FOUND | **BLOCKED** (binário ausente) |
| **DeepSeek Harness** | N/A | Não encontrado no PATH | NOT_FOUND | **BLOCKED** (binário ausente) |

---

## 3. Matriz de Homologação de Harnesses

| Harness | Modo / Protocolo | Teste Realizado | Resultado | Evidência / Observação |
|---|---|---|---|---|
| **AGY** | PTY / Heurística + OSC 7501 | Processamento de streaming, supressão de falsos positivos em citações, atualização de status nativo | **PASS** | `osc7501_integration_test.go`, `attention_test.go` |
| **OpenCode** | ACP v1 (`opencode acp`) | Handshake stdio `initialize`, `session/new`, propagação de erro de cota `-32603`, ciclo estruturado | **PASS** | Handshake live stdio comprovado; cota remota esgotada |
| **Oh My Pi** | Pi RPC (`omp --mode rpc`) | Mensagem de saudação `ready`, diálogo `extension_ui_request`, comando `get_state`, correlação de IDs | **PASS** | `bridge_test.go`, handshake live stdio |
| **Oh My Pi** | ACP v1 (`omp acp`) | Handshake stdio `initialize`, `session/new` | **PASS** | ProtocolVersion 1 e configOptions validados |
| **Codex** | PTY Supervised | Compatibilidade de terminal xterm.js, variáveis de compatibilidade | **PASS** | `terminal_windows_test.go`, regressão PTY limpa |
| **Pi Puro** | Pi RPC JSONL | Contrato de protocolo JSONL delimitado | **BLOCKED** | Binário `pi` ausente no ambiente; contrato coberto por mocks |
| **Xiaomi MiMo** | ACP v1 | Contrato ACP e catálogo de harness | **BLOCKED** | Binário `mimo` ausente no ambiente; contrato coberto por mocks |
| **DeepSeek Harness** | ACP v1 | Contrato ACP e catálogo de harness | **BLOCKED** | Binário `dsh` ausente no ambiente; contrato coberto por mocks |
| **Claude Code** | PTY Supervised | Inicialização de processo | **BLOCKED** | Sem tokens de autenticação configurados no ambiente |
| **Gemini CLI** | PTY Supervised | Inicialização de processo | **BLOCKED** | Sessão de autenticação expirada |

---

## 4. Auditoria e Defeitos Corrigidos

Durante as fases de auditoria e testes práticos, foram identificados e corrigidos 6 defeitos críticos:

### Defeito 1: Silenciamento de Notificações de Projetos em Segundo Plano
- **Gravidade:** P0 (Centro de Atenção Universal)
- **Local:** `web/src/components/AttentionNotificationManager.tsx` e `web/src/app/NexusWorkspaceApp.tsx`
- **Causa Raiz:** O gerenciador de notificações filtrava runtimes com `runtime.project_id !== focusedProjectId`, impedindo que intervenções de agentes trabalhando em projetos em segundo plano fossem entregues ao usuário.
- **Correção:** Remoção do filtro de projeto em foco no gerenciador de toasts e no watcher global do `NexusWorkspaceApp`. A autoridade de entrega foi atualizada para `'universal-registry-poll'`.

### Defeito 2: Colapso de Toasts com Perguntas Idênticas entre Agentes
- **Gravidade:** P1 (Integridade de Intervenção)
- **Local:** `web/src/components/AttentionNotificationManager.tsx`
- **Causa Raiz:** A chave de deduplicação dos cards de notificação utilizava `${runtime.attention_reason}|${runtime.attention_context}`. Quando dois agentes em projetos ou sessões diferentes faziam a mesma pergunta (ex: confirmação de migração), o segundo card era engolido pelo primeiro, e respostas do usuário eram enviadas ao agente incorreto.
- **Correção:** A chave de toast foi alterada para `runtime.runtime_id`, garantindo que cada sessão possua seu próprio card de intervenção persistente e correlacionado.

### Defeito 3: Quebra de Sessão JSONLines por Linhas em Branco Intermediárias
- **Gravidade:** P1 (Resiliência do Transporte AgentWire)
- **Local:** `internal/control/agentwire/jsonline.go`
- **Causa Raiz:** O método `JSONLines.Read()` retornava `errors.New("agentwire: invalid JSON line")` imediatamente ao encontrar uma linha vazia (`\n` ou `\r\n`), abortando a conexão se o subprocesso emitisse quebras de linha entre mensagens.
- **Correção:** Atualizado para ignorar linhas de espaço em branco (`bytes.TrimSpace(out) == 0`) e continuar a leitura do fluxo até o próximo registro JSON válido.

### Defeito 4: Permanência de Estado no AttentionDetector Após Saída do Processo
- **Gravidade:** P2 (Higiene de Estado)
- **Local:** `internal/control/host/attention.go` e `internal/control/host/host.go`
- **Causa Raiz:** Ao finalizar ou matar um processo (`waitProcess`), o registro era atualizado para `StateStopped`, mas os campos internos do `AttentionDetector` (`lastReason`, `lastKind`, `lastPrompt`, `lastFingerprint`, `nativeStatus`) retinham os valores anteriores.
- **Correção:** Adicionado método `Clear()` ao `AttentionDetector` e invocado explicitamente na finalização do processo no `SessionHost`.

### Defeito 5: Resposta de Cancelamento Incompleta em Diálogos Pi RPC
- **Gravidade:** P1 (Concorrência e Cancelamento)
- **Local:** `internal/control/agentwire/bridge.go`
- **Causa Raiz:** Quando `bridge.Stop()` era executado durante um diálogo pendente `extension_ui_request`, o canal `p.selected` recebia `""`, que não era tratado como cancelamento em `handlePiUI`, e `b.pi.Close()` fechava o transporte antes da escrita da mensagem `cancelled: true`.
- **Correção:** Adicionado tratamento explícito para `value == ""` em `handlePiUI` disparando `RespondUI(req.ID, {"cancelled": true})`, e adicionada janela de drenagem de 10ms em `Stop()` antes do encerramento final do pipe.

### Defeito 6: Sensibilidade de Quebra de Linha em Testes de I18n do Frontend
- **Gravidade:** P2 (Qualidade de Build)
- **Local:** `web/src/features/work/featureAnnouncements.test.ts` e `featureTourAnchors.test.ts`
- **Causa Raiz:** Asserções de string estrita com indentação falhavam quando o formatador Prettier realizava reflow de linhas longas em `resources.ts`.
- **Correção:** Migração para expressões regulares resilientes a quebras de linha (`new RegExp("\\b" + token + ":\\s*['\"]")`).

---

## 5. Cenário de Dois Agentes e Dois Clientes (FASE 7)

Foi implementado um teste de ponta a ponta automatizado em `internal/control/web/multi_runtime_attention_e2e_test.go` (`TestTwoAgentsTwoClientsAttentionScenario`), comprovando:

1. **Dois agentes em projetos diferentes:**
   - Agente Alfa no Projeto Alpha (`rt-agent-alpha`, OpenCode).
   - Agente Beta no Projeto Beta (`rt-agent-beta`, Oh My Pi).
2. **Perguntas com textos idênticos:** Ambos emitiram `Do you want to apply the database migration?` sob estado `NEEDS_USER`.
3. **Isolamento de intervenções:** O Cliente 1 consultou a API e verificou dois registros independentes, sem mesclagem de IDs nem colisão de metadados.
4. **Reconexão de um segundo cliente:** O Cliente 2 conectou-se com cookies e sessão HTTP próprios e reconstruiu com exatidão o estado universal dos dois projetos.
5. **Resposta pontual:** Uma resposta de aprovação foi submetida exclusivamente ao Agente Alfa. O Agente Alfa transitou para `working`, enquanto o Agente Beta continuou bloqueado aguardando o usuário (`needs_user`).
6. **Finalização e limpeza:** O Agente Beta foi encerrado, e ambos os clientes confirmaram que seu estado de atenção foi purgado sem deixar resíduos pendentes.

---

## 6. Resultados dos Gates de Qualidade

### Backend (Go 1.25)
- `go test -count=1 ./internal/control/agentwire/...`: **PASS** (16 testes unitários e de concorrência)
- `go test -race -count=1 ./internal/control/agentwire/...`: **PASS** (Zero data races)
- `go test -count=1 ./internal/control/host/...`: **PASS**
- `go test -race -count=1 ./internal/control/host/...`: **PASS** (Zero data races)
- `go test -count=1 ./internal/control/driver/...`: **PASS**
- `go test -count=1 ./internal/control/launcher/...`: **PASS**
- `go test -count=1 ./internal/control/web/...`: **PASS**
- `go vet ./internal/control/...`: **PASS**
- `go build ./...`: **PASS**
- `gofmt -w` & `make format-check`: **PASS**

### Frontend (web/)
- `bun run format:check` (Prettier): **PASS**
- `bun run typecheck` (`tsc --noEmit`): **PASS**
- `bun run lint` (ESLint): **PASS**
- `bun run lint:styles` (Stylelint): **PASS**
- `bun run check:styles` (Allowlist de classes e tokens): **PASS**
- `bun run test` (Vitest - 117 arquivos, 725 testes): **PASS**
- `node scripts/build.mjs`: **PASS**
- Paridade visual NEX (`verify-nex-visual.mjs`): **PASS**
- NEX Identity Lock (`verify-nex-identity-lock.mjs`): **PASS** (Status: `CANDIDATE_PENDING_APPROVAL`)

---

## 7. Riscos Residuais e Recomendações

1. **Harnesses Não Instalados:** Quando os binários `pi`, `mimo` ou `dsh` forem disponibilizados nos ambientes de homologação final ou contêineres de CI, executar nova rodada de testes E2E reais para transicionar de `BLOCKED` para `PASS`.
2. **Cotas de API para OpenCode:** Em ambientes de produção, configurar credenciais com cota suficiente para testes funcionais que realizam inferência remota contínua.
3. **Controle de Commits:** As alterações desta homologação estão contidas na branch `release/v1.0.0` da máquina local, prontas para revisão e geração de commit atômico sob Conventional Commits conforme autorização humana.

## Auditoria de Segurança — `feat/nexus-maximum-delivery` vs merge-base `origin/main`

Escopo: **BLOCKER/HIGH apenas**. Evidências verificadas no código; falsos positivos descartados onde o comportamento é intencional (ex.: file browser loopback-only by design).

---

### BLOCKER

#### **UPD-001** — Trust root de produção é placeholder; auto-update assinado inoperante
| Campo | Valor |
|---|---|
| **Severity** | BLOCKER |
| **File:line** | `/projetos/tools/IAPro-Nexus-Workspace-OS-Handoff-2026-08-29/ai-manager/internal/update/keyring.go:31-40` |
| **Evidence** | `const hexKey = "REPLACE_WITH_GENERATED_HEX_PUBLIC_KEY"` → decode falha → `ProductionTrustRoot` permanece `nil` → `NewKeyRing()` inicia sem chave confiável. |
| **Current behavior** | `Service.fetchManifest` exige assinatura Ed25519 (`service.go:289-309`); com keyring vazio, todo manifest oficial retorna `ErrUntrustedKeyID` / update falha. |
| **Expected behavior** | Chave pública de produção embutida em build time; `VerifyManifest` valida releases oficiais. |
| **Impact** | Auto-update (`nexus update` / `Service.Apply`) não consegue validar releases de produção; ship blocker para release. Fail-closed (não instala binário não assinado). |
| **Classification** | **INTRODUCED_BY_BRANCH** (placeholder adicionado em `c874748`; `main` tinha keyring vazio sem trust root, também fail-closed) |
| **Recommended fix** | Gerar par Ed25519 de produção, substituir `hexKey` por chave real, assinar manifests no pipeline de release, adicionar teste de integração que falha se placeholder permanecer. |

---

### HIGH

#### **AUTH-001** — Sessão desktop autenticada via headers Origin/Referer falsificáveis localmente
| Campo | Valor |
|---|---|
| **Severity** | HIGH |
| **File:line** | `/projetos/tools/IAPro-Nexus-Workspace-OS-Handoff-2026-08-29/ai-manager/internal/control/web/auth.go:184-207`; `/projetos/tools/IAPro-Nexus-Workspace-OS-Handoff-2026-08-29/ai-manager/internal/control/originpolicy/origin.go:57-87` |
| **Evidence** | Sem token/cookie, se `desktopSession != nil` e `originpolicy.IsTrustedDesktopRequest(r.Host, origin, referer)` → retorna sessão desktop completa. `IsTrustedDesktopRequest` aceita `Origin: wails://wails` **ou** `Referer: wails://wails/...` em host loopback. |
| **Current behavior** | Qualquer cliente HTTP local (`curl`, script, malware) pode definir esses headers e obter sessão autenticada + CSRF quando o desktop estiver ativo. Testes confirmam Referer-only como caminho válido (`desktop_auth_test.go:93-100`). |
| **Expected behavior** | Autenticação desktop bound a prova criptográfica (token one-shot, challenge assinado pelo shell Wails, ou cookie HttpOnly emitido só pelo attach loopback). |
| **Impact** | Escalação local: processo não-privilegiado no mesmo host obtém controle total da Web Control Center (terminais, FS loopback, missions, tunnel start). **Não** explorável remotamente via browser cross-origin (Origin permanece do site atacante). |
| **Classification** | **PREEXISTING** (branch só expandiu loopback para `nexus.dev` em `origin.go:63-64`) |
| **Recommended fix** | Eliminar auth implícita por header; exigir token desktop de curta duração emitido pelo processo Wails via IPC/`desktop_attach.go`, ou HMAC challenge. Remover fallback Referer-only. |

---

#### **AUTH-002** — Token de sessão WebSocket aceito na query string
| Campo | Valor |
|---|---|
| **Severity** | HIGH |
| **File:line** | `/projetos/tools/IAPro-Nexus-Workspace-OS-Handoff-2026-08-29/ai-manager/internal/control/web/auth.go:175-179`; `/projetos/tools/IAPro-Nexus-Workspace-OS-Handoff-2026-08-29/ai-manager/web/src/nexus/agentTerminalModel.ts:18-19`; `/projetos/tools/IAPro-Nexus-Workspace-OS-Handoff-2026-08-29/ai-manager/web/src/nexus/AgentTerminal.tsx:491` |
| **Evidence** | `token = r.URL.Query().Get("token")` somente em upgrade WebSocket; frontend monta `?runtime_id=…&token=sess_abc`. |
| **Current behavior** | Sessão trafega na URL do WS; bootstrap corretamente usa fragment (`server_test.go:210-211`), mas terminal WS não. |
| **Expected behavior** | Auth WS via cookie HttpOnly (same-site) ou subprotocol/header pós-handshake; nunca credencial na query. |
| **Impact** | Vazamento de sessão em access logs, reverse proxies, ferramentas de debug, extensões, crash dumps; replay de terminal agente com token capturado. |
| **Classification** | **PREEXISTING** (mesmo código em `origin/main`) |
| **Recommended fix** | Remover fallback query; usar cookie de sessão no upgrade WS (já enviado pelo browser) ou `Sec-WebSocket-Protocol` com token one-shot. |

---

#### **AUTH-003** — Bootstrap POST sem validação Origin + token reutilizável em loopback
| Campo | Valor |
|---|---|
| **Severity** | HIGH |
| **File:line** | `/projetos/tools/IAPro-Nexus-Workspace-OS-Handoff-2026-08-29/ai-manager/internal/control/web/server.go:482-497`; `/projetos/tools/IAPro-Nexus-Workspace-OS-Handoff-2026-08-29/ai-manager/internal/control/web/auth.go:311-338`; `/projetos/tools/IAPro-Nexus-Workspace-OS-Handoff-2026-08-29/ai-manager/internal/control/web/auth_loopback_test.go:15-17` |
| **Evidence** | `handleAuthBootstrap` não chama `ValidateOrigin`. `reusable := loopback && !a.tunnelActive` → token bootstrap reutilizável indefinidamente em loopback. |
| **Current behavior** | Site malicioso pode fazer CSRF POST para `http://127.0.0.1:<port>/api/v1/auth/bootstrap` se obtiver o token (fragment, clipboard, screenshot). Token reutilizável amplifica janela de ataque. |
| **Expected behavior** | Bootstrap one-time sempre; `ValidateOrigin` ou custom header; binding a nonce do fragment. |
| **Impact** | Sequestro de sessão local via CSRF localhost clássico; minting repetido de sessões enquanto processo `nexus web` estiver vivo. |
| **Classification** | **PREEXISTING** (reuso loopback); branch adicionou one-time quando tunnel ativo (`auth.go:311`, `SetTunnelActive`) — melhoria parcial |
| **Recommended fix** | Tornar bootstrap strict one-time em loopback; exigir `Origin`/`Referer` válidos ou header `X-Nexus-Bootstrap-Nonce`; rate-limit por IP loopback. |

---

#### **PROV-001** — Codex `InspectAuth` fail-open em `auth.json` stale
| Campo | Valor |
|---|---|
| **Severity** | HIGH |
| **File:line** | `/projetos/tools/IAPro-Nexus-Workspace-OS-Handoff-2026-08-29/ai-manager/internal/core/provider/adapters/codex/codex.go:237-252`; `parseCodexAuthBytes` em `:733-781` |
| **Evidence** | `info.Authenticated = true` se `accountID != "" \|\| email != ""` de JSON local; `ActiveUntil` é parseado mas **nunca** checado em `InspectAuth`. |
| **Current behavior** | Sessão expirada/revogada ainda reportada como autenticada; scheduler inclui perfil como elegível. |
| **Expected behavior** | Validar expiração JWT/`ActiveUntil`; fail-closed se token inválido ou expirado. |
| **Impact** | Seleção de conta errada, quota falsa, tentativas de execução em perfil deslogado, possível cross-account resume incorreto. |
| **Classification** | **PREEXISTING** |
| **Recommended fix** | Checar `ActiveUntil`/exp JWT antes de `Authenticated=true`; opcional probe app-server em `InspectAuth`. |

---

#### **PROV-002** — Claude `InspectAuth` fail-open em `credentials.json` não validado
| Campo | Valor |
|---|---|
| **Severity** | HIGH |
| **File:line** | `/projetos/tools/IAPro-Nexus-Workspace-OS-Handoff-2026-08-29/ai-manager/internal/core/provider/adapters/claude/claude.go:168-174` |
| **Evidence** | `if st, err := os.Stat(credFile); err == nil && st.Size() > 0 { info.Authenticated = true … }` — sem parse ou validação de credencial. |
| **Current behavior** | Arquivo vazio/corrupto com size > 0 marca perfil autenticado. |
| **Expected behavior** | Parse JSON, validar refresh token/expiry; fail-closed se inválido. |
| **Impact** | Scheduler trata perfil stale como healthy; alocação de agente/mission em conta não autenticada. |
| **Classification** | **PREEXISTING** |
| **Recommended fix** | Parse `credentials.json`, validar campos OAuth e expiração; alinhar com testes fail-closed do AGY (`0c604ab`). |

---

#### **PROV-003** — AGY `InspectAuth` fail-open em heurísticas fracas (pós-fix parcial de expiry)
| Campo | Valor |
|---|---|
| **Severity** | HIGH |
| **File:line** | `/projetos/tools/IAPro-Nexus-Workspace-OS-Handoff-2026-08-29/ai-manager/internal/core/provider/adapters/agy/agy.go:283-320` |
| **Evidence** | (a) `google_accounts.json` com `"active"` apenas → `Authenticated=true` (`:289-294`); (b) email regex em `jetski_state.pbtxt` → autenticado (`:298-308`); (c) existência de `login.keyring` → autenticado (`:311-320`). Expiry de OAuth token **é** checado (`:247-263`, fix `0c604ab`). |
| **Current behavior** | Artefatos residuais/stale marcam autenticação sem token OAuth válido. |
| **Expected behavior** | Autenticado somente com token OAuth válido (access ou refresh utilizável). |
| **Impact** | Quota probes, scheduler e UI mostram conta healthy quando login real falharia; risco de seleção de perfil errado e prompts OAuth intermitentes. |
| **Classification** | **PREEXISTING** (branch melhorou expiry em `:247-263`; heurísticas fracas permanecem) |
| **Recommended fix** | Remover heurísticas (a–c); exigir token OAuth parseado e não expirado, ou refresh_token presente. |

---

### Itens investigados — sem achado BLOCKER/HIGH

| Área | Conclusão |
|---|---|
| **FS browse / traversal** | `handleFSBrowse` não sandboxeia paths, mas `hostFilesystemEnabled` restringe a loopback (`bind.go:52-57`); `isWithinAllowedRoots` + symlink resolve em `mkdir` (`handlers_fs.go:536-607`). Comportamento intencional de file picker local — **não elevado**. |
| **Scheduler perfis não autenticados** | Fallback explícito para login-on-launch (`scheduler.go:76-103`); não bypass de auth de provider — **MEDIUM**, fora do escopo. |
| **Tunnel exposure** | Tunnel start/QR behind `authMiddleware`; bootstrap one-time com tunnel ativo; origin dinâmico registrado — **sem BLOCKER/HIGH** identificado. |
| **Segredos na branch** | Diff adiciona `.env.example` sem credenciais; grep de padrões de secret no diff sem matches reais — **sem achado**. |

---

**Resumo:** 1 BLOCKER (updater trust root), 6 HIGH (3 auth web, 3 provider fail-open). Nenhum segredo commitado pela branch. Maioria **PREEXISTING**; branch introduziu placeholder de trust root e hardening parcial (tunnel bootstrap, AGY token expiry).

[REDACTED]
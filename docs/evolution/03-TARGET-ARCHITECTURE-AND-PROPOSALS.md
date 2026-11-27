# 03 — Arquitetura Alvo e Propostas de Evolução (Contas, Capacidade, Paralelismo, Automação e Superfícies)

> **Escopo:** Propostas técnicas concretas mantendo a divisão arquitetural entre **Nexus** (cockpit diário, workspaces, isolamento de perfis/quotas, terminais, notificações), **IAPro-SDK** (contratos puros compartilhados) e **Orquestrador Maestro** (runtime autônomo externo).

---

## 1. Gerenciamento Inteligente de Múltiplas Contas e Pools Conformes com ToS

### Princípio Inviolável de Conformidade (`2026-11-27`)
- **Proibido:** relay/proxy HTTP de assinaturas consumidoras (Anthropic Pro/Max, OpenAI Plus/Pro), extração indevida de tokens de sessão do navegador ou compartilhamento não autorizado de credenciais.
- **Permitido e Oficial:** execução via **CLI oficial instalado na máquina** com diretórios isolados (`CLAUDE_CONFIG_DIR`, `CODEX_HOME`, `HOME` dedicado por perfil), setup tokens oficiais, chaves de API (`x-api-key` / `OPENAI_API_KEY`) e contas Service/Enterprise autorizadas.

### Evolução do Gerenciador de Contas & Pools
1. **Unicidade estrita por identidade canônica** (já iniciada em `53956e84` e `ed891a0d`):
   - `provider + canonical_account_id` (ou `email` normalizado) aceita exatamente **um perfil ativo**.
   - Ao importar/autenticar conta repetida, o Nexus atualiza o perfil existente em vez de criar cópia.
2. **Account Pools & Prioridade Explicável (sem magia oculta):**
   - Permitir agrupar contas legítimas num **Pool por Projeto ou Papel** (ex.: `pool:backend-codex`, ordem: `primary -> aux-api-key`).
   - Políticas explícitas e configuráveis por pool:
     - `CONSUME_FIRST` (inspirado no CCSwap: esgota janela 5h prestes a resetar antes de tocar reserva),
     - `PRESERVE_PRIMARY` (mantém assinatura principal livre para sessões interativas e descarrega background em chaves secundárias),
     - `COST_CAP` (teto diário/mensal antes de trocar ou pausar).
3. **Proveniência e Auditoria Visual no Card da Conta:**
   - Mostrar badge claro da origem: `Host-Imported` vs `Isolated-OAuth` vs `API-Key`, data de expiração estimada, e estado de lock TUI (`IsTUILocked`).
4. **Contrato Compartilhado no `IAPro-SDK`:**
   - Mover estruturas puras de `AccountScope`, `EligibilityReason`, `PoolPolicy` e regras de dedup por identidade para `IAPro-SDK/providers` e `IAPro-SDK/quota`.

---

## 2. Capacity Monitor Unificado (Visão Realista sem Spam)

Evolução direta sobre `internal/nexus/capacitymonitor` + `QuotaDropMonitor`:

1. **Snapshot Canônico por Conta + Janela:**
   - Campos normalizados (via `IAPro-SDK/capacity` + `quota`): `provider`, `profile`, `account_scope`, `window_kind` (`5h`, `7d`, `daily`, `rpm`), `remaining_pct`, `reset_at`, `fetched_at`, `trust_level` (`LIVE` | `CACHED` | `RATE_LIMITED` | `DEFERRED_TUI` | `UNSUPPORTED`).
2. **Cálculo Local de Taxa de Depleção e Risco de Exaustão:**
   - Ring buffer em memória (últimas N leituras por janela) calculando `burn_rate_pct_per_hour` e `estimated_exhaustion_at`.
   - Compara `estimated_exhaustion_at` com `reset_at` -> emite estado `SAFE`, `WATCH`, `AT_RISK` ou `BLOCKED` sem precisar consultar API externa.
3. **Coleta Adaptativa (Shipped P0-2 + Próximo Passo P1):**
   - **Já entregue em P0-2:** skip limpo de provedores `QuotaUnsupported` e reuso de snapshot fresco (`< quota.DefaultTTL = 5m`) para contas ociosas.
   - **Próximo passo (P1):** intervalo adaptativo por perfil: `30s` quando há runtime ativo nessa conta, `5m` quando ocioso e com quota `< 40%`, `15m` quando ocioso e saudável (`>= 40%`).
4. **Handoff Recomendado, Nunca Silencioso:**
   - Quando uma conta atinge exaustão durante trabalho ativo, o Monitor recomenda a melhor conta alternativa elegível no pool com comparativo (`quota restante`, `reset`, `provenance`), preservando consentimento/política do usuário.

---

## 3. Execução Simultânea de Múltiplos Agentes + Git Worktrees + Revisão Inline

Inspirado no que há de melhor em **Orca**, **Superset**, **T3 Code** e **Herdr**, adaptado ao núcleo Go/PTY do Nexus:

1. **Fluxo "1-Passo -> Agente em Worktree Isolado":**
   - Atalho global na Command Palette (`Cmd/Ctrl+K -> Novo Agente em Worktree`) e botão no header do projeto.
   - Cria branch + worktree sob `.nexus/worktrees/<slug>` (ou `~/.local/share/ai-manager/worktrees/...`), seleciona automaticamente conta saudável do pool e executa hook opcional `.nexus/setup.sh` (padrão análogo ao `.superset/setup.sh`).
   - Impede acidentalmente rodar dois agentes de escrita simultâneos na árvore canônica (reforçando a guarda A10 já existente no `handleProjectGitCheckout`).
2. **Fleet Dashboard Focado (sem sobrecarga de terminais montados):**
   - Manter no DOM apenas o terminal ativo + até 2 visíveis em split (lazy mount + skip-render estilo Herdr para abas ocultas).
   - Para agentes em background, exibir card compacto com estado real (`WORKING`, `WAITING_INPUT`, `BLOCKED_QUOTA`, `DONE`, `ERROR`), branch/worktree, arquivos modificados (`git diff --stat` periódico sob demanda) e link rápido para focar PTY.
3. **Loop Curto de Revisão de Diff (sem clonar VS Code):**
   - Painel leve de Diff por worktree/agente (diff unificado + árvore de arquivos alterados).
   - Permitir ao desenvolvedor selecionar linhas ou arquivo no diff e clicar **"Enviar comentário ao agente"**, injetando bloco estruturado no PTY/Composer do agente responsável (padrão vencedor de Orca & Vibe Kanban).
   - Botão **"Abrir na IDE"** continua invocando `openProjectInOS(..., 'editor')` para edições manuais pesadas.
4. **Checklist de Prevenção de Orfãos:**
   - Todo `__control-host` e worktree efêmera carrega PID/sessão pai e limpeza assistida no encerramento ou comando `nexus doctor --prune-worktrees`.

---

## 4. Automação Diária Determinística (`Event -> Condition -> Action -> Result`)

O usuário precisa automatizar rotinas recorrentes **sem depender de LLM caro/imprevisível para decidir o básico**:

### Modelo Declarativo
```text
TRIGGER (Event)
  -> CONDITION (Guard determinística, avaliada em Go sem LLM)
  -> ACTION (Ação segura com allowlist e idempotência)
  -> RESULT (Registro auditável no SQLite + notificação via NEX)
```

### Exemplos Práticos de Alto Valor no Dia a Dia
1. **Agente terminou tarefa em worktree (`AGENT_TURN_SETTLED`)**
   - *Condição:* worktree contém `package.json` ou `go.mod` e usuário habilitou auto-verify no projeto.
   - *Ação:* rodar comando determinístico configurado (`npm test` / `go test ./...`) com timeout e anexar saída ao resumo da sessão / Outputs.
   - *Resultado:* se falhar, notificar via NEX com opção 1-clique "Enviar falha ao agente"; se passar, marcar card pronto para review.
2. **Reset de Quota Detectado (`QUOTA_WINDOW_RESET`)**
   - *Condição:* existem tarefas/runs pausados aguardando recurso nesse provedor (`WHEN_RESOURCES`).
   - *Ação:* reavaliar elegibilidade; se `autonomy_mode == supervised`, pedir confirmação via NEX; se `autonomy_mode == auto_approved_resume`, retomar com teto de orçamento.
3. **Início de Dia / Abertura de Projeto (`WORKSPACE_OPENED`)**
   - *Condição:* projeto tem worktrees prunáveis ou sessão anterior interrompida.
   - *Ação:* exibir briefing leve no Overview ("2 sessões recuperáveis, 1 PR/branch pendente, 0 colisões") sem auto-disparar agentes pagos sem consentimento.
4. **Guardas Obrigatórias de Segurança & Custo:**
   - Dry-run testável na UI antes de ativar regra;
   - Limite de execuções por janela (anti-loop);
   - Nenhuma automação envia comandos destrutivos (`git push --force`, `rm -rf`) ou executa gasto fora de orçamento sem aprovação explícita.

---

## 5. Experiência Diária, Atenção Unificada e Evolução do NEX HUD

1. **Consolidação Total via NEX (concluída na arquitetura, a evoluir em P1):**
   - Todas as notificações operacionais (quota baixa, conta expirada, conflito de arquivo, agente bloqueado esperando decisão, falha pós-tarefa) passam pelo Hub NEX com deduplicação por episódio e fallback para notificação nativa caso o HUD não confirme leitura.
2. **Start-of-Day Continuity View (P1):**
   - Ao abrir o Nexus pela manhã, o primeiro painel útil responde em < 1s:
     1. *O que precisa da sua decisão agora?* (Attention Queue unificada — perguntas/aprovações bloqueando agentes);
     2. *Onde você parou ontem?* (últimos workspaces + branches + sessões recuperáveis);
     3. *Como está sua capacidade hoje?* (resumo compacto por provedor/conta sem ruído visual).
3. **Evolução Segura do NEX HUD (`RELEASE_READINESS.json` respeitado):**
   - Enquanto os 11 gates humanos de `NEX-ID-v1` estiverem `BLOCKED`, manter homepage pública e asset masters congelados conforme contrato.
   - No desktop Linux/macOS/Windows, **desacoplar o indicador persistente de 1.000 px WebKitGTK**: reduzir HUD ocioso para tray/indicador nativo leve (`< 25 MB`), abrindo a janela WebView rica do NEX apenas ao chamar/expandir ou receber alerta ativo.
4. **Progressive Disclosure nas Telas:**
   - Navegação principal do dia a dia focada em: **Overview (Briefing)**, **Terminals / Fleet**, **Composer / Flow** (quando habilitado), **Resources (Contas/Quotas)** e **Outputs**.
   - Superfícies diagnósticas legadas (`legacy-runtimes`, `legacy-providers`, `legacy-events`) agrupadas sob seção "Diagnóstico / Avançado", reduzindo ruído cognitivo para novos usuários.

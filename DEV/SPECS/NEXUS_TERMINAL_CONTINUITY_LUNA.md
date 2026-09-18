# Plano de execução Luna — Nexus: CI, quota, pools e continuidade multiplataforma

Status: EXECUÇÃO PARCIAL; P0/P1 locais e fundações P3/P4/P5/P6/P7/P8 passaram
validação Linux; implementação transacional/fallback e validação nativa restantes pendentes. Data:
2026-09-07.
Base inspecionada: `1899ca6334576d859056d48a394e51d03758f313`, branch
`feat/nexus-maximum-delivery`, com extenso trabalho staged e unstaged preexistente.
Este documento é o contrato de execução; exemplos de comandos NOVOS abaixo são
interfaces propostas, não funcionalidades já disponíveis.

Estado: P0/P1 PASS local; P3 PASS unitário com lease; P4 PASS unitário com
prioridade de provider; P5 PASS inicial na UI com agrupamento e estados honestos;
P6 PASS inicial com `--supervised` opt-in; P7 PASS
inicial para sugestões no SessionHost e completion shell; P8 PASS unitário com
checkpoint Maestro schema 4. P2/P5/P9/P10/P11/P12 continuam em execução. O
modo supervisionado ainda não é padrão porque Windows/macOS nativos não foram
executados neste ambiente.

## 1. Resultado esperado e limites verificáveis

Entregar Nexus utilizável exclusivamente pelo terminal (`nexus agy --yolo`), além
de Web/Desktop, com contas agrupadas, quota rastreável, alertas sem repetição,
controle interativo, retomada e fallback configurável Codex → AGY → OpenCode.
Linux, macOS e Windows devem executar o contrato comum com backends nativos.

O Luna deve concluir cada pacote com evidência, preservar alterações anteriores,
atualizar memória DEV e preparar mudanças revisáveis. Não criar commit, push,
merge, tag ou alteração de proteção remota automaticamente. Preparar tudo localmente
antes do passo humano de publicação. Falta de acesso a runner exige provisionar ou
solicitar o acesso exato; manter gate pendente, nunca substituí-lo por compilação.

Garantias implementáveis:

- Todo número exibido tem fonte, instante da observação, unidade, janela e escopo.
  Nunca apresentar ausência de informação como 0%, 100% ou capacidade confirmada.
- No máximo um escritor Nexus autorizado por sessão lógica durante transições.
- Checkpoint persistido antes de transferir; falha de destino não vira sucesso.
- Terminal permanece conectado à sessão lógica mesmo quando o runtime muda.
- Toda capacidade anunciada tem teste correspondente e evidência por SO/arquitetura.

Não existe garantia de quota externa eternamente atualizada, memória interna
idêntica entre modelos, nem execução exatamente uma vez de efeitos externos de
CLIs arbitrárias. Implementar freshness, evidência de retomada, reconciliação e
pausa explícita quando o efeito de uma ação interrompida não puder ser determinado.

## 2. Melhorias de desenho incorporadas

1. Sessão lógica estável acima de gerações de processos. Reutilizar Agent/RuntimeGeneration
   existentes; para terminal avulso, adicionar vínculo durável mínimo compatível.
2. Pool de roteamento separado de identidade de conta, credencial e domínio de quota.
   Dois perfis da mesma assinatura podem compartilhar limite: nunca dobrar capacidade.
3. Um coletor coordenado por usuário/data root, sem exigir Web, Desktop ou daemon do SO.
   Hosts elegem um líder com lease; clientes CLI apenas consomem observações/eventos.
4. Separar permissão YOLO de autorização para trocar provider. Autonomia, custo,
   modelos e destino são políticas explícitas, não consequências de `--yolo`.
5. Checkpoints incrementais antes do esgotamento. Quando a quota acabar, a origem
   pode já não conseguir produzir resumo algum.
6. Controle de terminal em modo explícito, com fallback de teclado portátil.
   Ctrl+Space é opcional: combinações podem ser absorvidas pelo terminal/SO.
7. Rollout supervisionado inicialmente opt-in, com promoção a padrão somente após
   paridade por provider/SO. Modo direto permanece rota de compatibilidade.

## 3. Evidência local que determina o plano

As referências são da árvore de trabalho inspecionada, não prova de CI remoto atual.
Luna deve revalidar símbolos porque linhas podem mudar.

| Evidência | Local existente | Consequência |
| --- | --- | --- |
| Lançamento curto chama adapter.Run | `internal/app/app.go:265`; `internal/core/provider/adapters/agy/agy.go:100`; `internal/runtime/runtime.go:189` | `nexus agy` ainda não recebe slash do SessionHost |
| Attach copia bytes e retorna nil ao terminar stream | `internal/app/control_cmd.go:443` | Criar reconexão por geração, cancelamento de leitura e exit status real |
| Slash retém prefixos antes do processo | `internal/control/host/slash_prefix.go:62`; `slash_router.go:47` | Revisar coexistência com autocomplete nativo, paste, teclas e eco |
| Checkpoint com objetivo vazio nos dois handoffs | `internal/control/handoff/context.go:97`; `account.go:117` | Acrescentar objetivo e contrato Maestro persistentes |
| Kickoff só pede inspeção dos arquivos | `internal/control/handoff/context.go:26` | Reidratação obrigatória e estados honestos de confirmação |
| Handoff entre providers aguarda tempo limitado e rollback só muda registro | `internal/control/handoff/context.go:114` | Provar término da origem; recuperação deve recuperar processo, não só label |
| Monitor singleton apenas por processo | `internal/nexus/quota_monitor_service.go:18` | Eleição/lease e dedupe entre processos |
| Monitor aceita LIVE/CACHED e pula demais | `internal/nexus/quota_monitor_service.go`, método `check` | Tratar RATE_LIMITED e indisponibilidade do monitor explicitamente |
| Cursor declara Usage:false | `internal/core/provider/adapters/cursor/cursor.go:28`; `internal/profile/account.go:17` | Auditar autenticação/cobertura separadamente de quota |
| Prioridades existentes são por perfil | `internal/core/config/config.go:46` | Adicionar política entre providers sem reinterpretar config antiga |
| Recência e identidade já têm implementações | `internal/control/workspace/workspace.go:24`; `internal/core/config/canonical_path.go:83` | Consolidar e testar, não criar terceira canonicalização |
| UI escolhe perfil concreto | `web/src/nexus/ResourcePicker.tsx`; `web/src/features/work/DirectSessionLauncher.tsx` | Introduzir seleção por pool mantendo override avançado |
| Missões já possuem fallback | `internal/nexus/mission_executor.go`; `internal/nexus/continuity.go` | Compartilhar decisão e transação com terminal, não duplicar scheduler |
| Jobs nativos e Desktop já existem | `.github/workflows/ci.yml`; `.github/workflows/release.yml`; `.goreleaser.yaml` | Estender gates existentes e conservar evidência do mesmo commit |

Documentos DEV contêm observações históricas conflitantes: por exemplo, requisito
antigo de ambos os grupos AGY disponíveis versus roteamento por modelo. Resolver
pelo contrato de modelo/janelas deste plano e registrar decisão; não copiar afirmações
históricas de PASS para uma nova árvore.

## 4. Arquitetura e contratos

Fluxo proposto para uso interativo supervisionado:

```text
nexus agy --supervised --yolo
  → seleção/política compartilhada
  → sessão lógica + SessionHost persistente
  → PTY Unix / ConPTY Windows
  → CLI do provider

cliente terminal / Web / Desktop
  → protocolo de controle versionado
  → sessão lógica → geração atual

coletores → observações de quota → pool/scheduler → mesma transação de handoff
```

O cliente local controla teclado/renderização; o host controla processo, autoridade
de escrita e eventos. Host destacado é necessário para detach sobreviver ao cliente.
Não iniciar servidor HTTP nem interface gráfica para usar CLI supervisionada.

Contratos propostos (adaptar tipos existentes, migrações aditivas):

- `QuotaObservation`: id, provider, quotaDomainID, accountID, profileID, modelGroup,
  windowID, remaining, unit, observedAt, fetchedAt, expiresAt, resetAt opcional,
  source, status, reason. Renovar fetchedAt não renova observedAt de rollout antigo.
- `ResourcePool`: id, provider, membros, grupos/modelos elegíveis e disponibilidade
  calculada. Não somar percentuais entre contas, janelas ou providers.
- `RoutingPolicy`: lista ordenada de providers, prioridades por perfil, famílias
  permitidas, política de desconhecidos, limites de tentativas/custo, cooldown,
  condições de retorno ao prioritário e autorização para contexto sair do provider.
- `Session`: id estável, geração atual, objetivo, política, checkpoint, workspaceRef.
- `HandoffTransaction`: id idempotente, origem/destino, etapa, fencing token,
  evidências, erro/recuperação, timestamps. Persistir antes dos efeitos.
- `ContextPack`: versão, objetivo, próximos passos, decisões, testes/pendências,
  identidade do workspace, commit/diffs staged e unstaged, untracked allowlisted,
  fontes DEV com digest/freshness, comandos em andamento e ações incertas.

Disponibilidade é por operação/modelo: basta um candidato elegível para o pool
atender; esse candidato precisa satisfazer TODAS as janelas aplicáveis à operação.
Se Gemini tiver 0% e Claude tiver quota no AGY, o pool é parcialmente disponível,
sem sugerir Gemini. Se a janela semanal estiver zerada, 5h positiva não libera uso.
Um 429 transitório é rate limit, não prova automática de saldo financeiro zerado.

## 5. Pacotes de execução e aceitação

### P0 — Baseline, inventário e plano de evidências

- Ler AGENTS/hierarquia Maestro e DEV em ordem; ler os dois padrões em
  `docs/engineering/`. Capturar branch, SHA, status e diffs staged/unstaged sem
  divulgar segredos. Não usar `git add .`, reset, checkout destrutivo ou reparador
  antigo preso ao SHA sem revisar patch por patch.
- Inventariar binários/versões e capacidades reais de Codex, AGY, Claude, Gemini,
  OpenCode e Cursor; distinguir executável instalado, login e quota disponível.
- Criar `DEV/validation/terminal-continuity/` com manifestos por execução:
  commit, hash da árvore candidata quando suja, SO/arch/versão, ferramentas,
  comando, resultado, timestamp, artefatos e limitações.
- Executar baseline proporcional e mapear falhas atuais; resultados históricos
  de DEV são contexto, não validação nova.
- Aceitação: inventário e backlog rastreáveis; nenhuma alteração anterior perdida.

### P1 — Fechar CI/runtime existente antes da promoção

- Auditar patches locais em canonical_path, worktree, workspace, Codex shared-host,
  terminal_windows, process_alive, host fixtures, testes app/web e resources.ts.
- Consolidar comparação por identidade; fake clock/sequence para recência;
  JSON via marshal, PATH com separador nativo e fixtures executáveis por SO.
- Windows: verificar direção/lifetime de handles ConPTY, CreateProcessW,
  environment UTF-16 com terminador duplo, named pipe readiness, PID vivo,
  Job Objects e cleanup de processo filho/neto.
- Corrigir causa antes de atribuir cascata de Named Pipe ao ConPTY. Listener já
  é criado antes do provider no host atual: não reimplementar hipótese antiga.
- Aceitação: testes focados e completos verdes nativamente; logs preservados
  inclusive em falha; zero skip novo para mascarar funcionalidades centrais.

### P2 — Quota confiável e auditoria de todos os providers

- Centralizar leitura/normalização em `internal/core/quota`, adapters e
  `internal/profile/usage.go`; UI e scheduler consomem a mesma observação.
- Validar NaN, infinito, negativos, limites/unidades, payload parcial, clocks,
  reset ausente, dado expirado e associação account/profile/host.
- Codex: fixtures de rollout com identidade correta, arquivo recente com evento
  antigo, auth compartilhada, aliases de caminho e ausência de rate limits.
- Separar `UNKNOWN`, `UNSUPPORTED`, `ERROR`, `STALE`, `ESTIMATED`, quota confirmada
  e `RATE_LIMITED`; migrar estados legados explicitamente, sem default permissivo.
- Auditar seis providers: fonte nativa autenticada ou artefato local oficial;
  verificar documentação oficial/versão instalada antes de integrar protocolos.
  Se não houver fonte numérica, monitorar saúde/erros e mostrar “quota não exposta”.
- Acrescentar diagnóstico de cobertura em `nexus doctor --json` e usage: por que
  Codex não alertou, última observação, coleta falhou ou limite não foi cruzado.
- Aceitação: fixtures não cruzam contas; dado expirado não fica verde; nenhum
  provider fica silenciosamente confundido com capacidade ilimitada.

### P3 — Alertas únicos e coleta coordenada

- Evoluir `quota_monitor_service.go` e `quota_monitor.go` com lease interprocesso,
  heartbeat, fencing e persistência transacional usando armazenamento existente
  ou mecanismo nativo portátil justificado. Uma mutex Go não basta.
- Identidade do alerta: domínio de quota, grupo, janela, episódio/ciclo estável,
  limiar e canal. Texto do countdown nunca é chave.
- Histerese configurável; queda para limiar mais grave alerta uma vez; recuperação
  real rearma; reboot ou cache refresh isolado não rearma.
- Deduplicar aliases da mesma conta e separar consumo de aviso “monitor degradado”.
  Tratar notificação por canal com id estável e outbox/recibo quando possível;
  documentar janela de crash de canais sem entrega idempotente.
- Deadline/backoff por provider: uma coleta travada não bloqueia as outras.
  CLI foreground encerrado não pode deixar goroutine sem owner.
- Aceitação: 3 processos simultâneos e 10 polls a 22% geram um alerta por canal;
  morte do líder recupera coleta; queda adicional e recuperação/reset rearmam
  corretamente; quotas desconhecidas não disparam “saldo zero”.

### P4 — Pools e política única de seleção

- Reutilizar `internal/core/scheduler`, `internal/nexus/scheduler.go`, resource
  discovery e recomendações; evitar decisões diferentes entre CLI e Web.
- Perfil continua dono de credenciais isoladas; pool apenas referencia membros.
  Identidade verificada de assinatura define quota compartilhada; e-mail sozinho
  não prova compartilhamento nem independência. Incerteza fica explícita.
- Adicionar prioridades entre providers e regras por modelo/workspace; sugestão
  inicial Codex → AGY → OpenCode, configurável. OpenCode também pode usar o mesmo
  upstream de outro provider: respeitar domínio de quota/cooldown compartilhado.
- Preservar configurações antigas e pin explícito de perfil. Não transformar
  automatic_fallback legado em permissão silenciosa de troca entre providers.
- Oferecer explicação determinística da escolha e descartes com snapshot IDs.
- Aceitação: dois aliases não duplicam saldo; pool seleciona candidato elegível;
  candidatos empatados têm ordem estável; nenhum ciclo de fallback entre aliases.

### P5 — Layout de uso consistente em CLI/Web/Desktop

- Atualizar `internal/tui/usage_table.go`, apresentação em app.go, ResourcePicker,
  DirectSessionLauncher e telas de uso existentes. Pesquisar componentes antes.
- Default agrupado por provider/pool e família de modelos; expandir contas
  mostra perfil ativo, quota independente/compartilhada, fonte e horário.
- Vermelho + texto/ícone para quota esgotada; verde só para capacidade confirmada;
  parcial e desconhecida visualmente distintos. Estado auth tem indicação própria.
- Destacar recursos utilizáveis, mantendo esgotados localizáveis. Sem ordenar
  dinamicamente de modo a mover item sob cursor durante escolha.
- Mostrar “2 contas elegíveis”, nunca “160% de quota”. Não derivar custo monetário
  de percentual; custo só com fonte e unidade verificadas.
- SCSS Modules, tokens, i18n pt/en/es, teclado, contraste e modo sem cor no CLI;
  texto/JSON usam os mesmos dados, sem escapes ANSI no JSON.
- Aceitação: mesmas observações produzem mesmos estados nas três superfícies;
  null arrays, nomes longos, 80×24, 120×30 e viewport web 320–1440px cobertos.

### P6 — Supervisão CLI portátil e transparente

- Refatorar caminho interativo curto para reutilizar Launcher/SessionHost/attach,
  opt-in inicial `--supervised`, mantendo seleção, flags e credenciais equivalentes.
  `--direct` é escape proposto, não existente; flags Nexus são removidas antes
  do provider, e `--` preserva passagem literal conforme contrato documentado.
- Não interceptar help/login/print/pipelines automaticamente. Detectar stdin/stdout
  TTY, preservar stdout máquina, stderr e status de saída no modo não interativo.
- Host persistente + cliente terminal cancelável. Versionar saída/control channel
  separadamente de bytes PTY; recuperar código do processo real, distinguir detach,
  falha de transporte, morte do host e encerramento voluntário.
- Resolver leitura bloqueante de stdin ao reconectar: um owner por cliente, não
  uma goroutine nova de io.Copy por geração. Não perder input nem duplicar writes.
- Resize inicial e contínuo; Unix SIGWINCH; Windows eventos de console ou polling
  limitado como fallback. Restaurar modo terminal em todos os caminhos de saída.
- Encerrar árvore via grupos de processos Unix e Job Objects Windows. Ctrl+C deve
  conservar a semântica interativa do provider; stop Nexus é operação distinta.
- Host inicial mantém UI nativa; autenticação e normalização de --yolo continuam
  nos componentes existentes, sem ampliar permissões por causa da supervisão.
- Aceitação: paridade de teclado, Unicode, resize, exit codes 0/1/42, filhos/netos,
  detach/attach, SSH e interrupção em Linux/macOS/Windows, sem browser aberto.

### P7 — Comandos e autocomplete no terminal

- Um catálogo de comandos produz help, completions e controles Web/Desktop.
  Comandos propostos: `/nexus status`, `usage`, `accounts`, `continue <provider>`,
  `handoff <profile>`, `detach`, `stop`; seleção automática de conta fica no backend.
- Resolver sintaxe atual divergente: slash `continue <provider>` versus onboarding
  `continue --with <provider>`; suportar alias documentado ou corrigir onboarding
  com teste de execução de todos os exemplos publicados.
- Cliente fornece modo de comando Nexus com prefixo de teclado configurável
  (proposta Ctrl+] seguido de `n`) e `/nexus<TAB>` quando em editor Nexus.
  Ctrl+Space é alternativa opcional, não único acesso.
- Não inferir linha vazia de qualquer CR/LF: TUI tem histórico, edição multiline,
  paste e eventos de cursor. Revisar router existente; modo explícito garante
  acesso independente da CLI. Integrações nativas opcionais via adapter/plugin
  apenas quando suporte oficial for comprovado, sem editar binário do provider.
- Para renderização local, definir owner único: modo de controle assume tela,
  mantém estado VT/saída em buffer limitado e restaura tela do provider usando
  estado conhecido. Não confiar em imprimir ANSI por cima de uma TUI ativa ou
  reaplicar um trecho de ring buffer como reconstrução universal de tela.
- Autocomplete exibe destino/pool elegível e comando; Tab/setas/Enter/Esc,
  backspace, paste, UTF-8 fragmentado e cancelamento testados. Dados de quota
  assíncronos não bloqueiam renderização.
- Completar shell antes do lançamento (bash/zsh/fish/PowerShell) é entrega distinta
  do autocomplete interno; ampliar completionCmd a partir do catálogo.
- Aceitação: `/help` nativo preservado; `/nexus` executa uma vez; textos colados
  com slash não executam controle; escape literal funciona; redraw durante menu
  não corrompe a tela; nenhum comando Nexus chega ao LLM por acidente.

### P8 — Contexto Maestro durável antes de qualquer troca

- Evoluir checkpoint/context existentes, reutilizando contextsnapshot, readiness
  e persistência DEV; Maestro continua opcional como instalação global.
  Sem instalação, Nexus usa contrato de checkpoint local compatível, sem criar
  skills globais paralelas ou instalar Maestro silenciosamente.
- Persistir objetivo ao iniciar/adotar tarefa, decisões e próximo passo ao finalizar
  unidades de trabalho; quando faltar, sinalizar lacuna. Não inventar objetivo
  a partir do nome da pasta ou afirmar que capturou conversa oculta.
- Ler DEV INDEX/README → HANDOFF → CONTEXT → SPECS/ACTIVE → VERIFY e referências
  relevantes. Registrar hash, omissões e limites; truncar não pode virar “completo”.
- Snapshot consistente após cessarem escritores; checkpoint incremental durante
  atividade é preliminar até reconciliação. Capturar staged/unstaged/untracked
  allowlisted, arquivo renomeado/deletado, worktree e repo sem Git.
- Escrita atômica, schema versionado, limites de bytes/tempo, ACL privada Windows
  e permissões Unix, redaction, proteção contra symlink fora de workspace e
  retenção configurável. Não copiar tokens, auth.json ou chaveiros entre contas.
- Contexto de projeto é dado; não deve elevar privilégios nem redefinir política.
- Confirmar entrega separadamente de reidratação: `CONTEXT_SAVED`,
  `CONTEXT_DELIVERED`, `REHYDRATION_CONFIRMED` apenas com recibo verificável de
  checkpoint/digest/workspace via integração cooperante. CLI genérica sem recibo
  permanece não confirmada; habilitar retomada assistida ou modo automático
  explicitamente degradado, sem anunciar garantia inexistente.
- Aceitação: quota termina antes do resumo e checkpoint anterior recupera tarefa;
  truncamento/arquivo ausente/objetivo vazio ficam visíveis e bloqueiam modo
  estrito; integração cooperante confirma e termina uma tarefa de teste real.

### P9 — Handoff transacional e reconexão à sessão lógica

- Compartilhar transação entre account.go, context.go e fluxo de missões.
  Estados: REQUESTED → PREFLIGHT → QUIESCING → CHECKPOINTED → STARTING_TARGET
  → VERIFYING → COMMITTED; falhas: RECOVERING ou NEEDS_ATTENTION.
- Preflight valida CLI/auth/modelo/permissões/quota e compatibilidade de resume.
  Sessão nativa entre contas só quando fonte/destino e versão suportam; caso
  contrário usar contexto explicitamente, sem copiar credenciais.
- Parar escritor e confirmar término antes de iniciar destino que escreve;
  timeout sem confirmação impede ativação. Abrir destino em standby apenas se
  protocolo realmente impedir tools/escrita até commit.
- Persistir intenção e fencing; requisição repetida/destino já iniciado é
  reconciliada por id e identidade de processo, nunca apenas PID reutilizável.
- Rollback relança/retoma origem quando possível; se não, estado interrompido
  recuperável. Proibido marcar RUNNING um processo que já morreu.
- Cliente segue geração via canal de controle mesmo se socket antigo fechar.
  Ring replay com sequência/bounds; não duplicar prompts na reconexão.
- Efeitos externos em andamento: registrar ação incerta e reconciliar antes de
  repetir deploy/pagamento/push. Fencing interno não desfaz efeito remoto.
- Aceitação: fault injection em cada etapa, dois pedidos simultâneos, host morto,
  destino sem auth, source que ignora stop, restart no meio da transação;
  no máximo um escritor e nenhuma falsa continuidade confirmada.

### P10 — Fallback automático e espera retomável

- Reutilizar P4/P9 no mission_executor e sessões interativas. Autorizar via política
  independente e persistente; manual funciona mesmo com automático desligado.
- Confirmar esgotamento/rate limit por evidência; separar erro de auth, rede,
  cancelamento do usuário e crash de quota. Não trocar por toda saída não zero.
- Esgotar candidatos elegíveis do provider prioritário, depois próximo permitido.
  Respeitar domínio upstream compartilhado, orçamento, modelo e cooldown.
- UNKNOWN não é verde. Default estrito: refresh e aguardar se não houver capacidade
  confirmada. Opção configurável de tentativa limitada usa evidência operacional,
  sem produzir número fictício e sem executar probe oneroso silenciosamente.
- Não retornar imediatamente ao Codex quando recupera se outro provider trabalha;
  reconsiderar no limite da tarefa/turno para evitar alternância contínua.
- Sem candidatos: `WAITING_FOR_QUOTA`; contexto salvo, próximo check/reset mostrado,
  timers canceláveis e jitter/backoff; não suspender o computador. Depois de reboot,
  revalidar leases, auth, quota e contexto antes de retomar.
- Pausa/stop do usuário persistem e impedem wake automático. Se reset desconhecido,
  usar intervalo limitado configurado; nenhum busy loop.
- Aceitação: Codex A→B→AGY A→OpenCode→espera→recuperação sem perder sessão lógica;
  prioridades editadas funcionam; pause impede restart; config antiga não opta
  silenciosamente por compartilhamento de contexto com outro provider.

### P11 — Diagnóstico, paridade e benefícios adicionais

- `doctor` mostra transporte, capacidades efetivas, quota/monitor, leader,
  checkpoint e motivo de seleção; export redigido e --json.
- Eventos com sessionID, generation, transactionID e observationID; sem prompts
  ou credenciais nos logs comuns. Métricas de tempo de startup/handoff, falhas e
  recuperação; custo somente quando mensurável.
- Notificações de input/aprovação só se confirmadas; scraping TUI segue fallback
  explicitamente heurístico. Detach mantém host; fechar terminal não implica stop.
- Web/Desktop usam mesmos comandos/política e seguem runtime novo automaticamente.
- Aceitação: sessão iniciada por CLI pode ser observada/retomada nas superfícies
  existentes sem roubar escrita; observador lento não bloqueia processo.

### P12 — Matriz nativa, promoção e publicação

- Executar matriz abaixo; corrigir causas até contrato comum passar nos três SOs.
- Comparar direct/supervised com fixture determinística: p95 de echo local ≤100ms
  para 1KiB/100 amostras no runner, sem perda em stream de 10MiB; repetir 100 ciclos
  start/stop sem processos órfãos nem crescimento contínuo de handles/goroutines.
  Guardar baseline e hardware; investigar ruído antes de alterar meta.
- Soak de 8h com fake provider, resets simulados, detach, falhas de rede/IPC,
  crash/restart e duas superfícies; depois smoke autenticado limitado por provider.
- Promover modo supervisionado por plataforma/provider somente com evidência de
  paridade. Objetivo final cobre toda a matriz suportada; opt-in é etapa de rollout,
  não dispensa de resolver Windows/macOS.
- Preparar diff/PR/checklist/rollback; publicação humana gera SHA candidato.
  Confirmar jobs remotos todos nesse SHA e gates do merge result/main quando
  diferente. Não misturar PASS de commits distintos; skipped/cancelled não é PASS.
- Preparar regras de main (PR, required checks estáveis, up-to-date/merge queue
  conforme fluxo existente, bloquear force push); aplicação administrativa só
  após autorização. Sem bump/tag até gates e release checklist aprovados.
- Aceitação: CLI/runtime, Frontend, Security, Browser/Axe, Desktop por SO e
  GoReleaser verdes; artefatos com versão/SHA/checksum coerentes; matriz pública
  atualizada com links para evidência, sem suporte nativo presumido de cross-build.

## 6. Matriz obrigatória e resolução por plataforma

| Alvo | Implementação a validar | Evidência obrigatória |
| --- | --- | --- |
| Linux amd64/arm64 | PTY, UDS privado, grupos de processo, SIGWINCH, Secret Service conforme disponibilidade | Execução nativa por arquitetura prometida; SSH e desktop WebKitGTK |
| macOS amd64/arm64 | PTY/UDS, SIGWINCH, Keychain, aliases /var e /private/var | Runner Intel e Apple Silicon; terminal e app WKWebView |
| Windows amd64/arm64 | ConPTY, named pipe ACL, Job Object, UTF-16, resize, Credential Manager | Execução nativa por arquitetura prometida; PowerShell/cmd e WebView2 |

Usar a matriz oficial `docs/platform/PLATFORM_SUPPORT_MATRIX.md` e artefatos
`.goreleaser.yaml` para estabelecer exatamente quais arquiteturas são entregues.
Não remover alvo para conseguir verde. ARM64 sem runner hospedado exige máquina
self-hosted/VM nativa disponível; emulação e cross-build são evidências auxiliares.
Validar Windows 10/11 e versões mínimas anunciadas de macOS/Linux com VMs/hardware
adequados: windows-latest Server e macos-latest não provam suporte ao SO mínimo.
Registrar versão mínima efetiva do ConPTY e dependências WebView/WebKit detectadas;
provisionar imagem compatível e reproduzir, em vez de deixar teste ignorado.

Para cada combinação SO/provider disponível oficialmente: help/version, lançamento
autenticado, digitação, cancelamento, resume, auth isolada, quota/unknown correto,
handoff permitido, contexto e cleanup. Quando fornecedor não distribui CLI para
um alvo, concluir núcleo com fake provider e oferecer adapter remoto explícito
(por exemplo host via SSH com workspace e política definidos) como extensão
separada; não chamar isso execução nativa nem fabricar capacidade do fornecedor.
Bloqueio externo deve identificar pacote/versão/acesso faltante e ação concreta.

Fixtures multiplataforma obrigatórias: shell interativo, CLI full-screen fake,
output Unicode fragmentado, ANSI/OSC, alternate screen, bracketed paste,
processo que ignora stop, filho/neto, quota emitida em etapas e provider sem quota.
Fuzz em prefix/router, parsing quota e protocolo; controlar buffers/timeouts.

## 7. Verificação e organização do trabalho

Ordem: P0 → P1; P2→P3/P4→P5; P6→P7; P8→P9; P3/P4/P9→P10;
P5/P7/P10→P11→P12. P2, P6 e P8 podem avançar em paralelo após baseline,
com contratos acordados; integração sempre serial para arquivos compartilhados.

Luna é executor/integrador. Se houver subagentes, distribuir quota/pools,
terminal/plataformas e checkpoint/transação; frontend entra após contrato P4.
Designar revisor independente para concorrência/recuperação e evidências antes
da promoção. Não afirmar consenso/revisão externa se ferramenta não foi executada.
Ralph pode persistir na execução conforme skill disponível; não inventar comandos
do modelo Luna, flags OMX ou disponibilidade de subagentes.

Verificações focadas por pacote, depois gate global do candidato:

```bash
go test -count=1 ./internal/core/quota/... ./internal/core/scheduler/...
go test -count=1 ./internal/control/host/... ./internal/control/terminal/...
go test -count=1 ./internal/control/protocol/... ./internal/control/handoff/...
go test -count=1 ./internal/nexus/... ./internal/app/... ./internal/tui/...
go test -count=1 ./...
go test -race -count=1 ./...
go vet ./...
make quality
make security
git diff --check
```

Frontend: `cd web` e `bun install --frozen-lockfile`, `bun run verify`;
confirmar scripts atuais para format/typecheck/lint/styles/tests/build e browser.
Go lint: usar versão fixada no projeto. Windows deve executar comandos Go e scripts
nativos equivalentes se Make não estiver instalado; não depender de Bash para
declarar sucesso Windows. `-race` só nos alvos suportados pelo toolchain; complementar
outros com stress/fault injection nativos e registrar diferença. Desktop usa jobs
Wails existentes; snapshot usa `.goreleaser.yaml` sem publicar.

Relatório final por requisito: implementado, verificado, evidência, limitação e
próxima ação externa se houver. Verificações antigas não contam após mudança
material. Não repetir suíte inteira sem motivo a cada edição de documentação.

## 8. Riscos, mitigação e rollback

| Falha prevista | Prevenção/recuperação | Teste decisivo |
| --- | --- | --- |
| Menu corrompe TUI ou consome slash nativo | Owner explícito de input/VT, comando portátil e replay correto | Redraw + paste + resize enquanto menu aberto |
| Quota antiga troca para conta inválida | observedAt, TTL, domínio, refresh e explicação | Rollout novo com evento antigo e contas alias |
| Dois hosts notificam/escrevem juntos | Lease, fencing, idempotência e transação | Matar líder no ponto de commit e reconectar |
| Destino falha após origem parar | Recuperação real ou estado interrompido recuperável | Falha injetada em cada transição |
| Task repete ação externa | Ledger de pendências e reconciliação | Interromper após efeito antes do recibo |
| Windows retém pipe/filho | Lifetime de handles, Job Object, close idempotente | 100 start/stop + kill do cliente/host |
| Mudança quebra scripts do usuário | Modo não TTY preservado e rollout opt-in | Pipes/redirecionamento e exit codes |

Rollback de feature: desabilitar supervisão/automático para novos launches;
sessões existentes permanecem gerenciadas até terminar/detach. Não iniciar modo
direto automaticamente se a origem talvez ainda esteja viva. Migração aditiva
preserva IDs/credenciais e permite leitura compatível; testar restore de backup
redigido de configuração/estado. Nunca sobrescrever trabalho Git do usuário.

## 9. Prompt pronto para entregar ao Luna

> Execute `DEV/SPECS/NEXUS_TERMINAL_CONTINUITY_LUNA.md` como contrato do trabalho.
> Leia a hierarquia AGENTS/Maestro e DEV antes de agir. Preserve o worktree dirty
> e revalide evidências locais; exemplos novos no plano precisam ser implementados.
> Comece em P0/P1, depois respeite dependências P2–P12. Reutilize os componentes
> indicados e use Ralph/Maestro quando disponíveis. Resolva Linux, macOS e Windows
> com testes nativos, sem trocar falha por skip, número inventado ou capacidade
> presumida. Atualize DEV/WORKLOG, VERIFY e HANDOFF após cada pacote com evidência
> e próximo passo. Conclua implementação e preparação revisável; não faça commit,
> push, merge, tag ou configuração administrativa remota automaticamente. Se uma
> permissão ou runner impedir o gate final, conclua o trabalho independente e
> apresente o acesso/comando exato necessário, mantendo o gate pendente.

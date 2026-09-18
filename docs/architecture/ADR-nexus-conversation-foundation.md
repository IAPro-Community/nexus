# ADR: Fundação conversacional do Nexus (assistant-ui + ExternalStore)

## Status
Aceito — Fase 1 (Composer)

## Contexto
O Composer Web renderiza turns como `role + content` em cards, sem infraestrutura
reutilizável de chat. Nexus Work, agentes, multimodal e Computer Use precisarão
da mesma experiência conversacional. Duplicar chat por feature viola o padrão
do Workspace OS.

O Nexus já é dono de sessões, turns, LivingBrief, skills, artifacts e
persistência SQLite. Não devemos adotar um runtime que torne o provider/modelo
do pacote de UI a fonte de verdade.

## Decisão
1. **UI/UX conversacional**: `@assistant-ui/react` (MIT, peer React 18/19) como
   primitives de thread, input, attachments e Tool UI.
2. **Runtime**: `ExternalStoreRuntime` (`useExternalStoreRuntime`). O Nexus
   continua dono de messages, sessions, streaming e persistence.
3. **Adapter**: `NexusConversationAdapter` mapeia eventos Nexus ↔
   `ThreadMessageLike` via `nexusConversationMapper`.
4. **Sem Vercel AI SDK** como runtime central.
5. **Sem segundo design system**: primitives restilizadas com tokens `--nx-*`
   e SCSS Modules.
6. **Attachments**: domínio `NexusAttachment` + storage em `DataDir`; nesta
   fase PNG/JPEG/WEBP na UI sem VisionProvider.
7. **Bootstrap create**: POST `Accept: application/x-ndjson` emite steps +
   heartbeat; UI mostra AgentActivity (não “Iniciando…” cego).

### Alternativa rejeitada: AssistantTransport
Só faria sentido se o backend já emitisse snapshots ricos de agente. Hoje o
Composer devolve JSON de sessão / turns síncronos. ExternalStore evita
acoplamento prematuro.

## Consequências
- Uma superfície `ConversationSurface` reutilizável por Composer e, depois, Work.
- Turns existentes (`USER`/`ASSISTANT` + content) permanecem válidos; `kind` +
  `payload_json` estendem o modelo sem quebrar histórico.
- Bundle cresce com assistant-ui; spike valida esbuild + React 19 antes da
  migração completa.
- CUA / Vision / multimodal routing ficam para fases posteriores.

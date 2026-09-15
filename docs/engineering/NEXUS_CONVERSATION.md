# Nexus Conversation — contrato da fundação

## Como criar uma nova surface

```tsx
import {
  ConversationSurface,
  useNexusConversationRuntime,
  nexusEventsToThreadMessages,
} from '../features/conversation';

function MySurface({ events, onSend, inspector }) {
  const runtime = useNexusConversationRuntime({
    events,
    onSend,
    isRunning: false,
  });
  return (
    <ConversationSurface
      runtime={runtime}
      inspector={inspector}
      emptyLabel={t('conversation.empty')}
    />
  );
}
```

## Tipos de evento

| kind | Uso |
|------|-----|
| `user` | Mensagem do usuário (texto ± imagens) |
| `assistant` | Resposta do assistente |
| `system` / `status` | Status operacional |
| `activity` | AgentActivity (passos, progresso) |
| `tool_call` / `tool_result` | Tool UI |
| `artifact` | Artefato (prompt final, etc.) |
| `attachment` / `image` | Referência a NexusAttachment |
| `approval` | Aprovação humana |
| `warning` / `error` | Erros amigáveis |
| `completion` | Conclusão de run |

Nunca mostrar chain-of-thought privada. Activity = eventos operacionais reais.

## Persistência

Fonte de verdade: backend Nexus (SQLite). assistant-ui só renderiza o snapshot
convertido. Refresh deve restaurar ordem e sessão.

## Attachments

`NexusAttachment`: id, sessionId, name, mimeType, size, type, storageRef,
metadata, createdAt. Arquivos em `DataDir/attachments/`. MIME aceitos na fase 1:
`image/png`, `image/jpeg`, `image/webp`.

## Streaming

O runtime aceita updates incrementais. Create Composer usa NDJSON
(`step` / `heartbeat` / `complete` / `error`). Não simular com timers de UI.

## Limites de performance

Históricos longos: paginação / windowing documentados como follow-up se
passarem de ~500 mensagens na thread. Imagens nunca ficam como base64 em
estado React de longa duração — só URL/blob temporário de preview.

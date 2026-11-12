# ADR — Compartilhamento de observações por grants viewer (workspace-sharing)

**Status:** Proposed (implementação iniciada sob override explícito do maestro: loop contínuo sem perguntas; owner pode reverter)
**Data:** 2026-11-10 · **Capability:** `workspace-sharing` · **Issue:** 109

## Contexto
O Nexus compartilha observações de runtime/custo (`CostObservation` por projeto/agent) apenas com sessões autenticadas do host. Para trabalho remoto multi-dispositivo e revisão por terceiros, é necessário compartilhar observação por projeto/Agent sob RBAC, começando viewer-only.

## Decisão
Grants de leitura (`ObservationShareGrant`) com escopo `project_id` obrigatório e `agent_id` opcional:

1. **Viewer-only inicial**: único papel é `viewer`, autoriza apenas `read`. Nenhuma ação de mutação é exposta pelo caminho do grant.
2. **Negar por padrão com razão explícita**: grant desconhecido, revogado, projeto divergente (cross-project leakage), agent fora do escopo ou ação != read — todos negados com razão determinística (`Authorize` retorna `(reason, ok)`).
3. **Revoke imediato**: `Revoke` marca `RevokedAt` e toda checagem posterior consulta o estado corrente do registry (sem cache de decisão), então a negação é efetiva na requisição seguinte.
4. **Registry em memória com mutex** (fase 1): decisões são baratas e centralizadas; persistência durável fica para fase 2 no evidence path.

## Alternativas consideradas
- **Sessões derivadas do QR com escopo por projeto**: reutiliza auth, mas o aceite exige grants nomeáveis/revogáveis individualmente — sessões não dão revoke granular imediato.
- **RBAC completo multi-usuário**: escopo muito maior, sem necessidade demonstrada hoje (viewer-only é o aceite).
- **Tokens opacos sem registry central**: espalharia a checagem e dificultaria o revoke imediato auditável.

## Consequências
- + Caminho de leitura compartilhada com deny-by-default testável e revoke imediato.
- − Grants em memória não sobrevivem a reinício (gap explícito no evidence map; fase 2 = persistência no evidence path assinado).
- Riscos: vazamento por grant muito amplo — mitigado por escopo obrigatório de projeto e teste de cross-project; enumeração de grants — IDs aleatórios criptográficos (`crypto/rand`), não sequenciais.

## Verificação
- Testes puros: grant→read ok; cross-project negado; revoke imediato; viewer não muta; escopo de agent respeitado; grant desconhecido negado.
- Teste de integração: fluxo completo por HTTP (criar → ler observações → revogar → 403).
- Model: `internal/nexus/workspace_sharing.go` · Handler: `internal/control/web/handlers_workspace_sharing.go`.

## Follow-ups
- Persistir grants (evidence path assinado) — fase 2.
- Expiração (TTL) opcional por grant.
- Auditoria de acesso (log de reads por grant).

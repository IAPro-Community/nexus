# Plano de finalização do Nexus 1.0 e integração Maestro

Atualizado em 2026-11-24. A missão de continuidade nesta rodada termina após disponibilizar o candidato local e pausar para o ensaio do maestro. O candidato é baseado em `release/v1.0.0@6c322427` com árvore modificada. O estado geral do release permanece **NO_GO**.

## Checkpoint atual — continuidade de contas

- **Implementação local concluída.** Refresh de quota concorrente é coalescido por perfil no processo; nomes legados inválidos como `--with` são filtrados; a TUI identifica linhas/grupos e classifica pelo menor saldo restante; drivers precisam declarar suporte a retomada cross-account.
- **Handoff Codex protegido.** A origem precisa parar e ficar quiescente; o Nexus adquire lock exclusivo de manutenção no destino, importa a sessão/rollout exatos e verifica o arquivo e o marcador de sessão/quota antes de iniciar o destino. Falha de preparação executa rollback para recuperar a origem. Fingerprints iguais, mudança de identidade na origem e driver sem suporte são recusados.
- **Limite AGY.** Não há importação de sessão cross-account verificada. AGY é bloqueado antes da parada da origem e preserva checkpoint. A alternativa exige novo runtime com checkpoint/contexto explícito e aceite, ou suporte oficial de importação comprovado.
- **Verificação local PASS.** `make PATH="/home/desenvolvedor/go/bin:$PATH" quality-full` (lint Go, Go completo + race, Web 151 arquivos/959 testes e govulncheck) e `make build build-desktop build-hud`. CLI: versão `1.0.0`, metadata `6c322427`, Go 1.26.11.
- **Candidato isolado:** `/home/desenvolvedor/.local/share/ai-manager/profiles/codex/kivergmail/home/.local/share/nexus-continuity-check/nexus`, SHA-256 `38a918264021ec306c5efca4ab4ce16d550ef7f9de7c215ee1f5c7e3f883e945`. Backup: `nexus.pre-exact-import-20261120T044219Z` (SHA anterior `c84a3614040ed2ef4105908dfb54d17419ad21c09a6ed6d8e29999e72c3998fa`). O candidato antigo não deve ser usado.
- **Pausa para o maestro:** OAuth, refresh de quota, geração e handoff real não foram executados. O último estado sanitizado conhecido tinha `codex:kiver` sem autenticação e quotas `UNKNOWN`. Antes de qualquer geração, confirmar identidade upstream distinta e quota recente, atribuível e positiva. O plano de ensaio está abaixo.

## Ensaio do maestro — pré-requisitos e sequência

Usar o candidato isolado informado acima, não o PATH padrão. Preservar sessões e credenciais AI Manager em uso. Não copiar `auth.json`, não compartilhar tokens, e interromper se identidade, quota ou linhagem da sessão forem incertas.

1. Reabrir apenas a TUI antiga quando seguro. Confirmar que o inventário atual não recria `--with` nem aliases removidos; validar que o título conta linhas/grupos e que o status mostra o menor saldo efetivo.
2. Concluir OAuth oficial no perfil Nexus dedicado `codex:kiver`, escolhendo a conta externa realmente distinta de omegaedu. Confirmar por fingerprint sanitizado que origem e destino são contas distintas; obter `usage --refresh --json` recente e confirmar quota medida, atribuível e positiva. Não iniciar sessão se qualquer janela aplicável for desconhecida/esgotada.
3. No terminal A, abrir sessão Codex supervisionada na conta de origem e enviar um marcador curto descartável. Confirmar que a resposta foi persistida antes da troca.
4. No terminal B, executar handoff manual com `--no-attach` para o perfil de destino. No terminal A, confirmar reattach ao sucessor, pedir o marcador e verificar histórico único, checkpoint disponível e apenas um writer/geração ativa.
5. Em sessão de ensaio separada, verificar failover automático após uma limitação real da origem, com destino ainda elegível. Não provocar limitação por geração artificial. Registrar resultado sanitizado, estágio, presença de checkpoint, geração ativa e versão/hash do candidato.
6. Se qualquer etapa falhar, não insistir com prompt/replay. Registrar código/estágio redigidos, preservar checkpoint e confirmar rollback à origem; corrigir a fronteira que falhou antes de repetir.

### Decisão após o ensaio

- Sucesso manual e automático comprova apenas o cenário provider/contas/versão testado; anexar evidência ao SHA/hashes exatos e manter demais gates de release independentes.
- Falha de identidade/quota é bloqueio de pré-condição, não prova de falha de handoff. Falha de import/reattach deve permanecer NO_GO até correção e repetição.
- Após registrar o resultado do maestro, retomar as fases 2–5 abaixo. Não fazer commit, push, tag ou publicação sem autorização explícita e evidências do SHA final.

## Fase 2 — Reconciliar árvore, worktrees e P0s

1. Preservar alterações locais e worktrees paralelas. Auditar cada diff/commit exclusivo antes de integração; reconciliar as alterações preexistentes da API de eventos e demais patches locais em lotes pequenos e reversíveis na branch `release/v1.0.0`.
2. Fechar A1–A4 contra o contrato [agent-workspace-v1](../../docs/reference/agent-workspace-v1.json) e o [evidence map](../../docs/reference/agent-workspace-v1-evidence-map.json). Revisar semanticamente as assertions, não promover fixtures a certificação.
3. Executar os fluxos aplicáveis no artefato candidato: A1 capability/proveniência; A2 onboarding/cancelamento sem processos órfãos com perfis reais; A3 atenção ligada a runtime real; A4 escolha estruturada, idempotência, retry e isolamento entre runtimes.
4. Reproduzir os gates de lease/handoff pós-revogação Windows anotados em #216/#214, distinguindo falha de adapter, sincronização e fixture. Writer único e reconexão precisam de evidência.
5. Testar Comparison Lab e atribuição de custo #228 com consentimento e provedores reais. Validar balance AGY com 2/2 perfis elegíveis, evidência 5h em todos os grupos aplicáveis e atribuição correta. Se o provedor não expõe uma janela necessária, manter inelegível e documentar a limitação.
6. Auditar os 97 critérios das 48 capacidades. Classificar evidência de fonte/teste, provider real, ambiente e aceite. Critérios sem prova permanecem `pending`/`unverified`.

## Fase 3 — Homologar Maestro e congelar candidatos

1. Usar a branch runtime `v1` do Maestro. Validar descoberta/negociação pelo CLI distribuído oficial, não apenas pelo checkout local; verificar versão de capabilities e comportamento legado.
2. Testar admissão, retomada após reinício, idempotência, autorização e projeção de uso ponta a ponta. Maestro permanece dono único de execução, memória e ledger; Nexus não duplica runtime.
3. Registrar SHA e hash do pacote Nexus e Maestro. Congelar os candidatos somente após fechar gaps; executar quality, build, contratos, evidence map e docs no SHA congelado.
4. Vincular evidências ao SHA exato e validar rollback. A tag `v1.0.0` antiga não representa automaticamente o candidato atual.

## Fase 4 — Plataformas, NEX e evidência de release

1. Validar Linux, Windows amd64 e macOS arm64 no SHA congelado, incluindo runner nativo quando exigido. Guardar logs e hashes reais; executar pareamento com dois dispositivos físicos, túnel estável, revogação, lease, handoff e reconexão.
2. Fechar os gates do [NEX release readiness](../../docs/design/nex/RELEASE_READINESS.json) segundo o identity lock. HUD e Site devem ser homologados em conjunto; não inserir NEX na página pública antes do pacote completo e aprovação visual do owner no SHA exato.
3. Usar a trust root do owner e o [runbook de evidências](../../docs/reference/RELEASE_EVIDENCE_RUNBOOK.md). Construir e assinar artefatos oficiais, validar instalação/atualização e rollback. Não inventar evidência visual ou física.

## Fase 5 — Decisão GO e distribuição

GO exige continuidade comprovada no cenário acordado, 48/48 capacidades com evidência/aceite aplicável, P0s fechados, quota/balance e integração Maestro aprovados, plataformas e túnel certificados, NEX 11/11 com aceite do owner, quality/CI verdes no SHA final e assinatura verificada. Executar `node scripts/verify-nex-release-readiness.mjs --require-ready`. Só então, com autorização explícita, publicar/taguear conforme estratégia aprovada, smoke de instalação/upgrade e registrar hashes/rollback em DEV. Até lá, manter **NO_GO**.

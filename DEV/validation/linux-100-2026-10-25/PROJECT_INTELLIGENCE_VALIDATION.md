# Project Intelligence e grounding do Composer — P4

Data: 2026-10-25  
Branch: `release/v1.0.0`  
HEAD-base consultado: `7a04f1b4cf5950b97b4ea1e156dda31ac513823b`  
Fingerprint efetivo do overlay: atualizado no manifesto `logs/source-provenance-p4-project-intelligence-2026-10-25.txt`.

## Resultado

A descoberta estática de stack, dependências, comandos, testes e CI já existia. Esta campanha fechou limites que faltavam no contexto entregue ao Composer: teto serializado de 16 KiB, no máximo 64 fatos, strings e citações limitadas, listas reduzidas, redaction reaplicada, data de observação em cada fato e identidade atual do código. O envelope não envia o caminho absoluto local nem warnings com caminhos do host.

O Composer recebeu o comando `vitest run` do `package.json`, preservou a citação `package.json:scripts.test`, nível `HIGH` e a data da observação. A meta vaga de autenticação manteve uma pergunta bloqueante material; um prompt de teste com escopo, critérios, verificação e exclusões não abriu perguntas bloqueantes. Os testes de symlink e alteração do arquivo confirmam que evidência fora da raiz não é lida e que o conteúdo novo recebe outra identidade.

O resultado global continua `NO_GO_LINUX`. O executável `ai-cli` não está instalado nesta máquina, então uma sessão real com esse provedor está `BLOCKED`; as E2Es usam servidor isolado do Nexus e dados fixture, sem credenciais ou envio de código a serviço externo. A primeira tentativa do spine P5 leu o estado antes de aguardar a resposta HTTP do `/finalize` e observou `READY_WITH_GAPS`. Adicionei sincronização explícita com a resposta do endpoint e uma regressão de armazenamento; a repetição passou e persistiu `FINALIZED`. A primeira falha permanece registrada como evidência da corrida no teste, não como defeito de produto.

## Matriz P4 e cruzamento P5

| ID | Procedimento | Resultado | Status | Evidência | Severidade |
|---|---|---|---|---|---|
| P4-001 | Serializar fixture de `package.json` com string UTF-8 de 80 KiB e chave OpenAI; invocar `BoundedProjectIntelligenceContext` | JSON final ≤16 KiB, marca truncamento e não expõe `sk-proj-` nem strings ilimitadas | PASS | `logs/go-test-p4-project-intelligence.log`; `internal/nexus/project_intelligence_context_test.go` | — |
| P4-002 | Criar sessão vaga em Chromium com `package.json` contendo `scripts.test=vitest run` | Brief mostra `vitest run`; contexto mantém fonte, locator, confiança `HIGH` e `observed_at` | PASS | `composer-project-intelligence/composer-project-intelligence-p4.json`; screenshot PNG correspondente; `logs/composer-project-intelligence-e2e-p4.log` | — |
| P4-003 | Alterar `package.json` depois da primeira leitura | `identity_digest` muda; fatos não são tratados como pertencentes à observação anterior | PASS | `logs/go-test-p4-project-intelligence.log`; `internal/nexus/project_intelligence_context_test.go` | — |
| P4-004 | Criar diretório symlink no projeto para pasta externa com `package.json` | Scanner não segue o link e nenhuma citação aponta para fora da raiz | PASS | `logs/go-test-p4-project-intelligence.log`; `internal/nexus/contextsnapshot/identity_discovery_test.go` | — |
| P4-005 | E2E: perguntar de forma vaga sobre autenticação com stack frontend observado | Responde com a evidência Vitest e mantém uma pergunta material | PASS | `composer-project-intelligence/composer-project-intelligence-p4.json`; screenshot PNG correspondente | — |
| P4-006 | E2E: iniciar sessão com prompt completo de teste, critérios, comando de validação e exclusões | Zero perguntas bloqueantes | PASS | `composer-project-intelligence/composer-project-intelligence-p4.json`; `logs/composer-project-intelligence-e2e-p4.log` | — |
| P4-007 | Executar pergunta por `ai-cli` real autenticado | Não executado: `ai-cli` não está instalado; não foi usada conta externa | BLOCKED | `logs/ai-cli-availability-p4.txt` | P1 |
| P5-001 | E2E amplo Composer → esclarecer → finalizar → WorkPlan → preview | Dois esclarecimentos; finalização observada após HTTP 201; WorkPlan com cinco etapas; preview não inicia Mission | PASS | `logs/composer-vague-spine-p5-repro.log`; `logs/go-test-p4-project-intelligence.log`; `web/scripts/e2e-composer-vague-execution-spine.mjs` | — |

## Implementação

- `BoundedProjectIntelligenceContext` limita o envelope inteiro a 16 KiB, fatos a 64, strings individuais a 512 bytes e arrays a 32 itens de 128 bytes cada.
- Conteúdo de texto é redigido novamente antes de integrar o envelope. Valores não reconhecidos são excluídos; valores e metadados cortados definem `truncated=true`.
- O envelope inclui `identity_digest`, estado/completude, scanner e data de observação; remove `canonical_path` e warnings do scanner do contexto do modelo.
- Cada fato inclui `observed_at`, fonte relativa e locator. A entrada `PROJECT_FACT` do Composer inclui a data no resumo e a confiança no campo próprio.
- A descoberta permanece estática, allowlisted e sem executar scripts de projeto. `filepath.WalkDir` não percorre links simbólicos externos; regressão coberta.
- A E2E específica cria um servidor Nexus isolado, fornece apenas `package.json` fixture e verifica o briefing e perguntas do Composer pela UI/API autenticadas.
- O spine P5 agora aguarda a resposta do endpoint de finalização antes de consultar o estado persistido; a regressão Go verifica `FINALIZED` no store.

## Reprodução

```bash
go test ./internal/nexus/contextsnapshot ./internal/nexus -count=1
go test -race ./internal/nexus/contextsnapshot ./internal/nexus -run 'Test(BoundedProjectIntelligenceContext|DiscoverDoesNotFollowSymlinkOutsideProjectRoot)' -count=3
go vet ./internal/nexus/contextsnapshot ./internal/nexus
bun run --cwd web test:e2e:project-intelligence
```

O teste amplo de finalização, que excede P4, pode ser reproduzido com:

```bash
bun run --cwd web test:e2e:composer-vague-spine
```

Primeira observação: falha em `clarified session must finalize normally` após leitura sem aguardar a resposta do endpoint. Reexecução instrumentada: PASS. Nenhuma tentativa de execução real do `ai-cli` está incluída na E2E fixture.

## Direção do mascote

A implementação mantém o contrato já confirmado: um único NEX. Não foi criado Companion 2D, segundo mascote ou seletor adicional.

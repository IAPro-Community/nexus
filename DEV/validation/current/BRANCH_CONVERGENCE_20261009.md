# Nexus 1.0 — auditoria de convergência de galhos (2026-10-09)

**Base imutável auditada:** `release/v1.0.0@8e9d70e6620788f1e87b1d1067267425eacced2a`  
**Destino de integração:** `release/v1.0.0`; **sem merge direto, tag ou publicação**.  
**Natureza:** comparação do grafo Git (`ahead/behind`), inventário recursivo de blobs e
inspeção dirigida de contratos/arquivos críticos. **Não** certifica equivalência
semântica de todos os arquivos divergentes, execução de CI nem homologação Windows.
O galho de auditoria contém somente uma correção independente de ResourcePicker
e este registro. Revalidar HEAD antes de eventual merge.

## Inventário

Consultados 75 galhos remotos: 55 `archive/` e 20 fora de `archive/`.
Entre os 20 há `main`, `release/v1.0.0`, `release/v1.1.0`, 14 outros
galhos divergentes e três galhos de trabalho integralmente ancestrais da 1.0.
A `main` também é ancestral da release. Nenhum PR estava aberto no início.
Não eliminar branches nem cherry-pickar árvores inteiras apenas pelo histórico.

### Já incorporados pelo histórico (ahead=0)

- `fix/beta7-windows-KIVER-20261007`
- `integration/v1.0.0-css-and-themes-20261009`
- `refactor/v1.0.0-workspace-css-modules-20261009`

**Ação:** candidatos a arquivamento depois de conferir referências externas e
trabalho humano em andamento; não exigem merge de código.

### Divergentes: decisão por domínio

| Galho | ahead/behind da base | Resultado e decisão |
| --- | ---: | --- |
| `audit/v1.0.0-beta9-maintainability-20261008` | 9/72 | Helper transacional de versão, testes e aviso de dry-run estão presentes na release; manter relatório como histórico, sem merge integral |
| `codex/windows-beta6-convergence` | 5/396 | Dois arquivos exclusivos de seleção manual de recurso e teste, além de helper de comando de teste Windows; modelo de seleção portado seletivamente neste galho de auditoria |
| `feat/v1.0.0-flat-quiet-themes-20261009` | 39/61 | Release já registra **20 temas**, incluindo presets flat, Nexus Dusk e Ember Studio; não importar ThemeProvider/CSS antigos |
| `fix/beta7-codex-terminal-protocol-p0` | 4/260 | Não há arquivos exclusivos; release tem filtros/fixtures de mouse, CSI, SS3 e ConPTY posteriores; **não** retroceder parser/PTY |
| `fix/beta7-terminal-review-trace-lifecycle` | 10/101 | Sem arquivos exclusivos; mudanças no ciclo de host/trace exigem comparação funcional antes de reaproveitar qualquer hunk |
| `fix/beta7-terminal-trace-diagnostics` | 24/259 | Sem arquivos exclusivos; a release evoluiu rastreamento mascarado e lifecycle; não copiar versões antigas de arquivos críticos |
| `fix/beta7-windows-autopilot-20261001` | 1/260 | Dois relatórios históricos de Windows, sem código exclusivo; preservar como evidência antiga |
| `fix/beta7-windows-homologation-20261008` | 3/95 | Documento beta.7 exclusivo, sem código exclusivo; não substituir status atual por evidência beta.7 |
| `fix/beta8-manual-opaque-quota` | 1/260 | Política Go distinta: trata `UNSUPPORTED` como manual-only. A release atual o considera operacionalmente elegível inclusive para AUTO, com evidência `OPAQUE`. **Decisão de produto/roteador exigida** antes de mudar o backend |
| `fix/v1.0.0-gap-closure-20260930` | 38/263 | Quatro fontes exclusivas: `continuity_controller.go`, `continuity_controller_test.go`, `execution_target.go`, `execution_target_test.go`. Revisar compatibilidade com o roteamento vigente; não transplantar state machines e handoff sem gates |
| `fix/windows-beta8-homologation-20261008` | 16/83 | Nenhum arquivo exclusivo; evidência beta.8 não certifica beta.10 |
| `fix/windows-beta10-homologation-20261008` | 158/61 | `internal/core/provider/versionprobe/versionprobe.go` e teste ainda exclusivos. O relatório do galho documenta PASS Windows **somente para o SHA e46d0619**, não para o HEAD da release; integração do helper requer revisão de todos os adaptadores e `doctor` |
| `integration/v1.0.0-beta10-pr45-pr46-20261008` | 81/67 | Nenhum arquivo exclusivo; contratos principais de ConPTY, helper de versão, temas, captura e testes já aparecem na release. Evitar merge integral, que pode substituir correções recentes |
| `test/beta7-terminal-final-boundary-regression` | 3/258 | Nenhum arquivo exclusivo; fixtures de boundary/trace presentes e ampliadas na release. Manter referência histórica, sem recópia automática |
| `release/v1.1.0` | 249/47 | **Outra linha de produto**: Computer Use, execution targets remotos, design mode. Não misturar na 1.0 |

Os valores `ahead/behind` descrevem divergência de commits, **não** número de
funcionalidades faltantes. Mesmo arquivos com SHA de blob diferente podem ter
mudanças concorrentes válidas em ambos os galhos.

## Bug verificado: escolha manual de quota UNSUPPORTED

- Backend: `internal/nexus/resource_discovery.go` valida seleção manual
  por `quota.EvaluateEligibility(accountEligibilityInput(account, true))`.
- Backend: `internal/core/quota/eligibility.go` deixa uma conta autenticada
  `UNSUPPORTED` operacionalmente elegível, apesar de `Available=false`,
  sem inventar saldo medido.
- Frontend anterior: `ResourcePicker.tsx` desabilitava qualquer conta com
  `!account.available`, mesmo quando o endpoint de seleção manual a aceita.
- Resolução proposta: reutilizar `resourceSelectionModel.ts` e seus testes do
  galho histórico, aplicando `canManuallySelectResourceAccount` **apenas**
  na habilitação da escolha explícita.
- Guardas continuam bloqueando: sem autenticação, network error, rate limit,
  health indisponível e motivo `EXHAUSTED`. `UNKNOWN` só pode ser escolhido
  quando a conta está disponível. O backend continua sendo autoridade final.
- **Não alterado:** quota automática, scheduler, score, roteamento multi-conta,
  provider adapter, Codex, ConPTY, tema, versão ou configuração de CI.

## Backlog de integração seletiva

1. **P0 — Congelar e homologar Codex beta.10:** CI same-SHA + smoke interativo
   de scroll/setas em Windows ConPTY e Linux PTY. Nenhuma correção antiga de
   terminal pode entrar enquanto isso não passar.
2. **P1 — Seleção manual `UNSUPPORTED`:** este draft PR, com `bun run
   quality:full` e testes de recurso/seleção antes de merge.
3. **P1 — Provider versionprobe:** avaliar como evitar classificar ferramentas
   instaladas com falha de `--version` como desinstaladas. Levar os testes para
   o contrato atual e conferir comportamento do `doctor`, mantendo saída
   capturada privada.
4. **P1 — ContinuityController e ModelCatalog:** comparar com o architecture
   target/resolver atual e com a linha 1.1; aceitar somente partes novas que
   preservem as invariantes de lease e não-dois-escritores.
5. **P1 — Política `UNSUPPORTED` em AUTO:** decidir explicitamente se pode ser
   selecionada no modo automático. Não mudar de política por cherry-pick.
6. **P2 — Manutenção:** reduzir CSS/terminais extensos por patches graduais, e
   depois arquivar galhos históricos já absorvidos.

## Gates antes de integrar este PR

```bash
cd web
bun install --frozen-lockfile
bun run format:check
bun run lint
bun run typecheck
bun run test
bun run build
```

Rodar também `go test -count=1 ./internal/core/quota ./internal/nexus`
e o teste manual do ResourcePicker com provider de quota opaca.
Exigir CI exato da cabeça do PR/resultado de merge e homologação específica
da beta.10 antes de considerar tag ou distribuição. **Nenhum desses gates foi
executado nesta auditoria via conector GitHub.**

Não apagar os galhos divergentes, não fundir a 1.1 com a 1.0 e não atribuir
ao SHA atual as evidências Windows `e46d0619` ou beta.7 anteriores.

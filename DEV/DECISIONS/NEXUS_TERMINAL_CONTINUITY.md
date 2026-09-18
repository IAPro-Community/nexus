# Sessão de decisões — Terminal, quota e continuidade

Data: 2026-09-07  
Contrato: `DEV/SPECS/NEXUS_TERMINAL_CONTINUITY_LUNA.md`  
Base local: `1899ca6334576d859056d48a394e51d03758f313`

Esta sessão registra decisões tomadas durante a execução do contrato Luna. Ela
separa o que foi implementado e verificado localmente do que ainda depende de
runner nativo, credencial, CI remoto ou integração externa.

## Decisões aceitas

### Supervisão não implica exclusividade de perfil ou Agent

Uma conta/perfil representa credenciais e configuração compartilhadas. Cada
invocação supervisionada recebe runtime, terminal, endpoint, generation,
provider session e Lead independentes, inclusive quando usa o mesmo perfil.
TUIs Codex mantêm leases compartilhados; apenas preparação/migração e probes
`app-server` exigem lock exclusivo. O estado de uma execução não pode encerrar
ou reconfigurar outra.

### Quota não observada nunca é saldo

`UNKNOWN`, `UNSUPPORTED`, `ERROR` e observação vencida não são convertidos em
0%, 100% ou verde. Verde é reservado a uma observação `LIVE` com janela e
percentual válidos. `CACHED`/`ESTIMATED` permanecem explicitamente degradados.
Isso também explica por que uma conta Codex sem observação confirmada não gera
alerta: emitir alerta de quota nesse caso seria fabricar evidência.

### Alertas usam episódio, não texto

O alerta é identificado por provider, domínio de quota, conta/pool, grupo,
janela, limiar e episódio. O lease interprocesso do monitor impede três
processos Nexus de coletarem e notificarem o mesmo evento. Uma queda de quota
para um limiar mais grave pode gerar um novo episódio; polls repetidos não.

### Conta agrupada, credencial separada

Web/Desktop exibem um grupo por provider e deixam os perfis expandidos para
diagnóstico e seleção manual. Agrupar visualmente não soma percentuais nem
mistura credenciais. A identidade de quota continua sendo resolvida pelo
backend; e-mail isolado não prova que duas contas têm limite independente.

### Prioridade é política configurável

O default é Codex (1), AGY (2), OpenCode (3), mas a seleção não trata isso como
capacidade. Um provider com quota confirmada pode vencer um provider prioritário
sem quota; `UNKNOWN` não é verde. Troca automática entre providers continua
dependendo da política persistida e da transação de handoff, não de `--yolo`.

### Supervisão não é padrão ainda

`nexus <provider> --supervised` usa o SessionHost e torna os controles Nexus
disponíveis; o caminho direto permanece compatibilidade até haver evidência
nativa de paridade em Linux, macOS e Windows. `--yolo` autoriza a execução do
provider, mas não concede autorização implícita para trocar provider, conta,
modelo ou executar fallback.

### Continuidade precisa ser honesta

O contrato distingue sessão lógica, geração de processo, checkpoint salvo,
checkpoint entregue e reidratação confirmada. Trocar o rótulo de provider não é
continuidade. Se o destino não confirmar contexto e autoridade de escrita, o
estado deve permanecer recuperável/pendente; não pode ser anunciado como
`LIVE`.

## Decisões rejeitadas

- `time.Sleep`/retry fixo como correção de readiness de named pipe.
- `UNKNOWN → LIVE` ou `quota_remaining` sem fonte reconhecida.
- Somar duas contas em “160% de quota”.
- Fallback automático acionado por qualquer exit code diferente de zero.
- Alterar binários dos providers para injetar slash commands.
- Declarar macOS/Windows verdes com cross-build, Wine ou logs históricos.
- Tornar o modo supervisionado default antes da matriz nativa.

## Evidência desta sessão

### Catálogo Maestro: contagem confiável e contexto executável

O catálogo da tela Maestro mescla o `.orquestrador` global com o catálogo do
perfil ativo, deduplicando por `skill.id`; metadados do perfil ativo têm
precedência para a mesma identidade. Na validação local, a composição passou
de 48 para 53 skills e a UI exibiu `53 de 53 disponíveis`.

O antigo “Copiar comando” foi substituído por “Copiar contexto de uso”. Cada
card expõe o conteúdo integral de `skills/<id>/SKILL.md` em “Como usar esta
skill” e copia esse mesmo contexto. Se o arquivo não existir, há um fallback
explícito baseado em nome, descrição, triggers e requisitos de verificação.

Assim, a contagem vem do catálogo mesclado, a identidade é deduplicada por ID
e o prompt mostrado/copiado mantém a mesma origem usada pelo runtime. A
regressão cobre merge, ordenação e leitura do `SKILL.md`, com smoke real no
servidor.

### Tipografia e acessibilidade visual

A auditoria Open Design identificou que a escala compartilhada começava em
10–11px e que o preset `nexus-dark` sobrescrevia `--nx-subtle` com uma cor
abaixo do contraste AA em elementos reais do shell. A escala semântica foi
elevada de forma incremental (2xs 11px, xs 12px, sm 13px, base 14px) e o
subtle dark foi alinhado na fonte de verdade do tema e no CSS base. Controles
de filtro e cópia do Maestro passaram a ter 30px de altura mínima.

O verificador visual agora usa um `BrowserContext` real, coleta screenshots em
320×568, 390×844, 768×1024, 1024×768 e 1440×900, verifica overflow e executa
Axe. Após a correção, os cinco viewports passaram sem violações sérias.

Os tokens tipográficos agora usam `rem`, com `1rem = 14px` na escala Nexus
normal. O `html` aplica a escala do usuário uma única vez; isso mantém zoom do
navegador, preferências de acessibilidade e densidade compacta/comfortable
coerentes sem misturar `px` fixos com tokens escaláveis.

Ajuste de leitura é uma preferência frequente e foi promovido ao toolbar:
`FontScalePicker` oferece 90%, 100%, 110% e 120% sem exigir navegação até
Configurações. Configurações continua sendo o local para a descrição detalhada
de acessibilidade e densidade; o toolbar é o atalho operacional.

Uma segunda auditoria encontrou que tokens em `rem` não bastavam: o shell,
terminais, modais, Composer e Plan Builder ainda possuíam fontes literais em
pixels. Essas declarações, incluindo estilos inline estáticos, foram migradas
para `rem`; a busca final não encontra mais `font-size`/`font`/`fontSize` literal
em `px` no código de `web/src`.

### Direção Open Design do Maestro

A tela adotou a direção **Tech Utility**: catálogo operacional, alta densidade
legível e status acionável. A intervenção foi `critique → polish → adapt`:
hierarquia do hero reduzida, catálogo com título/contagem próprios, busca como
controle semântico, cards resistentes a nomes longos, foco visível, estados de
loading/erro/vazio e breakpoints para 320–1440px. Gradientes decorativos,
glows, card nesting excessivo e texto inline foram evitados.

- Web: `MaestroSurface.tsx` e `MaestroSurface.module.scss`.
- Validação: typecheck, ESLint, Stylelint, testes direcionados e build passaram.
- Risco restante: a captura anexada é uma referência visual; screenshots reais
  nos cinco viewports ainda devem ser coletados pelo browser/E2E do ambiente.

- `make quality` terminou com `make_quality_exit=0` após os ajustes de lint e
  formatação.
- `go test -race -count=1 ./...`, `go vet ./...` e `cd web && bun run verify`
  passaram na execução desta sessão antes da alteração visual final.
- TypeScript e os testes focados do modelo de quota passaram após a alteração:
  `bun run typecheck` e `bun run test --run src/features/work/directSessionModel.test.ts`.
- A auditoria local de providers encontrou Codex e AGY autenticados, mas sem
  observação numérica atual (`UNKNOWN`/sem `fetched_at`); OpenCode instalado sem
  autenticação e Gemini instalado sem profiles. Isso é diagnóstico, não prova de
  saldo.

## Pendências que bloqueiam promoção oficial

1. Reexecutar a matriz nativa Windows/macOS, incluindo ConPTY, named pipe,
   resize, cleanup e worktree identity.
2. Integrar o agrupamento de pools ao contrato backend e concluir handoff
   transacional/reconexão antes de ligar fallback interativo automático.
3. Executar o CI remoto no mesmo SHA; cross-compilação Linux não substitui esses
   gates.
4. Só depois configurar branch protection/required checks com autorização
   administrativa e preparar tag.

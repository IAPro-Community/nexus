# Product Discovery — validação estática e pendências (27/10/2026)

**Branch:** `release/v1.0.0`  
**Status:** Código integrado / inspeção estática PASS parcial / **RUN & RELEASE NO_GO**.

## Escopo implementado

- `featureAnnouncements.ts`: catálogo versionado de novidades por
  id/revision, rota Nexus e seletor `data-tour`, com foco no recurso real.
- `ProductUpdatesCenter.tsx`: menu global Novidades do Nexus, contagem de
  não lidas, navegação à área e dispensar todas.
- `FeatureTour.tsx`: menu contextual por área, aviso pequeno descartável,
  ação Mostrar no componente, progresso existente e leitura persistente.
- `FeatureHint.tsx`: ajuda focada em controles específicos via Radix Tooltip.
- `ProductTour.tsx`: renderer e passos existentes continuam sendo a fonte
  única de tour. **Não foi incluída segunda biblioteca de tour.**
- `NexusShell.tsx`/`NexusWorkspaceApp.tsx`: menu global vinculado à flag
  `COMPOSER_FLOW_ENABLED`, navegação real em `work`/`missions`.
- `featureAnnouncements.test.ts`: testes de catálogo, versão, alvo, rota,
  traduções e vínculo ao tour.
- `web/src/i18n/resources.ts`: PT-BR/EN/ES e ajuda por componente.
- Documentação e template de PR exigem que novos recursos sejam registrados
  no catálogo no mesmo PR, com regra para novas revisões e permissões.

## Resultados observados por leitura direta dos arquivos do GitHub

- Foram encontrados **6 anúncios** no catálogo, cada um com
  `openSurface` e `data-tour` existente nas superfícies de Composer/Flow.
- Foram identificadas **3 seções de tradução discovery** e todas as
  12 referências de chave de título/descrição do catálogo aparecem nas três.
- `web/bun.lock` permaneceu inalterado.
- `NexusShell` monta o menu global e `FeatureTour` usa os mesmos IDs de
  produto com marcação de leitura local sincronizada.
- Nenhuma informação sensível, prompt ou token vai para um servidor de
  novidades; registros vistos são revisões numéricas em `localStorage`.
- O renderer visual ainda é `ProductTour` próprio. A substituição por
  biblioteca mantida por terceiros está registrada em
  [#103](https://github.com/IAPro-Community/nexus/issues/103).

## Limites importantes

**Os avisos não são gerados magicamente pela existência de uma função.**
Um desenvolvedor precisa registrar o recurso em `featureAnnouncements.ts`
durante seu PR. Mudanças só são visíveis aos usuários após distribuir uma
build que as contenha. A lista global pode exibir recursos compilados mesmo
sem uma atualização remota; o flag desativado impede apresentação da área
experimental.

A leitura vista é por navegador; limpeza do armazenamento ou outro device
pode reapresentar a novidade. Não há sincronização entre pessoas/dispositivos.
Novidades nunca devem competir com o Centro de Atenção por alertas
operacionais, nem bloquear sessões.

## Homologação não realizada

- [ ] Bun `install --frozen-lockfile`, lint, typecheck, format, estilo,
      Vitest e build do SHA final.
- [ ] Playwright no DOM real: navegação do global à área, popover Radix,
      selecionar novidade e iniciar tour no alvo correto, mesmo fora do
      viewport.
- [ ] Marcar como lido em área e observar contador global desaparecer;
      reiniciar app e confirmar persistência; simular nova revisão e conferir
      novo aviso uma única vez.
- [ ] LocalStorage desabilitado, troca de projeto, flag experimental OFF,
      sessão Composer offline, Flow manual e Mission em execução.
- [ ] Teclado, Esc, foco, mobile, zoom 200%, alto contraste, reduced-motion
      e PT-BR/EN/ES.
- [ ] Desktop Wails Linux/macOS/Windows e build realmente instalada, com
      hash/manifest correspondente ao commit.

## Critério de pronto

O código de novidades só poderá ser anunciado como homologado se a issue
[#102](https://github.com/IAPro-Community/nexus/issues/102) contiver evidência
de testes reais e visual QA do mesmo SHA. A estratégia de aproveitar Driver.js,
React Joyride ou FeatureDrop, com o filtro de licença/maturidade, está descrita
em [docs/product/feature-discovery.md](../../docs/product/feature-discovery.md).

# Nexus Product Discovery — tours, ajuda contextual e novidades

**Status de código em 27/10/2026:** implementação integrada em
`release/v1.0.0`, sem certificação de build nem evidência visual do
aplicativo distribuído. **Não afirmar homologação** por haver componentes no
repositório. Este documento não modifica a licença vigente do projeto.

## Necessidade atendida

A experiência anterior dependia de um tour manual fixo. O usuário podia
não descobrir novas funções instaladas, e os mantenedores tinham de adicionar
novos avisos e pontos de entrada separadamente em cada área.

O Nexus agora tem **um catálogo declarativo de recursos** para compor:

1. **Novidades do Nexus (global):** ícone na barra superior com total de
   entradas não lidas. Ao abrir uma entrada, encaminha à área real da função.
2. **Novidades desta área (contextual):** ao abrir Composer ou Flow com novos
   recursos instalados, surge aviso pequeno e dispensável. O menu guarda as
   descrições com ações **Mostrar no componente**.
3. **Ajuda por componente:** `FeatureHint` envolve o `Tooltip` Radix já
   instalado e explica opções complexas ao focar/passar o ponteiro.
4. **Tour do módulo:** reaproveita `ProductTour` e as etapas tipadas por
   superfície para explicar a função in loco. Não cria planos ou executa
   comandos.

Recursos antes implementados que não estejam presentes/visíveis na tela
não são anunciados por aquela superfície. Um tour completo pode pular
etapas invisíveis e os avisos são sempre opt-in para navegar/abrir o tour.

## Adoção de componentes prontos

| Candidato | Licença verificada em fontes públicas | Aplicação | Decisão |
| --- | --- | --- | --- |
| **Radix Tooltip / DropdownMenu** | MIT nos componentes do ecossistema | Ajuda, menu acessível e navegação via teclado | **Reutilizado agora** (já está no lockfile) |
| **Sonner** | MIT | Avisos transitórios | Já está no Nexus; não usado para novidades para evitar competir com o Centro de Atenção |
| **Driver.js** | MIT | Spotlight, tours com foco, destaque e espera por elementos assíncronos | **Candidato prioritário** a substituir a implementação visual de `ProductTour` em PR isolado |
| **React Joyride v3** | MIT | Tours React 19 e componentes personalizados | Alternativa viável; verificar conflitos de focus/portal/desktop |
| **FeatureDrop** | MIT no projeto consultado | Changelog, badges, checklists e anúncios, sem servidor obrigatório | Referência de design. Avaliar maturidade/segurança antes de incluir nova dependência |
| **Tour Kit** | BSL-1.1 no projeto atual | Tour/headless/adoption | Não selecionar automaticamente como dependência open source por ser source-available |
| **Intro.js** | AGPL/termos comerciais | Tours antigos | Licença a rever cuidadosamente, não necessário para a Community |

Fontes:
- https://www.radix-ui.com/primitives/docs/components/tooltip
- https://driverjs.com/
- https://react-joyride.com/docs/new-in-v3
- https://github.com/glincker/featuredrop
- https://github.com/domidex01/tour-kit
- https://introjs.com/

**Por que não instalar todas?** O Nexus já contém Radix e um componente de
tour compartilhado. Instalar outra biblioteca de spotlight durante a release
sem atualizar/testar `web/bun.lock` e validar os efeitos na desktop shell
seria uma regressão potencial. Driver.js deve substituir a parte visual
existente, não ser somado como segundo motor ativo. Não baixar código de um
CDN em runtime: manter operação self-hosted/offline-first.

## Como anunciar uma nova funcionalidade

O desenvolvedor adiciona **uma entrada no mesmo PR da funcionalidade** em
`web/src/features/work/featureAnnouncements.ts`:

- `id`: identificador estável; não reutilizar id de feature removida;
- `revision`: inteiro crescente; se mudar substancialmente, incrementar;
- `surfaces`: onde a funcionalidade de fato existe;
- `openSurface`: rota de área suportada pelo Nexus, não inferida do nome;
- `target`: seletor semântico existente `data-tour="..."`, na interface;
- `title` e `summary`: chaves i18n em EN/PT-BR/ES;
- `tourStep`: id do passo local que explica a função, quando houver.

Adicionar/atualizar etapas em `featureTour.ts` **somente se necessário**.
Se o usuário já conheceu o recurso na revisão 1, não verá novamente o aviso
até que o mantenedor incremente `revision`. Como o catálogo é compilado na
build, **um commit remoto ou uma feature flag desligada não cria novidades na
instalação do usuário**. Não tentar deduzir lançamentos de diffs Git ou do
`CHANGELOG.md`: poderia publicar informação de recurso incompleto.

As chaves `iapro:nexus:feature-announcement:v1:<id>` são locais ao navegador
e contêm apenas a revisão vista, sem prompt, projeto, token ou dados
analíticos. A mesma feature não é anunciada repetidamente para cada projeto.
Limpar armazenamento local ou usar outro dispositivo pode reapresentá-la;
sincronização entre instalações só com consentimento explícito e desenho
futuro separado. Nenhuma telemetria ou serviço comercial é necessário.

## Regras de UX e segurança

- Não bloquear o primeiro uso nem abrir tours automaticamente sobre
  terminais ou uma Mission em andamento.
- Nunca tratar uma novidade de produto como alerta de segurança, erro ou
  ação necessária; essa responsabilidade continua no Centro de Atenção.
- No máximo um aviso contextual compacto visível por área; mais novidades
  ficam no menu. `Marcar todas como lidas` está disponível.
- Nunca executar ações enquanto o tour demonstra botões. Uma etapa aponta
  para o controle real, não clica nem dá GO.
- Reusar tokens de tema Nexus, design system, localizações e comportamento
  de teclado/ESC/reduced-motion; não impor UI remota.
- Para recurso futuro em módulo novo, incluir seu identificador de
  superfície e montar `FeatureTour`; **isso não é descoberta automática de
  todo JSX**.

## Testes de conformidade

```bash
cd web
bun install --frozen-lockfile
bunx vitest run src/features/work/featureAnnouncements.test.ts \
  src/features/work/featureTour.test.ts \
  src/features/work/featureTourAnchors.test.ts src/app/tour/tour.test.ts
bun run typecheck
bun run lint
bun run lint:styles
bun run format:check
bun run build
```

Testar manualmente: primeiro acesso, dispensa, navegação global para a
superfície, atualização de `revision`, sincronização de badges sem reload,
múltiplos projetos, armazenamento bloqueado, Zoom 200%, teclado, temas e
estados sem elementos visíveis. Validar a **build do SHA exato**, não apenas
o código da branch.

Veja [tours do Composer e Flow](composer-flow-tour.md) e
[gate de visual QA](../../DEV/validation/COMPOSER_FLOW_GUIDED_UX_2026-10-27.md).

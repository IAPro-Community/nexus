# Nexus 1.0 beta.10 — Review de frontend e temas flat

Data: 2026-10-09.
Base auditada: `release/v1.0.0` em `a6795704e0f74971f6b255e66ea1527a3ae83165`.
Implementação separada: `feat/v1.0.0-flat-quiet-themes-20261009`.
Este documento descreve revisão **direcionada**, não certificação global do repositório.

## Diagnóstico

**Não é necessário reescrever o design system.** A base existente já possui Radix UI, conceitos Nexus, presets tipados, `ThemeProvider`, tokens semânticos, persistência em `iapro:nexus:theme:v1`, contrastes WCAG e testes browser. Existiam **15** presets, incluindo Monokai Dark e Obsidian; preservá-los por ID evita invalidar preferências.

Dívidas que merecem refatoração incremental, em ordem:
1. **P1 — CSS monolítico:** `web/src/app/workspace-os.css` tinha aproximadamente 8.250 linhas; mistura raiz/tokens, workspace, janela, rail, diálogos, estados e splash. Migrar seção a seção para módulos por feature, com linha-base de CSS não permitido e snapshot visual por superfície. Evitar split cego que altere cascata.
2. **P1 — Responsabilidades grandes:** `AgentTerminal.tsx`, `WorkspaceRenderer.tsx` e `NexusWorkspaceApp.tsx` concentravam muitos fluxos. Extrair modelos puros e hooks testáveis por comportamento; não criar outro terminal ou outro frontend.
3. **P1 — Gates de qualidade:** o ESLint ainda aceita `any` e faz alguns problemas de hooks/variáveis apenas warnings. Ativar restrições por diretório e orçamento de violações, antes de impor no repositório inteiro. A cobertura frontend quantitativa não estava certificada na auditoria anterior.
4. **P1 — Experiência visual:** efeitos globais (gradientes, sombras fortes, halos) contribuem para ruído, mesmo com paletas boas. Nunca apagar estados de foco, contraste, error/warning, status do provider ou indicadores de tarefa.
5. **P0 release — Evidência:** a validação Windows em andamento está associada ao SHA congelado daquela execução. Qualquer merge na release alteraria o HEAD e exigiria execução de certificação novamente.

## Correção implementada sem alteração do runtime

- Registrados três novos presets compatíveis: `slate-flat`, `paper-flat`, `clay-flat`. A seleção é feita pelo mesmo `ChoiceGroup` acessível, na categoria `Essenciais Flat`.
- `ThemeDefinition.finish?: 'flat'` e `getThemePresetFinish` tipado; `ThemeProvider` publica `data-finish=flat|standard`; nenhum componente precisa conhecer IDs de temas. `high-contrast` continua no finish padrão.
- CSS flat compartilhado suprime gradientes de rail/workspace/cards/canvas, reduz sombras da grade/janela/splash e remove glow do splash. Preserva foco de teclado, estados operacionais, títulos customizáveis e cores semânticas.
- Preservadas sem modificação as 15 paletas anteriores, default, storage key, density e paleta própria de xterm.
- Novos testes no `theme.test.ts` verificam registro, persistência, compatibilidade e passam a incluir os três novos presets nos testes de contraste; `e2e-composer-themes.mjs` verifica a seleção, `data-finish`, recarga, responsividade e Axe como nos demais presets.

## Verificações pendentes

- `cd web && bun install --frozen-lockfile && bun run quality:full`.
- `cd web && bun run test:e2e:composer-themes` com Playwright/Chromium e evidência visual.
- Revisão visual de janelas sobrepostas, temas claro/escuro, estados de pergunta/erro e contrastes de texto no Windows/WebView2.
- Não mover esta branch para a release enquanto houver homologação Windows exata em andamento; integrar depois, com autorização de qualidade para o novo SHA.
- Os requisitos de distribuição/assinatura, Codex/ConPTY e smoke Windows/Linux continuam totalmente independentes da proposta de UI.

## Complemento — Nexus Dusk (2026-10-09)

O código de referência [stablyai/orca — main.css](https://github.com/stablyai/orca/blob/main/src/renderer/src/assets/main.css) foi inspecionado: `.dark` usa `#0a0a0a` no background, `#171717` no card/sidebar, `#262626` nos elementos secondary/muted e bordas `rgb(255 255 255 / 0.07)`. A hierarquia vem principalmente de camadas neutras e variação pequena de brilho; **não foi copiado um tema de terceiros**.

- Novo tema **Nexus Dusk** (`nexus-dusk`): grafite escuro (`#0b0c0e`), painel (`#17191c`), acento azul dessaturado (`#91a8d3`).
- Ampliado o contrato de `finish` com `subtle-gradient`, usando os mesmos tokens `--nx-*`, provider, persistência e seletor acessível. Os 15 temas originais e os três flat existentes não mudam.
- `data-finish="subtle-gradient"` aplica fundos com transição discreta em rail/workspace, canvas e chrome de janelas; sem neon e sem mudar o canvas ANSI do terminal ou a lógica de títulos customizáveis.
- A categoria **Degradê Suave** inclui apenas este tema.
- Matriz unitária ampliada a **19 presets**; E2E de Composer cobre seletor, reload, quatro tamanhos de tela/estados já configurados e Axe.
- Checagem estática das cores Nexus Dusk (10 pares): contraste mínimo observado **7,34:1** nas comparações de textos e acento auditadas, acima das metas 4,5:1 para texto e 3:1 para elementos acentuados. Não equivale à execução real dos testes.
- Ainda precisa passar por `bun run quality:full`, E2E real e conferência visual WebView2. Não converter esta PR draft em release antes da homologação e revalidação do SHA final.

## Próxima refatoração prioritária

Extrair seletivamente os estilos de janelas e rails de `workspace-os.css` para módulos locais, mantendo tokens globais centralizados; adicionar regressões visuais que comparem todos os temas escolhidos sem variações acidentais em estado de foco e leitura de terminal.

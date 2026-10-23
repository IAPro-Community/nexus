# IAPro Nexus — Identidade Visual, Assets e Diretrizes de Marca

Este documento consolida a especificação, catálogo de assets, ferramentas de geração e a proposta arquitetural para evolução de identidade visual e substituição de elementos genéricos no IAPro Nexus.

---

## 1. Catálogo Canônico de Assets

Todos os assets foram derivados do master oficial (`assets/brand/source-logo.png`) através de extração com preservação subpixel de antialiasing, un-matting e recorte adaptativo:

| Arquivo | Localização | Dimensões | Uso Principal |
| :--- | :--- | :--- | :--- |
| `nexus-icon.png` | Raiz, `web/public/`, `assets/brand/` | 512x512 | Ícone mestre de alta resolução com fundo transparente |
| `nexus-icon.svg` | Raiz, `web/public/`, `assets/brand/` | Vetorial | Ícone escalável em SVG para interfaces e renderizações nítidas |
| `favicon.ico` | Raiz, `web/public/`, `dist/` | 16, 32, 48, 64px | Favicon canônico para navegadores e SOs |
| `nexus-icon-32.png` | `web/public/`, `dist/` | 32x32 | Favicon PNG moderno |
| `nexus-icon-16.png` | `web/public/`, `dist/` | 16x16 | Favicon PNG compacto |
| `apple-touch-icon.png` | `web/public/`, `dist/` | 180x180 | Ícone de tela de início para iOS / iPadOS / macOS |
| `nexus-icon-192.png` | `web/public/`, `dist/` | 192x192 | Ícone PWA padrão Android / Chrome |
| `nexus-icon-512.png` | `web/public/`, `dist/` | 512x512 | Ícone PWA splash / desktop |
| `nexus-logo.png` | Raiz, `web/public/`, `assets/brand/` | 1050x236 | Logo horizontal completo com tipografia em azul marinho escuro (Light Mode) |
| `nexus-logo-dark.png`| Raiz, `web/public/`, `assets/brand/` | 1050x236 | Logo horizontal completo com tipografia em branco puro e corte do 'X' em azul (Dark Mode) |
| `logo.png` | Raiz, `web/public/`, `dist/` | 1050x236 | Alias canônico para compatibilidade com links existentes |
| `nexus-social-card-dark.png` | `assets/brand/` | 1200x630 | Cartão OpenGraph / Social preview em fundo escuro (`#080a0f`) |
| `nexus-social-card-light.png` | `assets/brand/` | 1200x630 | Cartão social em fundo claro |
| `manifest.webmanifest` | `web/public/`, `dist/` | JSON | Manifesto PWA para instalação como aplicativo Desktop nativo |
| `nexus.ico` | `assets/brand/` | 16, 32, 48, 64, 128, 256px | Ícone de aplicativo Windows (recursos PE e atalhos), derivado da suíte `web/public/nexus-icon-*.png` |
| `rsrc_windows_amd64.syso` | `cmd/nexus`, `cmd/nexus-desktop`, `cmd/nexus-hud` | Recurso PE | Ícone embutido nos executáveis Windows, gerado com `go run github.com/akavel/rsrc@latest -ico assets/brand/nexus.ico -o <cmd>/rsrc_windows_amd64.syso` |

---

## 2. Paleta de Cores da Identidade

- **Terminal Background (Frente):** `#182234` (Azul escuro grafite/navy)
- **Prompt Glyphs (`>_`):** `#ffffff` (Branco puro)
- **Janela Intermediária & Trilha Central:** `#3b72e1` (Azul Nexus vibrante)
- **Aba Superior & Nó Superior:** `#70a4ea` / `#8cbbf4` (Azul celeste suave)
- **Nó Inferior:** `#182234` (Navy escuro)
- **Tipografia Dark Mode:** `#f8fafc` com o corte transversal do `X` em `#3b72e1`
- **Fundo do Workspace OS:** `#080a0f`

---

## 2.1 Mascote oficial: NEX (Pixel Art 32-bit)

Em 23/10/2026, a direção de produto oficializou o **NEX** como **único mascote central** do IAPro Nexus. Isso **não substitui** o logotipo/ícone institucional, os ícones de sistema, o favicon ou o tema visual do produto.

- Pose oficial: **frontal**, cabeça grande, visor navy arredondado, olhos ciano, duas aletas, acoplamentos circulares, corpo flutuante, núcleo de energia e quatro módulos orbitais com anel de luz azul.
- Implementação ativa no HUD: **NEX Compact** renderizado pelo arquivo `web/src/features/hud/NexPixelSprite.tsx`, SVG vetorial de 128x128 unidades pensado para uso pequeno na janela nativa.
- Reaproveitamento previsto: **NEX Standard** para produto/documentação e **NEX Hero** para site/marketing, ambos respeitando a mesma anatomia; não declarar esses dois exports como publicados enquanto não houver master correspondente.
- Estados atuais do HUD: `idle`, `working`, `warning`, `offline`, com animações apenas visuais e sem criar telemetria.
- Ilustrações geradas durante a concepção não são automaticamente sprite sheets editáveis. Arte oficial de produção tem fonte versionada e regras verificáveis.

Fonte normativa: [NEX — contrato visual oficial](../docs/design/NEX_32BIT_VISUAL_CONTRACT.md).

Referências desenhadas versionadas: [anatomia frontal](../docs/design/assets/nex/nex-character-reference.svg), [expressões](../docs/design/assets/nex/nex-expression-states.svg), [escala](../docs/design/assets/nex/nex-scales-layout.svg). Controle para outras IAs: [manifesto visual](../docs/design/NEX_VISUAL_MANIFEST.json), [guia de geração](../docs/design/NEX_AI_GENERATION_GUIDE.md), [checklist](../docs/design/NEX_VISUAL_REVIEW_CHECKLIST.md). As referências SVG são auxiliares e a prancha raster original de alta fidelidade permanece pendente de ingestão binária, conforme manifesto.
A aprovação da identidade **não** altera o status de homologação da release 1.0 ou das plataformas nativas.

---

## 3. Script Automatizado de Geração

Os assets são gerados de forma reproduzível via script Python:

```bash
python3 scripts/generate_brand_assets.py
```

O script realiza:
1. Leitura do master em alta resolução (`assets/brand/source-logo.png`).
2. Identificação conectada por flood-fill do fundo claro e desmatting (alpha recovery) das bordas.
3. Separação do ícone (`nexus-icon`) do lettering (`NEXUS`).
4. Geração das variantes Dark Mode e Light Mode com preservação do corte azul da letra 'X'.
5. Construção do ícone quadrado centrado com respiração de ~16% para favicons.
6. Exportação de todas as resoluções padrão (16, 32, 48, 64, 128, 180, 192, 256, 512).
7. Compilação do `favicon.ico` com múltiplos tamanhos embutidos.
8. Geração dos cartões sociais 1200x630.

---

## 4. Onde o Logo Já Foi Aplicado

1. **Documentação Principal (`README.md`, `README.en.md`, `README.es.md`):**
   - Atualizado para tag `<picture>` responsiva com suporte nativo a `prefers-color-scheme: dark`.
   - Usuários com tema escuro no GitHub visualizam o logo com tipografia branca/azul vibrante (`nexus-logo-dark.png`); em tema claro, visualizam a versão em azul marinho (`nexus-logo.png`).
2. **Favicon e Metadados Web (`web/index.html`):**
   - Inclusão de `favicon.ico`, `nexus-icon-32.png`, `nexus-icon-16.png`, `apple-touch-icon.png` e `manifest.webmanifest`.
3. **Barra Lateral de Navegação (`web/src/features/projects/ProjectRail.tsx`):**
   - Substituição do "N" genérico em texto pela tag `<img src="./nexus-icon.png" className="nx-brand-mark__img" />`.
4. **Hub de Projetos / Boas-vindas (`web/src/features/projects/ProjectHub.tsx`):**
   - O hero principal agora renderiza o ícone oficial em destaque `.nx-brand-mark--hero`.
5. **Aplicação Demo (`web/src/app/NexusDemoApp.tsx`):**
   - A demonstração interativa agora exibe o ícone oficial.
6. **Sidebar (`web/src/components/Sidebar.tsx`):**
   - Atualizado para o ícone oficial e texto corrigido para "IAPro NEXUS · Workspace OS".
7. **Notificações Push Desktop (`web/src/notifications/PushNotificationManager.ts`):**
   - Notificações do sistema operacional agora utilizam o ícone dedicado `/nexus-icon.png`.
8. **Script de Build e Embed Go (`web/scripts/build.mjs`):**
   - Sincronização de todos os assets públicos para `dist/` e embedding automático no binário Go.
9. **Bandeja do sistema e janelas nativas (`internal/desktop`, `cmd/nexus-desktop`, `cmd/nexus-hud`):**
   - O tray (Wails v3 `SystemTray`) usa a marca (`nexus-icon-32.png` embutido no pacote, `go:embed`) em vez do ícone genérico do toolkit; `NexusAppIcon()` (256px) alimenta o ícone de aplicação das janelas nativas.
10. **Executáveis Windows (`cmd/*/rsrc_windows_amd64.syso`):**
   - `assets/brand/nexus.ico` é embutido via `rsrc` nos binários `nexus`, `nexus-desktop` e `nexus-hud`, restaurando o ícone da marca no Explorador e nos atalhos (antes exibiam o ícone genérico).

---

## 5. Propostas de Locais para Adicionar ou Substituir Elementos Genéricos

### A. Terminal e CLI (Charm Bubble Tea / Lip Gloss)
1. **Banner ASCII Art no Comando `nexus` / `nexus web` / `nexus doctor`:**
   - Atualmente, os comandos de terminal exibem texto plano como `=== Nexus Control Center ===` ou `=== IAPro Nexus Diagnostics ===`.
   - **Proposta:** Adicionar um banner estilizado com Lip Gloss renderizando o bloco do terminal `>_` em ASCII colorido (azul vibrante e branco) com o subtítulo `IAPro Nexus · Workspace OS`.
2. **Comando `nexus --version`:**
   - Exibir o logo estilizado em terminal junto com versão, commit e informações do runtime.

### B. Interface Web (Workspace OS)
1. **Loading / Splash Screen de Inicialização (`nx-app-loading`):**
   - Atualmente há um spinner genérico quando a aplicação é carregada (`Starting IAPro Nexus...`).
   - **Proposta:** Substituir pelo ícone do Nexus com animação de pulsação suave (*breathing glow*) e transição suave ao carregar os projetos.
2. **Watermark nos Empty States:**
   - As superfícies de *Composer*, *Flow/Missions* e *Maestro* possuem estados vazios com ícones genéricos do Lucide (`Sparkles`, `FolderGit2`).
   - **Proposta:** Utilizar o ícone do Nexus como marca d'água estilizada com opacidade 8-12% no fundo dos cartões de empty state.
3. **Avatares de Projetos sem Git (`nx-project-avatar`):**
   - Atualmente usa apenas a primeira letra do diretório.
   - **Proposta:** Oferecer variantes do badge do Nexus com cores derivadas do hash do nome do projeto para pastas locais sem ícone definido.
4. **PWA Instalável e Barra de Janela no Desktop:**
   - Com o `manifest.webmanifest` agora presente, ativar prompt de instalação no cabeçalho ou nas configurações para transformar a aba do navegador em janela de desktop independente sem a barra de endereços do Chrome.

### C. Comunicação e Documentação
1. **Cabeçalhos de Guias e ADRs em `DEV/`:**
   - Incluir o selo oficial `IAPro Nexus` nos principais manuais técnicos e no `DEV/README.md`.
2. **OpenGraph e Social Previews:**
   - Apontar as tags `<meta property="og:image">` no GitHub e na documentação para `assets/brand/nexus-social-card-dark.png`.

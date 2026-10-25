# Follow-up PR #61 — busca do portal e contrato do mascote NEX

Data: 2026-10-25  
Branch local: `release/v1.0.0`  
HEAD local/remoto observado: `bf6adf0c80ac3531d7291844512a14a017ab0002`  
PR #61: aberta e Draft; PR #55: fechada.  
Worktree: modificado, sem commit, push, merge ou publicação.

Este follow-up valida o portal Docusaurus existente e registra a confirmação do produto sobre o mascote. Ele não substitui nem altera as contagens da matriz Linux principal e não muda o veredito `NO_GO_LINUX`.

## Contrato do mascote

O usuário confirmou que o Nexus deve ter **um único mascote NEX, bem acabado e funcional**. Não será criado nem integrado um segundo personagem ou modo Companion 2D. A inspeção da árvore e da captura nativa confirma um único caminho visual: `NexusHud` → `GuardianMascot` → `NexPixelSprite`. `MASCOT` é a apresentação do mesmo NEX; Barra e Bandeja são apresentações do HUD. `companionModel` calcula humor, atenção e adiamento para esse personagem e não contém outro renderer.

A PR #55 continua `CLOSED` no GitHub. A regressão de `GuardianMascot` confirma uma arte NEX por render; a regressão do seletor confirma que não existe opção Companion 2D.

## Falha reproduzida e causa

A busca abria, mas falhava ao converter sua resposta: o navegador solicitava `search-index.json` e recebia uma página HTML 404 (`Unexpected token '<'`). O smoke inicial reproduziu `404 !== 200`; evidência em `logs/pr61-docs-search-red.log`.

O preset Docusaurus publica a documentação em `routeBasePath: "/"`. O plugin `@easyops-cn/docusaurus-search-local` mantinha o default `docsRouteBasePath: ["docs"]`; como `indexPages` está desligado, nenhum documento da raiz entrava no índice e o arquivo JSON não era gerado. A configuração agora declara `docsRouteBasePath: ["/"]`, o que respeita a validação do plugin e aponta o índice para a rota raiz.

## Matriz deste follow-up

| ID         | Procedimento                                                         | Resultado esperado                                                       | Resultado observado                                                      | Status |
| ---------- | -------------------------------------------------------------------- | ------------------------------------------------------------------------ | ------------------------------------------------------------------------ | ------ |
| DOCS-001   | `npm run build --prefix docs-site`                                   | Build estático e índice de busca gerados                                 | PASS; `build/search-index.json` gerado com 2.442.219 bytes               | PASS   |
| DOCS-002   | Playwright: abrir busca, consultar `terminal` e selecionar Terminais | Índice JSON HTTP 200, resultado `/product/terminals`, sem erro de página | PASS; título “Terminais e diagnóstico de teclado”; captura registrada    | PASS   |
| DOCS-003   | `npm run check:docs --prefix docs-site`                              | Catálogo válido                                                          | PASS; 100 páginas                                                        | PASS   |
| DOCS-004   | `npm audit --audit-level=high --prefix docs-site`                    | Nenhuma vulnerabilidade reportada                                        | PASS; 0 vulnerabilidades                                                 | PASS   |
| DOCS-005   | Prettier e `node --check` nos arquivos alterados                     | Formato e sintaxe válidos                                                | PASS                                                                     | PASS   |
| DOCS-006   | `bun run quality:full` em `web/`                                     | Gatilhos frontend e build de produção aprovados                          | PASS; 105 arquivos e 660 testes Vitest, lint, estilos, typecheck e build | PASS   |
| MASCOT-001 | Seis suítes focadas do HUD/mascote                                   | Uma arte, estados e controles consistentes                               | PASS; 30/30 testes                                                       | PASS   |
| MASCOT-002 | Inspecionar captura do HUD nativo                                    | Um único NEX visível                                                     | PASS; XWayland mostra um NEX                                             | PASS   |

Resultado deste follow-up: **8 PASS / 0 FAIL / 0 BLOCKED / 0 NOT_RUN**. As limitações nativas da homologação Linux principal permanecem registradas na matriz principal.

Prettier e `node --check` passaram nos arquivos de configuração/teste e neste relatório. Uma verificação adicional sobre todos os documentos históricos `DEV/` apontou reflow pendente em arquivos extensos já modificados; eles não foram reformatados em massa para preservar o conteúdo e as alterações existentes.

## Evidências

- [Build final do portal](logs/pr61-docs-site-build-final.log)
- [Smoke Playwright final](logs/pr61-docs-search-final.log)
- [Reprodução 404 antes da correção](logs/pr61-docs-search-red.log)
- [Auditoria npm](logs/pr61-docs-site-audit-final.log)
- [Validação do catálogo de 100 páginas](logs/pr61-docs-site-catalog-check.log)
- [Qualidade frontend e build](logs/pr61-web-quality-full.log)
- [30 testes focados do mascote](logs/single-mascot-focused-tests-final.log)
- [Captura real da busca por Terminais](screenshots/docs-site-search-terminal.png), SHA-256 `0deddb33d68477513975e08b590a0d74801261ad0662dffce3fa2dde559d96ef`
- [Captura do HUD nativo com um NEX](screenshots/pr578-current-hud-native-xwayland.png)
- [Índice JSON gerado](logs/pr61-docs-site-artifact-sha256.txt), SHA-256 `15f450349ce496363356fa76b0acb4d211a37b79d99b981d68130cdd6ff1d8b3`
- Fingerprint do conjunto de arquivos de configuração/teste: `4be82ae6a584aa857e068a6377dfc7d132841e2419035a45408d759532d01139`; manifesto em `logs/pr61-docs-site-source-sha256.txt`.

## Reprodução

```bash
npm ci --prefix docs-site
npm run build --prefix docs-site
npm run serve --prefix docs-site -- --host 127.0.0.1 --port 3029
cd web
DOCS_SEARCH_SCREENSHOT=../DEV/validation/linux-100-2026-10-25/screenshots/docs-site-search-terminal.png \
  PLAYWRIGHT_EXECUTABLE_PATH=/usr/bin/google-chrome bun run test:e2e:docs-search
```

O portal local passou nesta configuração; a PR #61 continua Draft e nenhuma PR foi integrada remotamente. O status Linux da release continua `NO_GO_LINUX` por beta.10 sem distribuição estável e pelos gates nativos/runtime pendentes já descritos no relatório principal.

# Comparison Lab — evidência parcial

Data: 2026-11-09
Branch: `release/v1.0.0`
Status: **PARTIAL / NO_GO**

## Implementado

O Flow Studio permite selecionar duas execuções concluídas do mesmo projeto, plano, revisão congelada e base. O serviço confirma workspaces Nexus distintas, dentro do diretório gerenciado, e integrações vinculadas às execuções. O snapshot persistido reúne resultados de testes e verificação, diffs dos arquivos modificados, custos com proveniência e itens desconhecidos. A UI mostra lado a lado e permite registrar uma escolha humana única com justificativa. O fluxo não escreve no checkout, não aplica diffs e não faz merge.

A API e a persistência têm testes para escopo/proteção, idempotência da decisão, imutabilidade do snapshot, isolamento de workspace e ausência de alteração no checkout. A interface reutiliza os componentes existentes, possui traduções PT-BR/EN/ES e usa o Product Tour compartilhado.

## Lacunas

- O produto ainda não cria duas alternativas A/B a partir de um mesmo plano, nem as executa em paralelo. O comparador recebe duas execuções que já foram preparadas e concluídas.
- Testes e custos são evidências reportadas pelas execuções existentes; não existe instrumentação completa para provar comparabilidade de toda métrica de custo.
- Não foi homologado visualmente em aplicativo distribuído no SHA exato.
- O critério de alternativas concorrentes e a capability `comparison-lab` permanecem `partial`/`pending`; não contar como uma das 48 homologadas.

## Verificações executadas

- Testes focados Go para aplicação, store e rotas.
- Vitest focado para API, anúncios e âncoras de tour.
- TypeScript, ESLint e Stylelint nos arquivos alterados.
- `make quality-full`: PASS na árvore local, 131 arquivos Web/828 testes; Go completo e race; seis avisos ESLint preexistentes, sem erros; `govulncheck` sem vulnerabilidades.

PASS também em `make build`, validadores canônicos do contrato e catálogo, regressões do mapa (4/4) e `git diff --check`. Código commitado e enviado como `a02c298a`. `make docs-verify` segue BLOCKED: o guard visual relata que as capturas/manifesto vinculados a `7a26a298…` ficaram obsoletos após nove fontes de renderização alteradas. Falta browser E2E visual no SHA publicado; não se usaram capturas sintéticas para declarar homologação. Ver [discovery frontend](FRONTEND-DISCOVERY.json).

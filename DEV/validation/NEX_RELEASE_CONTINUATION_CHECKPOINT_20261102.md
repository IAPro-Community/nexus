# Checkpoint de continuação — NEX na release Windows

Data: 2026-11-02
Branch: release/v1.0.0
HEAD local: eaf77085cfb86fa7b8952057cce41ed5ecf9c717
HEAD remoto observado: f15481d26e93d2e1d6db5ab60d3b4367d08ef58a
Estado: três commits NEX locais, ainda não enviados à release remota. Worktree limpa. Manter a homologação pendente.

## Mudança de direção

A implementação foi transferida da branch documental/PR #189 para um clone limpo da branch canônica atual, porque a instrução do usuário exige que tudo permaneça em release/v1.0.0. A worktree anterior da release foi preservada intacta: estava em 19671d53, 923 commits atrás e continha alterações preexistentes. A PR #189 foi fechada sem merge. Seus commits continuam disponíveis como histórico, mas não são o destino da homologação.

O clone atual está em:
D:\Dev\nexus-release-homologation-20261102

O handoff #186 permanece na SHA 081fc5bc3e292fd9db1d186e83caebcf5dc6684a. Os commits de código foram cherry-picked linearmente e sem conflito sobre o HEAD remoto release f15481d2:

- 0ede68e12f74eaf67c2a22bf519a6f48c0c6ab58 — feat(nex): add layered rig and contextual motion
- de49f8b62991ca7f1f5dabe1d92767af4740fcf0 — fix(nex): gate celebrations and cool down greetings
- eaf77085cfb86fa7b8952057cce41ed5ecf9c717 — test(nexus): make Codex fixtures portable on Windows

## Ambiente e dependências

- Windows 10.0.26200, AMD64/x64
- Go 1.25.14; Bun 1.3.9; Node 24.21.0
- Dependências instaladas com bun install --frozen-lockfile no clone acima, sob D:. Instalação terminou com exit 0: 512 pacotes.
- Houve aviso de filesystem lento para D:\caches\npm. Nenhuma dependência foi instalada administrativamente; PATH persistente, perfis ou credenciais não foram alterados.
- node_modules está ignorado pelo Git; a worktree permaneceu limpa.

## Gates executados no HEAD eaf77085

- PASS: bun install --frozen-lockfile.
- PASS: bun run --cwd web check:styles.
- PASS: bun run --cwd web lint:styles.
- PASS: bun run --cwd web build.
- FAIL: bun run --cwd web format:check. Prettier listou GuardianMascot.module.scss, GuardianMascot.tsx, nexCompanionBehavior.test.ts e nexCompanionBehavior.ts.
- FAIL: bun run --cwd web test. 782/783 passaram; 1 falhou:
  GuardianMascot.test.tsx > NEX single Guardian mascot > keeps articulated fins and four independently anchored orbital modules.
  A asserção esperava data-motion="on"; a saída renderizada tinha mood="working", data-motion="off", data-nex-behavior="still". Investigar a origem do estado e corrigir no produto/teste sem enfraquecer a asserção.
- INTERRUPTED: lint e typecheck. Foram interrompidos após o usuário pausar a meta; seus logs contêm apenas o início, portanto não contam como PASS nem FAIL.
- NOT RUN neste SHA: suíte Go depois do cherry-pick; testes de integração WebView2/Wails, HUD nativo, site, matriz DPI e inspeção visual nativa.
- A suíte Go NEX/handoff/Desktop tinha passado antes da transferência, no clone anterior e no mesmo código de teste; essa evidência não substitui execução exata no HEAD atual.
- O gate release readiness segue bloqueado; não foi alterado.

## Logs existentes

Todos os logs brutos permanecem em:
D:\Dev\nexus-remediation\nexus-homologation-tools\evidence\nex\release-eaf77085\logs\

- build.log e build.log.exitcode
- format.log e format.log.exitcode
- lint.log
- stylelint.log e stylelint.log.exitcode
- styles.log e styles.log.exitcode
- typecheck.log
- vitest.log e vitest.log.exitcode

Logs Go e capturas do trabalho anterior:
D:\Dev\nexus-remediation\nexus-homologation-tools\evidence\nex\20261102\

O pacote sanitizado anterior é histórico e se refere ao SHA e branch anteriores; não apresentá-lo como evidência da release eaf77085.

## Próxima sequência

1. Corrigir a renderização/teste que faz o HUD aparecer com motion off no mood working, preservando o contrato de status.
2. Rodar Prettier nos quatro arquivos afetados; revisar o diff.
3. Reexecutar format:check, lint, Stylelint, check:styles, TypeScript, toda a suíte Vitest e build.
4. Executar Go suites relevantes no HEAD eaf77085 com -count=1.
5. Reexecutar preview browser e capturar visualmente o rig.
6. Homologar Wails/WebView2 nativo e a integração de produção, se as dependências e o ambiente permitirem.
7. Antes do push, verificar novamente que origin/release/v1.0.0 ainda é f15481d2. Só enviar commits da branch release, com worktree limpa e resultado documentado. Não criar tag nem publicar release.

Até os gates pendentes serem aprovados, o estado correto é NO_GO/INCONCLUSIVE, não homologado.

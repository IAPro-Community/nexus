# Homologação Windows beta.10 — 2026-10-09

| Campo | Valor |
| --- | --- |
| GIT_SHA | `638cfb3bbb7aa9a647d6802964e7e4ab794cc786` |
| EVIDENCE_COMMIT | será registrado no commit documental de acompanhamento |
| BRANCH | `fix/windows-beta10-homologation-20261008` |
| VERSION | `1.0.0-beta.10` |
| PLATFORM | Windows AMD64 |
| WORKTREE_FROZEN | YES — o gate iniciou com a árvore de código limpa neste SHA |

## Resultado

**PASS — homologação automatizada local Windows no SHA exato `638cfb3b`.** O comando `scripts/homologate-beta.ps1 -ExpectedSha 638cfb3bbb7aa9a647d6802964e7e4ab794cc786` terminou com `PASS automated Windows beta homologation`.

O remoto `origin/release/v1.0.0` estava em `deca290b8ad2479a1cfdcc7f7b6aebf8fc944b6d`, e esse commit é ancestral do SHA validado. A release remota ainda não aponta para o candidato homologado. Nenhum gate Linux foi executado nesta rodada.

Evidências locais: `DEV/validation/local-beta/1.0.0-beta.10/638cfb3bbb7aa9a647d6802964e7e4ab794cc786/windows-AMD64/` (diretório local ignorado pelo Git).

Ambiente do gate: Go 1.27.2, Bun 1.3.9 e Node 22.14.0 para Playwright. A execução fechou entre `2026-10-09T01:39:06Z` e `2026-10-09T01:51:48Z`.

## Gates aprovados

- Sintaxe PowerShell e documentação: 178 arquivos Markdown e 33 assets/arquivos requeridos.
- `bun install --cwd web --frozen-lockfile` e `bun run --cwd web quality:full`: 84 arquivos Vitest, 503 testes PASS.
- Formatação Go verificada com `gofmt -l` nos arquivos rastreados, `go vet ./...`, `golangci-lint` v2.14.0 (0 findings), `govulncheck` v1.7.0 (sem vulnerabilidades) e `go test -count=1 -v ./...`: 66 pacotes PASS, zero falhas e 18 skips de casos condicionados a plataforma ou permissão.
- Preflight de regressões provider/Mission/terminal, E2E de control/runtime e regressões Agent Interop/OpenCode: PASS.
- Oito suítes browser E2E, além dos gates Axe/acessibilidade e visual: PASS. A captura de erro “invalid origin” no Project Hub passou no viewport mínimo Windows; o placeholder do caminho segue o formato do sistema operacional.
- Builds Windows AMD64 da CLI, Desktop Wails v3 e HUD: PASS.

## Smokes nativos Desktop e HUD

Os smokes foram recompilados do mesmo SHA com IDs de instância de teste e diretórios temporários próprios.

- Desktop: PASS. Janela Wails `IAPro Nexus`, filho WebView2 `Chrome_WidgetWin_0` e perfil isolado permaneceram presentes por 11 segundos. SHA-256 do binário de smoke: `9e6be1b05452111fb1954250c305e6d52c6bba2ffacd9ce0dec05dda8cb6f032`.
- HUD: PASS. Processo permaneceu ativo por oito segundos e criou seu perfil WebView2 isolado. SHA-256 do binário de smoke: `41e1f0d0522a685c876b2530548ab907073c9d320d057378cf284e79226c28e1`.

Esses smokes confirmam inicialização e isolamento de perfil. Não certificam renderização visual da janela nem interação HUD show/hide/autostart. Os executáveis de smoke não são binários de distribuição.

## Providers e pendências Windows

`nexus doctor --json` classificou Gemini como `WARN` porque o wrapper local falha em `--version`; o detector não apresenta uma instalação sem versão verificável como `PASS`. O CLI Gemini do host está associado ao Node 18.12.1 do PATH padrão. Para o gate browser, Node 22.14.0 foi colocado à frente no PATH. Nenhum prompt de modelo foi enviado.

O teste manual anterior do TUI Codex com perfil sem autenticação terminou antes da navegação por setas; permanece **inconclusivo** e nenhum prompt foi enviado. As regressões automatizadas ConPTY/setas passaram no SHA validado.

Permanecem pendentes: navegação interativa ConPTY/Codex em sessão real, Mission com provider autenticado e validação de custo/isolamento, projeção OpenCode na UI, controles interativos HUD, Voice, captura visual Desktop, workflow Windows hosted no mesmo SHA, Authenticode, chave privada de atualização, build e smoke do instalador final. Este host não tem `gh`, `signtool.exe`, `makensis.exe` nem credenciais de assinatura/update configuradas.

**Veredito:** PASS para homologação automatizada local Windows do candidato beta.10; **NO-GO para tag, instalador assinado e distribuição** até a integração/revisão em `release/v1.0.0` e o fechamento dos gates operacionais pendentes.

## Auditoria de branches e vínculo com release

Consulta remota de 2026-10-09: `origin/release/v1.0.0=deca290b8ad2479a1cfdcc7f7b6aebf8fc944b6d`, `origin/fix/windows-beta10-homologation-20261008=e40791c01332cc6d0bc1ea39745e7064f581cf8d` e `origin/integration/v1.0.0-beta10-pr45-pr46-20261008=9aad7444afb7581fd1238541ee0c6909211a3e78`.

O candidato local inclui a ponta atual da release como ancestral (`0` commits exclusivos da release; `114` commits exclusivos do candidato no grafo). A branch beta.10 de homologação é a que deve ser vinculada à release por PR após este push: [comparar com release/v1.0.0](https://github.com/IAPro-Community/nexus/compare/release/v1.0.0...fix/windows-beta10-homologation-20261008?expand=1). Não fazer push direto para `release/v1.0.0`.

A branch `fix/beta7-windows-homologation-20261008` tem três commits exclusivos de histórico, mas o conteúdo de código do commit `9105853a` já está presente em forma equivalente na árvore beta.10 atual; o cherry-pick ficou vazio após preservar os testes mais recentes. Os outros dois commits são capturas/status beta.7 e não devem ser transplantados para os documentos beta.10. A branch beta.7 não precisa de PR de código separado.

A branch de integração beta.10 difere da árvore atual da release apenas em `DEV/validation/current/RELEASE_STATUS.md` e no harness `web/scripts/e2e-composer-vague-execution-spine.mjs`; não mesclar a branch inteira. A branch Windows beta.8 mantém `VERSION=1.0.0-beta.8` e é histórica. A branch `audit/v1.0.0-beta9-maintainability-20261008` tem commits exclusivos de auditoria/versionamento e uma árvore divergente da release, incluindo uma alteração de CI para separar Axe/visual. Não mesclar em bloco; qualquer cherry-pick deve ser avaliado em uma revisão de pipeline separada, fora deste candidato Windows.

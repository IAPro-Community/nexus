# Nexus 1.0.0-beta.10 — plano operacional para Codex local (Windows AMD64)

Data: 2026-10-16. Branch de integração única: \`release/v1.0.0\`.

## Contexto, referência e resultados legítimos

- Homologação reprovada: \`9d7530ea8f8bf8b834a8b3de3b751335d0011cb0\`.
- Pacote histórico e imutável: branch \`review/windows-homologation-9d7530e\`, commit \`e4e5e401b69e0f2358d024ee60f4715b9759d3ec\`.
- Correções remotas iniciais: \`16cc24ba\` (copylocks, ST1005, Windows Go, uninstall), \`3c5cc519\` (beta-channel installers), \`f6f4c4ec\` (Sass partials). Verificar HEAD remoto antes de agir.
- Fonte histórica dos FAIL: \`DEV/validation/local-beta/1.0.0-beta.10/9d7530ea8f8bf8b834a8b3de3b751335d0011cb0/windows-AMD64/\` **na branch de evidência**, não na branch canônica.
- O CI pode ficar queued/cancelled. Isso NÃO é PASS. Não publicar tag, release, instalador nem manifest público neste trabalho.

### Mandato

Você é o Codex rodando **no Windows AMD64 nativo**, autorizado a corrigir a \`release/v1.0.0\` diretamente em commits pequenos, somente após conferir que ela não avançou e que o checkout está limpo. Preserve funcionalidades, contratos de segurança e compatibilidade. Execute gates realmente, documente falhas e não marque como homologado um SHA anterior. Não criar outro galho. Não modificar a branch de evidências. Nunca usar \`git reset --hard\`, \`git clean\`, \`git stash\`, push --force, apagar arquivos de usuário, instalar sistema com privilégios ou mexer em perfis Codex/Claude reais.

## Fase 0 — Preparação e integridade

1. Usar clone específico de desenvolvimento/homologação, separado de \`C:\Dev\appce-monorepo\` e de instalações Nexus de produção. Não reaproveitar checkout com alterações não commitadas.
2. Em PowerShell 7, registrar:

\`\`\`powershell
git remote -v
git fetch origin release/v1.0.0
git switch release/v1.0.0
git pull --ff-only origin release/v1.0.0
git rev-parse HEAD
git status --porcelain
Get-Content VERSION
go version
bun --version
node --version
$PSVersionTable.PSVersion
\`\`\`

3. Se \`git status --porcelain\` retornar algo antes de começar, **PARAR**, inventariar, pedir a decisão ao usuário e preservar os arquivos. \`git pull --ff-only\` só deve ser executado se o checkout estava limpo e sem divergência local.
4. Registrar SHA inicial, SO/arquitetura, \`GOTOOLCHAIN\`, \`GOFLAGS\`, instalações de WebView2, Playwright, NSIS e signtool. Para a trilha Windows beta, usar Go **1.26.9** e Bun **1.3.9**. Preferir Go portátil no perfil do usuário quando não houver administrador, sem alterar o Go global. A versão Node do CI \`windows-beta10-homologation.yml\` é 22.14.0: conferir a compatibilidade quando a máquina usar Node 24.

## Fase 1 — Fechar P1 de Go, sem mudanças semânticas

1. Confirmar a correção já aplicada em \`internal/control/web/public_service_status_test.go\`: passar \`hits.Load()\` ao \`t.Fatalf\`, nunca a estrutura \`atomic.Int32\`.
2. Confirmar strings minúsculas em \`internal/nexus/externalusage/claude.go\`.
3. Usar o arquivo histórico \`gofmt-fail.txt\` para identificar os **17** arquivos que reprovaram, sem alterar arquivos Go não relacionados. Em cada um: \`gofmt -w caminho/do/arquivo.go\`. Produzir diff de formatação e verificar ausência de alteração funcional.
4. Executar:

\`\`\`powershell
go version
go vet ./...
if ($LASTEXITCODE -ne 0) { throw 'go vet FAILED' }
go test -count=1 ./...
if ($LASTEXITCODE -ne 0) { throw 'go test FAILED' }
go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.2 run --timeout=5m ./...
if ($LASTEXITCODE -ne 0) { throw 'golangci-lint FAILED' }
go run golang.org/x/vuln/cmd/govulncheck@v1.7.0 ./...
if ($LASTEXITCODE -ne 0) { throw 'govulncheck FAILED' }
\`\`\`

5. Repetir \`gofmt -l\` sobre arquivos Go **rastreados**; zero arquivo pendente. Não usar \`go fmt ./...\` sobre \`node_modules\` ou gerados. Analisar se testes comparam a grafia antiga das mensagens de erro e ajustar somente asserts comprovadamente afetados.
6. Se Go 1.26.9 for incompatível com configuração de dependência específica, reportar a falha e propor reconciliação CI/harness, não baixar requisitos do gate.

## Fase 2 — Fechar P1 de frontend e arquitetura de CSS

1. Reproduzir \`bun install --cwd web --frozen-lockfile\`; não modificar \`bun.lock\` para satisfazer instalação.
2. Executar \`bun run --cwd web format:check\`; nos 10 arquivos apontados pelo log original usar Prettier local (\`bun run --cwd web format\` somente após confirmar no diff o escopo, ou \`bun x prettier --write\` nos arquivos listados, executado de \`web/\`).
3. Executar \`bun run --cwd web lint:styles\`. Corrigir \`web/src/features/hud/_hudPlacement.scss:37\` respeitando a política \`custom-property-empty-line-before\`; evitar desligar a regra.
4. Revalidar o novo algoritmo de \`web/scripts/check-css-allowlist.js\`: os partials \`_hudCapsule.scss\`, \`_hudPopover.scss\`, \`_hudPlacement.scss\` só podem ser aceitos quando importados diretamente por \`NexusHud.module.scss\`. Criar teste de regressão positivo e negativo: inserção temporária de \`@use\` a um desses partials por um arquivo global permitido DEVE falhar. Remover SOMENTE o fixture criado pelo teste.
5. Rodar \`format:check\`, \`lint\`, \`lint:styles\`, \`check:styles\`, \`typecheck\`, \`test\`, \`build\`, \`verify\`, \`quality:full\`. Não interpretar 552 testes Vitest verdes como PASS agregado enquanto style/format/doc falharem.
6. Registrar capturas e validar o Nexus HUD compacto/expandido após alterações CSS. Sem alteração inesperada no tema default, cores ANSI, atalhos do terminal, contraste e persistência.

## Fase 3 — Integridade do instalador, desinstalador e canal beta

1. Auditar as mudanças de \`install.ps1\` e \`install.sh\`: \`1.0.0-beta.N\` usa \`channel-beta\`, estável usa \`v<versão>\`, e **o manifesto Ed25519 precisa validar antes de qualquer instalação**. \`manifest.version\` e \`manifest.channel\` devem coincidir exatamente com a versão solicitada.
2. Adicionar testes offline de resolução da tag/canal para \`v1.0.0-beta.10\`, \`1.0.0-beta.10\`, \`v1.0.0\`, \`latest\`, beta de versão anterior a pointer atual, manifesto com versão divergente, assinatura inválida, SHA256 errado e artefato ausente. Usar fixtures/mocks, **nunca chave privada de produção nem bypass da assinatura**.
3. Auditar \`uninstall.ps1\`: remover apenas arquivos Nexus conhecidos; diretório desconhecido, raiz de volume, perfil do usuário, junction/symlink, shortcut de terceiro, custom install sem marker e arquivos extras devem ser preservados. A instalação PowerShell escreve \`.nexus-managed\` e a NSIS escreve \`.nexus-nsis\`. Adicionar testes PowerShell de isolamento em diretório TEMP e mock de registro/shortcut sem apagar o PATH ou Desktop reais.
4. Considerar backward compatibility de instalações antigas: default legacy \`%LOCALAPPDATA%\Programs\IAPro Nexus\` é reconhecido quando existe \`nexus.exe\`, enquanto instalações customizadas antigas sem marker devem falhar com instrução de migração, não serem apagadas arbitrariamente.
5. Revisar a lógica NSIS em \`packaging/windows/nexus.nsi\`: \`AddToUserPath\` e \`un.RemoveFromUserPath\` devem comparar **segmentos completos delimitados por ponto-e-vírgula**, não substring de diretório. Cobrir PATH no início/meio/fim, prefixos semelhantes (\`Nexus\` vs \`Nexus2\`), entradas duplicadas, vazio, ausente, REG_SZ e REG_EXPAND_SZ.
6. Quando \`makensis.exe\` estiver disponível em modo portátil, compilar fixture isolado com payload sintético/real assinado conforme escopo. Testar install/uninstall com registro e Desktop isolados. Sem NSIS, resultado é \`BLOCKED_TOOL\` — jamais PASS.
7. O instalador deve proteger os dados de projeto, SQLite, sessões, credenciais e perfis, e a desinstalação deve ser idempotente.

## Fase 4 — Manifesto e documentação visual atualizados

1. Concluir e commitar as correções de código **antes** de capturar a documentação.
2. Usar checkout limpo no SHA de código congelado. Executar:

\`\`\`powershell
$sha = (git rev-parse HEAD).Trim()
.\scripts\refresh-beta-docs-windows.ps1 -ExpectedSha $sha
\`\`\`

3. Revisar as capturas sintéticas e o vídeo gerados sob \`docs/assets/\`; confirmar ausência de nomes, projetos, token de bootstrap e dados reais. Verificar VIS-001..VIS-011.
4. Commitar somente evidências geradas válidas. Não editar manualmente \`manifest.source_sha\`. Observar que o commit dos assets altera o SHA final; rodar os gates novamente no HEAD final.
5. \`bun scripts/docs-verify.mjs\` deve ter exit code zero. A execução de CI \`docs-capture-evidence\` deve gerar assets associados ao SHA do job e verificar origem.

## Fase 5 — Certificação automatizada do SHA final

Com checkout limpo, go1.26.9, bun1.3.9:

\`\`\`powershell
$sha = (git rev-parse HEAD).Trim()
.\scripts\homologate-beta.ps1 -ExpectedSha $sha
\`\`\`

Não utilizar \`-AllowDirty\` ou \`-SkipDesktop\` como certificação. Arquivar logs e JSON em \`DEV/validation/local-beta/<VERSION>/<SHA>/windows-AMD64/\`.

Executar \`node --test scripts/release-version.test.mjs\`; os E2E \`test:e2e:composer\`, \`test:e2e:surface\`, \`test:e2e:capacity\`, \`test:e2e:hud\`, \`test:e2e:composer-spine\`, \`test:e2e:composer-vague-spine\`, \`test:e2e:composer-themes\`, \`test:a11y\`, \`test:visual\`. Registrar comandos duplicados que invocam o mesmo harness sem contar como cobertura independente.

Conferir que os executáveis reportem \`version=1.0.0-beta.10\`, o **novo SHA** e plataforma \`windows/amd64\`.

## Fase 6 — Smokes Windows que não podem ser inferidos de unit tests

- **Desktop:** iniciar binário Wails de smoke com ID único, APPDATA/LOCALAPPDATA/NEXUS_DATA_DIR/AI_MANAGER_DATA_DIR isolados. Verificar janela visível, pixels não brancos, navegação, projeto, terminal, fechamento, processos órfãos, reinstância e WebView2.
- **HUD:** validar compacto/expandido, posições, bandeja, autohide, autostart em registro de teste, múltiplas contas, quota UNKNOWN vs EXHAUSTED, warning/critical, offline/loading, drag/teclado, DPI 100/125/150% e monitores, sempre que disponíveis. Não encerrar HUD pessoal preexistente.
- **ConPTY:** sessão temporária com Codex autenticado, 30 menus, scroll X10/SGR/URXVT, setas CSI/SS3, Enter, Esc, Ctrl+C, fragmentação, resize, detach/reconnect e JSON não-ANSI. \`doctor\` não equivale a smoke real.
- **Routing:** Direct/Lead/DELEGATE/MISSION; não promover sem consentimento; conta fixa/PIN, failover conforme políticas, quota LIVE/CACHED/UNKNOWN fiel. Não consumir API paga sem autorização do usuário.
- **Update:** A→B de fixtures assinadas com chave **de teste** em harness isolado; rejeitar assinatura inválida, downgrade, versões indevidas, release channel errado, traversal, package-managed self-update e atualização com trabalho ativo.
- **Install:** ciclo NSIS/CLI Desktop/HUD em VM/sandbox Windows, PATH e registro preservados, uninstall sem remover arquivos extras. Certificado Authenticode de produção permanece gate EXTERNAL; não produzir ou divulgar build público unsigned como final.

## Fase 7 — Linux, CI e distribuição

- Reexecutar Linux AMD64 **nativo com GUI** no mesmo SHA, \`scripts/homologate-beta.sh\`, Go 1.25.14 + Bun 1.3.9 conforme harness Linux vigente. WSL não é certificação nativa do Desktop Linux.
- Revisar divergência Windows Go 1.26.9 vs Linux Go 1.25.14: é contrato por plataforma; registrar explicitamente e justificar ou unificar após testes completos. Nunca afirmar ambos são a mesma toolchain.
- CI Actions do SHA final precisa concluir \`success\`; \`queued\`, \`cancelled\` e \`skipped\` não certificam.
- Verificar licenças/SBOM, assinaturas, provenance/source_sha, PKG/macOS se fizer parte da política estável, manifest e instalação do pacote final.
- Sem certificados/secrets, registrar \`BLOCKED_EXTERNAL_SIGNING\`; não forjar evidência nem substituir por autoassinatura.

## Fase 8 — Critérios de decisão, comunicação e commits

Cada gate: \`status: PASS|FAIL|BLOCKED|UNVERIFIED\`, \`command\`, \`exit_code\`, \`sha\`, \`platform\`, \`evidence\`, \`reason\`, \`scope\`, \`executed_at\`. Não usar \`PASS_LIMITED\` como status principal; use \`status=PASS, scope=process-launch-only\` e gate funcional separado \`UNVERIFIED\`.

Prioridade:
- P0: falha de segurança ou exclusão arbitrária, assinatura incorreta, manifesto de versão errada aceito, corrupção de sessão, install/uninstall destrutivos, bypass de autorização/isolamento.
- P1: gofmt, go vet, lint, docs, testes obrigatórios e divergência toolchain.
- P2: melhoria visual/refatoração não essencial.

Fazer commits pequenos e descritivos **somente na release/v1.0.0**, precedidos por \`git diff --check\`, \`git status --short\` e verificação do HEAD remoto. Push somente fast-forward. Se houver alteração remota inesperada, preservar o trabalho e pedir reconciliação; nunca forçar.

Resultado:
1. Informar novo SHA e diff resumido, com links de commits.
2. Listar por etapa comandos realmente executados + exit code e logs.
3. Identificar o que foi resolvido, o que permanece BLOCKED e qual gate exige acesso externo.
4. Emitir \`GO_WINDOWS_VALIDATION\` somente se todos os gates funcionais Windows requeridos tiverem evidência; nunca confundir com \`GO_DISTRIBUTION\`.
5. **Não criar tag, release, publicar manifesto nem copiar para main** neste ciclo sem decisão explícita.

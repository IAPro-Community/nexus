# Nexus 1.0.0 — Gates de homologação TokenBar / Linux / Windows

Parecer inicial: **NO-GO até que todas as evidências do mesmo SHA estejam PASS**.
Um teste adicionado não é um teste executado; um CI cancelado/pendente não
certifica uma release. Branch única: \`release/v1.0.0\`.

## Antes de começar

Interrompa pushes concorrentes na release. Faça fetch e fast-forward, registre
\`git rev-parse HEAD\` e confirme \`git status --porcelain\` vazio. Não
use \`reset --hard\` ou \`push --force\` para homologar. Testes devem usar
diretórios temporários isolados; não importe transcripts reais nos testes.

Antes de atualizar um banco SQLite real, faça **backup consistente**: feche
CLI, Web, Desktop e HUD e copie o arquivo de banco e seus eventuais
\`-wal\`/\`-shm\`, ou utilize \`sqlite3 DATABASE ".backup 'BACKUP'"\`.
Execute \`PRAGMA integrity_check;\` no backup e exija resposta \`ok\`.
Proteja o backup contra acesso público, inclusive em artefatos CI.

## Linux AMD64

Requer Go **1.25.14**, Bun **1.3.9**, Playwright Chromium e bibliotecas GTK3,
WebKit2GTK 4.1, libsoup 3, gio-unix. Para smoke gráfico isolado: Xvfb/DBus.

\`\`\`bash
git fetch origin release/v1.0.0
git switch release/v1.0.0
git pull --ff-only origin release/v1.0.0
test -z "$(git status --porcelain)" || exit 1
sha="$(git rev-parse HEAD)"
go version
bun --version
export NEXUS_EXPECTED_SHA="$sha"
./scripts/certify-tokenbar-integration.sh --browser
./scripts/homologate-beta.sh
test "$(git rev-parse HEAD)" = "$sha"
\`\`\`

Logs gerados em \`DEV/validation/local-beta/<VERSION>/<SHA>/linux-x86_64/\`.
Ambos os scripts devem terminar com exit code 0. O script amplo inclui
\`go vet\`, \`go test -race\`, \`govulncheck\`, linters, testes frontend,
browser E2E, builds nativos e pacote DEB. Os testes interativos indicados
em \`MANUAL_CHECKS.md\` continuam obrigatórios.

## Windows AMD64 (PowerShell)

O contrato Windows usa Go **1.26.9**, diferente do Linux, e Bun **1.3.9**.
Requer Playwright Chromium, PowerShell 7 e WebView2 Runtime.

\`\`\`powershell
git fetch origin release/v1.0.0
git switch release/v1.0.0
git pull --ff-only origin release/v1.0.0
if (git status --porcelain) { throw 'dirty tree' }
$sha = (git rev-parse HEAD).Trim()
go version
bun --version
.\scripts\certify-tokenbar-integration.ps1 -WithBrowser
.\scripts\homologate-beta.ps1 -ExpectedSha $sha
if ((git rev-parse HEAD).Trim() -ne $sha) { throw 'SHA changed' }
\`\`\`

Logs em \`DEV/validation/local-beta/<VERSION>/<SHA>/windows-AMD64/\`.
O script homologate compila Wails Desktop e HUD, mas **não substitui** os
smokes nativos de WebView2, ConPTY, instalação/desinstalação e assinatura.
No GitHub, a certificação Windows assinada exige workflow_dispatch do CI
com \`native_target_sha\` exato, runner \`nexus-windows\`, ambiente
\`native-signing\` aprovado, signtool, NSIS e credenciais. \`skipped\`
significa ausência de certificação, não PASS.

## Checklist manual de saída (ambos os sistemas)

1. Terminal: rodar \`nexus codex\` e \`nexus agy\`, testar setas, scroll,
   seleção, colagem Unicode e redimensionamento. Nenhuma sequência literal
   \`[A\`/\`[B\` pode aparecer e nenhuma conversa deve cancelar sozinha.
   No Windows repetir sob ConPTY real.
2. Perfis: vincular quatro ou cinco contas Codex e validar que Usage, HUD,
   quota pace, reset, login/logout e roteamento preservam isolamento.
3. Transcripts Claude: negar consentimento (nenhuma leitura), conceder
   manualmente, importar JSONL, acrescentar linhas, importar novamente,
   testar deduplicação, exclusão de registros/offsets e POST remoto negado.
4. Custo: diferenciar preço de assinatura, estimativa equivalente de API,
   valor informado pelo provedor e faturamento real; nenhum total UNKNOWN
   vira zero. Testar falha de rede no HUD sem mostrar dado antigo como LIVE.
5. CSV: testar exportação no navegador e Desktop/Wails, HTTP 401,
   autenticação, neutralização de fórmulas, limite de linhas e omissão de
   \`account_scope\`, credenciais e paths.
6. Provedores públicos: testar status OpenAI/Claude/GitHub com sucesso,
   timeout, falha de rede e resposta inválida; nunca convertê-lo em quota.
7. GUI: validar páginas Usage/Cost/HUD, tray, autostart, modo compacto e
   expansão, scroll e responsividade, sem tela preta ou cortes.
8. Nativo: validar startup/shutdown Desktop/HUD, WebView2 ou GTK,
   autenticação/CSRF, túnel, updater, rollback e desinstalação sem processos
   órfãos ou alteração de dados de outra sessão.

Cada evidência deve conter SHA completo, plataforma, comando/ação,
resultado, UTC e log/captura (sem segredos).

## Interpretação e correção

- **CI queued** sem runner atribuído: não houve teste; conferir runner
  self-hosted e homologar localmente se CI indisponível.
- **CI cancelled**: execução não concluída. Congelar pushes até rodar
  gates para o SHA final; a concorrência do CI cancela execuções antigas.
- **Gofmt/Prettier/TS/lint/go vet**: corrigir e publicar commit novo,
  invalidando evidências do SHA anterior.
- **Race/ConPTY/terminal**: reproduzir em ambiente isolado, recolher logs
  e corrigir antes de tentar GO.
- **SQLite**: abrir cópia do banco, inspecionar \`schema_migrations\`,
  \`PRAGMA integrity_check\` e comparar tabelas 0029/0030.
- **Wails/WebView2**: verificar dependências, sessão, logs e identidade
  da instância nativa usada no smoke. Não encerrar processos por nome global.
- **Assinatura / updater**: ausência de chave/credencial, pacote inválido
  ou falha de verificação exige NO-GO; não desabilitar esse gate.

## Política de rollback SQLite

0029/0030 são migrações **forward-only**. Não há downgrade destrutivo
seguro: interrompa todos os processos, restaure o backup consistente
pré-migração, reinstale o binário anterior e revalide a integridade do
SQLite. Nunca execute DROP TABLE/DELETE em produção para fazer rollback.
O reparo automático do Nexus recria objetos faltantes, mas estruturas
incompatíveis são bloqueadas para investigação.

**GO só é possível com evidências PASS em todos os gates no mesmo SHA**
(Go/TS/frontend/browser, Linux/Windows nativos, Codex/ConPTY, auth,
múltiplas contas, Wails, instalador, updater, assinatura). Em qualquer
outro caso o parecer é NO-GO.
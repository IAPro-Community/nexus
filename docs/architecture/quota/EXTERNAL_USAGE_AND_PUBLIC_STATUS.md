# Uso externo do Claude e status público — Nexus 1.0.0

Este módulo foi implementado com código próprio do Nexus. O TokenBar é
referência de arquitetura, sem cópia de código GPL-3.0-or-later.

## Uso externo do Claude Code

A fonte de arquivos é fixa: `~/.claude/projects` do **usuário do sistema
operacional que executa o backend Nexus**, não necessariamente o navegador.

- GET /api/v1/cost/external-claude apenas consulta metadados já persistidos.
- POST /api/v1/cost/external-claude exige uma sessão autenticada, proteção CSRF
  e corpo `{"consent":true}` em **cada execução manual**.
- Não há leitura contínua, processo em background nem watcher.
- O scanner nunca aceita um caminho de filesystem informado pela requisição.
- Limites por operação: até 800 arquivos, 8 MiB por arquivo e 48 MiB
  de leitura total. Somente linhas JSONL terminadas em newline são processadas.
- Arquivos symlink são ignorados. O scanner não escreve em diretórios do Claude.
- A migração SQLite 0030 armazena unicamente o hash do caminho, offset da última
  linha completa, hash do identificador da mensagem, modelo, timestamp e tokens.
  Não armazena prompts, respostas, credenciais, paths ou e-mails.
- Mensagens duplicadas atualizam o mesmo registro, sem duplicar contadores.
- A cada varredura manual, o Nexus remove automaticamente observações com
  mais de 90 dias e offsets não atualizados por 90 dias; uma remoção manual
  explícita continua disponível nas preferências de custo.
- A leitura valida o descritor aberto contra o arquivo verificado na listagem,
  impedindo a troca por symlink entre a descoberta e o `os.Open`.
- É impossível atribuir com segurança esses transcripts locais a perfis
  autenticados do Nexus. Todos ficam em LOCAL_DEFAULT_UNVERIFIED, numa tabela
  separada; não entram em `cost_observations`, orçamento, custo efetivo ou
  decisão do Resource Scheduler.
- Não são estimadas cobranças por tokens da assinatura. São medições de uso,
  não despesas em dólares.

Uma conciliação atribuída futura depende da verificação de conta, sessão e
invocação para provar que a observação não está duplicada no ledger.

## Incidentes de serviços públicos

- GET /api/v1/providers/public-status, autenticado e somente leitura.
- Fontes fixas: OpenAI, Claude e GitHub Status, via HTTPS, sem repasse de
  tokens, cookies ou identidade de contas do Nexus.
- Timeout por fonte de 3 segundos, consultas concorrentes, limite de leitura
  JSON, sem redirecionamentos e cache local de 5 minutos.
- Cada fonte falha de forma isolada. Falha de rede é UNKNOWN/stale.
- O estado público do serviço não altera a quota canônica, a autenticação,
  o monitor de capacidade ou o roteamento.

## Gates pendentes

O CI GitHub Actions está sem evidência verde no SHA integrado. Executar com
Go 1.25 e Node compatível, no mesmo SHA da release:

```shell
go test ./internal/nexus/externalusage ./internal/nexus/store ./internal/control/web ./internal/nexus/capacitymonitor ./internal/nexus
cd web && npm run quality:full
```

Certificar a interface Usage, Cost e HUD, verificar CSRF, isolamento entre
contas e smoke Windows WebView2/ConPTY. A validação não está concluída apenas
porque o GitHub aceitou os commits.

## Exportação de custo canônico

O endpoint autenticado `GET /api/v1/cost/observations.csv?days=30`
exporta no máximo 1.000 observações recentes do ledger canônico (até 365
dias mediante parâmetro). Não expõe `account_scope`, caminhos, prompts,
identificadores de execução ou credenciais. Um custo desconhecido aparece
como célula vazia, jamais como zero. Os textos são protegidos contra fórmulas
em CSV. Os dados externos do Claude continuam fora desse relatório.

As exportações CSV de custo e capacidade usam o mesmo helper frontend que
anexa o Bearer token no Desktop/Wails; links HTML diretos não garantem esse
cabeçalho. Em Web convencional a autenticação da sessão segue pelo cookie.

Ações manuais de importação e exclusão não concorrem: uma operação ativa
retorna HTTP 409 para a outra, impedindo que dados apagados sejam reinseridos
por uma importação simultânea. O scanner opera no sistema que hospeda o Nexus,
não no dispositivo do navegador remoto.

O parser descarta timestamps no futuro distante ou anteriores a 2000 e limita
rótulos de modelo a 128 bytes para evitar persistir metadados arbitrários.

### Homologação reproduzível sem CI

Na própria branch `release/v1.0.0`, sem alterações locais, execute:

- Windows PowerShell: `./scripts/certify-tokenbar-integration.ps1 -WithBrowser`
- Linux/macOS: `./scripts/certify-tokenbar-integration.sh --browser`

Esses scripts exigem Go >=1.25, validam `gofmt`, executam testes Go
focados e `npm run quality:full`. Com a opção de navegador executam
`test:e2e:capacity` (somente HTTP), **mais** Playwright real nas suítes
`test:e2e:surface` (responsividade e acessibilidade) e
`test:e2e:composer-themes` (temas e componentes de aparência). Os
smokes nativos Windows ConPTY/WebView2 e assinatura de release continuam
obrigatórios fora desses scripts. Enquanto o runner estiver na fila, o
resultado é **NÃO CERTIFICADO**.

O script não modifica arquivos: um erro de `gofmt` requer formatação
explícita e novo commit antes de retentar o gate no mesmo SHA.

## Incremental hardening after initial integration

- External Claude metadata can be purged even when the 7-day displayed summary
  is empty; the backend DELETE removes the full imported history and offsets.
- Cost and capacity CSV now use a shared spreadsheet-formula neutralizer,
  including Unicode BOM and zero-width prefixes, with regression tests.
- Capacity HTTP E2E validates consent, CSRF, exported CSV headers and public
  provider state independently. The `--browser` certification option runs
  real Playwright surface and theme suites as well as the HTTP suite.
- CSV downloads have frontend unit tests for desktop Bearer authorization,
  restricted URLs and handling of HTTP 401.

All of these gates are **specified but not recorded as passed** until run
against the exact final release SHA. GitHub Actions cancellation is not success.

## Endurecimentos adicionais na homologação

- A ação **POST de leitura de transcripts** exige conexão local ao backend e
  origem local (`localhost`, `127.0.0.1`, `::1` ou `wails.localhost`).
  Conexões de LAN e túneis públicos não podem iniciar a leitura do diretório
  privado do usuário do servidor. GET do resumo e exclusão permanecem sob
  sessão Nexus autenticada e CSRF para mutações.
- Offsets incrementais são vinculados a um hash do caminho **e do prefixo
  estável de 256 bytes do arquivo**. O prefixo só é processado em memória:
  não fica armazenado. Isso impede o caso de uma substituição por outro JSONL
  de mesmo tamanho herdar o offset antigo e ocultar novas mensagens.
- A varredura possui teto de 20.000 entradas de diretório, além dos limites
  anteriores de arquivos, bytes e tempo. Substituições de arquivo e chamadas
  de origem remota agora têm testes de regressão.
- O cálculo externo continua não atribuído; a defesa contra file replacement
  não prova que as mensagens pertencem a um perfil Nexus.

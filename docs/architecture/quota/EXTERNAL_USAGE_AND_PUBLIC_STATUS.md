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

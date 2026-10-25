# Referência da CLI

## TL;DR

Use `nexus --help` para a fonte de verdade da versão instalada. A CLI aceita
`--json` nos comandos indicados para automação; não faça parsing de tabelas
humanas quando houver saída JSON.

```bash
nexus --help
nexus doctor --json
nexus providers status --json
```

## Comandos por objetivo

| Grupo | Sintaxe | Uso |
| --- | --- | --- |
| workspace Web | `nexus`, `nexus web`, `nexus web open` | abre ou reabre a workstation local |
| provider interativo | `nexus <provider> [flags]` | inicia a CLI oficial com seleção de perfil |
| provider supervisionado | `nexus <provider> --supervised` | usa SessionHost e controles Nexus |
| runtimes | `nexus start`, `stop`, `ps`, `attach`, `handoff`, `continue` | controla execuções supervisionadas |
| providers e perfis | `providers`, `profiles`, `add`, `remove`, `rename`, `login`, `logout`, `use`, `status`, `inspect` | registra e opera perfis locais |
| projetos e agents | `projects`, `agents`, `plan` | trabalha com entidades persistentes do Nexus |
| sessões e bindings | `sessions`, `workspaces`, `current`, `bind`, `unbind`, `bindings` | consulta continuidade e preferências locais |
| diagnóstico | `doctor`, `security`, `history`, `stats`, `export`, `issue-report`, `config` | investiga e exporta dados locais |
| sistema | `update`, `maestro`, `completion`, `version`, `release`, `tunnel` | manutenção e integrações opcionais |

## Índice de comandos de topo

O índice abaixo foi verificado contra `nexus --help`. Flags específicas podem
variar por subcomando e versão; confirme com `nexus <subcomando> --help` antes
de automatizar.

| Sintaxe | Descrição |
| --- | --- |
| `nexus` | abre a Web Workspace OS padrão |
| `nexus web [flags]`, `nexus web open` | inicia ou reabre a Web local |
| `nexus tunnel` | opera o acesso remoto/tunnel conscientemente |
| `nexus control [subcmd]` | controla runtimes supervisionados (`nexus ui` é alias) |
| `nexus start <provider> [flags]` | inicia runtime supervisionado e anexa |
| `nexus stop <runtime-id>` | encerra um runtime |
| `nexus ps`, `nexus running` | lista runtimes ativos |
| `nexus attach <runtime-id>` | anexa ao terminal do runtime |
| `nexus handoff <id> <target>` | transfere contexto entre contas do mesmo provider |
| `nexus continue <id> --with <provider>` | transfere contexto entre providers |
| `nexus run "<goal>"` | classifica, roteia e executa um objetivo Nexus em foreground até a Mission concluir, falhar ou exigir intervenção |
| `nexus <provider> [flags]` | abre provider com seleção inteligente |
| `nexus <provider> --supervised` | abre provider via SessionHost |
| `nexus <provider>:<profile> [flags]` | pede um perfil específico |
| `nexus <provider>:auto [flags]` | pede seleção automática explícita |
| `nexus resume [id] [provider:name]` | retoma sessão com sintaxe nativa |
| `nexus providers [--json]` | lista instalações/capacidades de providers |
| `nexus providers status [--json]` | mostra registro e ciclo de vida de providers |
| `nexus providers register <provider> [--profile <name>] [--no-login]` | registra um provider/perfil |
| `nexus profiles [--json]` | lista perfis locais |
| `nexus paths` | mostra diretórios Nexus efetivos |
| `nexus add <provider> [name]` | adiciona perfil de autenticação |
| `nexus remove <provider> <name>` | remove perfil |
| `nexus rename <provider> <old> <new>` | renomeia perfil |
| `nexus login`, `nexus logout <provider> <name>` | executa login/logout oficial |
| `nexus use <provider> <name>` | define perfil padrão |
| `nexus status [provider[:profile]]` | mostra saúde/plano do perfil |
| `nexus usage [--json] [--refresh]` | consulta quota/capacidade |
| `nexus inspect <provider> <name>` | inspeciona configuração de perfil |
| `nexus projects [list\|add]` | lista ou registra workspaces |
| `nexus agents [project-id]` | lista Agents persistentes de um projeto |
| `nexus plan [list\|create\|show\|compile\|run]` | gerencia WorkPlans |
| `nexus sessions [search] [--json]` | consulta índice universal de sessões |
| `nexus workspaces [--json]` | mostra workspaces, histórico e bindings |
| `nexus current` | mostra provider/perfil ativo |
| `nexus bind`, `unbind`, `bindings` | gerencia preferências por workspace |
| `nexus explain <provider>` | explica a decisão de seleção |
| `nexus doctor [--json\|--bundle]` | gera diagnóstico local |
| `nexus doctor terminal [--json\|--recover]` | diagnostica e recupera o estado do terminal |
| `nexus security [profile] [--json]` | audita isolamento e compartilhamento |
| `nexus history`, `stats`, `export` | consulta/extrai dados locais |
| `nexus issue-report` | cria relatório diagnóstico |
| `nexus config <show\|validate>` | lê ou valida configuração |
| `nexus update [check] [--channel beta\|stable]` | verifica/aplica update do Nexus; mudança de canal é explícita |
| `nexus maestro <status\|doctor\|update>` | administra a integração Maestro opcional |
| `nexus completion <bash\|zsh\|fish>` | gera completion de shell |
| `nexus version [--json]` | mostra versão e plataforma |
| `nexus release` | orienta bump, build, instalação e validação de release |

Para execução autônoma iniciada pela CLI, o processo permanece proprietário do worker canônico até a Mission atingir um estado estável (`COMPLETED_VERIFIED`, falha, pausa ou `BLOCKED_NEEDS_USER`). A CLI não cria um segundo runner nem abandona uma goroutine ao encerrar. No Web, a execução continua assíncrona porque o processo servidor permanece vivo e `RecoverMissionRuns` restaura trabalho durável após restart.

Para projetos usados com Missions autônomas, `--isolation worktree` é o
default recomendado quando você quer que Agents sem configuração própria
herdem isolamento seguro:

```bash
nexus projects add /caminho/do/repo --name "Meu Projeto" --isolation worktree
```

Ele não é mais um requisito global. Steps `AUTO`/`CREATE` podem executar em
um projeto cujo default é `project`, porque o runtime reutiliza apenas Agents
com isolamento efetivo `worktree` ou cria um Agent isolado. Steps `EXISTING`
continuam bloqueados se o Agent fixado não for worktree-safe.

O nome posicional legado continua aceito; `--isolation` aceita `project`,
`worktree` ou `none`.

## Sintaxe de provider

```bash
nexus codex
nexus codex:work
nexus codex:auto
nexus agy --supervised
```

`<provider>:<profile>` pede um perfil específico; `:auto` torna a seleção
automática explícita. O provider continua responsável pela autenticação
oficial. Verifique disponibilidade antes com `nexus providers status --json`.

### Aliases canônicos

| Alias Nexus | Significado | Descrição e Comportamento |
| --- | --- | --- |
| `--yolo` / `-y` | Execução autônoma sem prompts | Pede bypass de aprovações e confirmações interativas ao provider quando suportado |
| `--lead` | Agente Lead Interativo (`STICKY_LEAD`) | Vincula a sessão ao papel Lead persistente; promove tarefas para WorkPlans estruturados |
| `--continue` | Continuidade da sessão anterior | Continua a sessão mais recente no mesmo workspace (AGY/Claude também aceitam `-c`) |
| `--resume <id>` / `-r <id>` | Retomada por ID | Retoma uma sessão existente pelo seu identificador único de histórico |
| `--print` / `-p` | Saída não-interativa | Executa em modo pipeline/print (AGY/Claude aceitam `-p`; Codex reserva `-p` para `--profile`) |
| `--effort <low\|med\|high>` | Nível de raciocínio | Ajusta o esforço cognitivo / reasoning budget do modelo (quando suportado) |
| `--plan` | Modo de planejamento | Inicia a sessão em modo de formulação de plano antes de qualquer modificação de código |

> **Nota sobre o Codex**: No provider Codex, `-c` representa `--config` e `-p` representa `--profile`. Para aliases canônicos do Nexus, use explicitamente `--continue` e `--print`.

## Execução Autônoma & Conductor

### `nexus run "<goal>"`
O comando de execução de maior nível do Nexus. Recebe um objetivo em linguagem natural, aciona o Conductor para classificação e decomposição, gera o DAG do WorkPlan e coordena a equipe virtual de agentes até a verificação final:

```bash
nexus run "Adicionar suporte a exportação de métricas em formato Prometheus no endpoint /metrics"
```

- **Pipeline Executado**: Análise de intenção → WorkPlan estruturado → Execução isolada em sandboxes → Testes automatizados → Relatório de evidências (`COMPLETED_VERIFIED`).
- **Segurança**: Pausa imediatamente e solicita intervenção humana se detectar risco destrutivo ou alteração fora dos guardrails.

### `nexus conductor dispatch "<goal>"`
Dispara explicitamente a camada de decomposição sem execução imediata:
```bash
nexus conductor dispatch "Refatorar camada de persistência para isolamento multitenant"
```

---

## Controle Supervisionado de Runtimes (`nexus control` / `nexus ui`)

O Nexus SessionHost permite iniciar, pausar, inspecionar e alternar entre sessões de agentes executando em segundo plano:

| Comando | Descrição |
| --- | --- |
| `nexus start <provider> [flags]` | Inicia um novo runtime supervisionado e anexa o terminal |
| `nexus stop <runtime-id>` | Encerra um runtime ativo |
| `nexus ps` ou `nexus running` | Lista todos os runtimes supervisionados em execução |
| `nexus attach <runtime-id>` | Anexa a sessão de terminal a um runtime em background |
| `nexus handoff <id> <target>` | **Handoff intra-provider**: transfere o contexto ativo para outra conta do mesmo provedor |
| `nexus continue <id> --with <provider>` | **Handoff cross-provider**: resume o contexto e histórico convertendo para um provedor diferente |

---

## Quotas, Inteligência de Custos & Diagnósticos

### `nexus usage`
Exibe a tabela interativa e pesquisável de consumo de quotas em tempo real:
```bash
nexus usage
nexus usage --json        # Ideal para integrações e dashboards
nexus usage --refresh     # Força chamada ativa às APIs/CLIs para atualizar cache
```

### `nexus explain <provider>`
Explica detalhadamente por que uma determinada conta/perfil foi selecionada, expondo a matriz de pontuação (pesos de quota, janelas de 5h e 7 dias, latência e histórico de rate limits):
```bash
nexus explain agy
nexus explain codex
```

### `nexus doctor`
Audita a integridade do ambiente de execução, terminais e dependências:
```bash
nexus doctor
nexus doctor terminal             # Audita modos TTY, PTY/ConPTY e gerações de terminal
nexus doctor terminal --recover   # Restaura modos crus (raw mode) e limpa flags travadas
nexus doctor --bundle             # Gera pacote diagnóstico sanitizado para suporte
```

### `nexus security`
Audita limites de isolamento, variáveis de ambiente e compartilhamento de arquivos:
```bash
nexus security
nexus security <profile> --json
```

---

## Exemplos validados contra o help

```bash
nexus providers status --json
nexus profiles --json
nexus projects list
nexus agents <project-id>
nexus plan list
nexus ps --json
nexus doctor --bundle
nexus completion zsh
```

Cada subcomando pode ter opções adicionais. Confirme antes de automatizar:

```bash
nexus <subcomando> --help
```

## Saída e erros

- Use `--json` quando o help do comando o oferecer.
- Preserve códigos de saída em scripts e trate `UNKNOWN`, falta de autenticação
  e quota limitada como estados distintos de sucesso.
- Não inclua tokens, cookies ou paths pessoais em logs/issue reports.

## Tunnel

`nexus tunnel` é uma operação de acesso remoto separada da Web loopback padrão.
Confirme status com `nexus tunnel status` e siga a política de segurança local
antes de iniciar ou compartilhar uma URL.

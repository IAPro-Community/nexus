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
| `nexus run "<goal>"` | classifica, roteia e executa um objetivo Nexus |
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
| `nexus security [profile] [--json]` | audita isolamento e compartilhamento |
| `nexus history`, `stats`, `export` | consulta/extrai dados locais |
| `nexus issue-report` | cria relatório diagnóstico |
| `nexus config <show\|validate>` | lê ou valida configuração |
| `nexus update` | atualiza somente o Nexus |
| `nexus maestro <status\|doctor\|update>` | administra a integração Maestro opcional |
| `nexus completion <bash\|zsh\|fish>` | gera completion de shell |
| `nexus version [--json]` | mostra versão e plataforma |
| `nexus release` | orienta bump, build, instalação e validação de release |

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

| Alias Nexus | Significado |
| --- | --- |
| `--yolo` / `-y` | pede bypass de aprovações ao provider, quando compatível |
| `--continue` | continua a sessão mais recente |
| `--resume <id>` / `-r <id>` | retoma uma sessão pelo ID |
| `--print` | pede execução não interativa |
| `--effort <low\|med\|high>` | preferência de esforço, quando suportado |
| `--plan` | inicia em modo de planejamento, quando suportado |

Para Codex, `-c` e `-p` permanecem flags nativas. Use `--continue` e
`--print` para os aliases Nexus.

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

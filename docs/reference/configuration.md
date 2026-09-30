# Configuração

## TL;DR

Use `nexus paths` para descobrir os diretórios efetivos e `nexus config show`
ou `nexus config validate` para inspecionar a configuração. Segredos de
providers são resolvidos em runtime e não devem ser persistidos na configuração
do Nexus.

## Variáveis documentadas

| Variável | Obrigatória | Default / efeito | Descrição | Exemplo seguro |
| --- | ---: | --- | --- | --- |
| `NEXUS_WEB_PORT` | não | `13000` no control host; `nexus web` também aceita `--port` | porta de controle/Web quando aplicável | `NEXUS_WEB_PORT=7420` |
| `NEXUS_LAUNCH_MODE` | não | heurística do Nexus | força `direct` ou `supervised` | `NEXUS_LAUNCH_MODE=supervised` |
| `NEXUS_DELEGATION_MODE` | não | `OFF` em sessões interativas | override explícito da política interativa: `AUTO`/`ASK` habilitam promoção para Mission; `OFF` mantém o terminal nativo e provider-owned | `NEXUS_DELEGATION_MODE=ASK` |
| `NEXUS_DATA_DIR` / `AI_CLI_DATA_DIR` | não | diretório de dados da plataforma | substitui o local de dados; útil em teste/isolamento | `NEXUS_DATA_DIR=/tmp/nexus-data` |
| `NEXUS_CONFIG_DIR` / `AI_CLI_CONFIG_DIR` | não | diretório de configuração da plataforma | substitui o local de configuração | `NEXUS_CONFIG_DIR=/tmp/nexus-config` |
| `AI_CLI_LANG` | não | locale detectado | escolhe idioma da CLI | `AI_CLI_LANG=pt-BR` |
| `NEXUS_ORQUESTRADOR_DIR` | não | descoberta padrão | local da integração Maestro opcional | `/opt/orquestrador` |
| `NEXUS_CODEX_APP_SERVER` | não | comportamento normal do adapter | controla a consulta app-server do Codex; use apenas para diagnóstico | `NEXUS_CODEX_APP_SERVER=0` |
| `NEXUS_DEBUG` / `NEXUS_AGY_DEBUG` | não | desativado | ativa logs de depuração | `NEXUS_DEBUG=1` |
| `NEXUS_UPDATE_REGISTRY_URL` | não | `https://github.com/IAPro-Community/nexus/releases/download/channel-stable/update-manifest.json` (ou `channel-beta`) | URL completa do manifesto assinado; sobrescreve o default GitHub | `https://updates.example.invalid/manifest.json` |
| `NEXUS_RELEASE_REPO` | não | `IAPro-Community/nexus` | owner/name do GitHub usado por installers e pelos ponteiros `channel-*` de update | `NEXUS_RELEASE_REPO=acme/fork` |

As variáveis `OPENAI_API_KEY`, `DEEPSEEK_API_KEY` e equivalentes são lidas
somente quando você configura um provider de inteligência correspondente. Nunca
as copie para documentação, issues ou commits.

## Docker de compatibilidade

`.env.example` é específico do caminho Docker/host-parity. Os campos
`HOST_HOME_DIR`, `HOST_UID`, `HOST_GID`, `WORKSPACE_DIR`, `HOST_BIND_ADDRESS`,
`NEXUS_WEB_PORT`, `I_ACCEPT_HOST_PARITY`, `ENABLE_SSH_AGENT` e
`SSH_AUTH_SOCK_MOUNT` devem ser revisados antes de subir o compose. Leia
[Docker compatibilidade](../getting-started/docker-compat.md) antes de definir
`I_ACCEPT_HOST_PARITY=1` ou montar um SSH agent.

## Regras de segurança

- Não grave tokens, senhas, chaves ou connection strings em `.env` versionado.
- Não configure uma porta pública para a Web sem uma revisão explícita de
  autenticação e rede.
- Use diretórios temporários isolados para testes que alteram `NEXUS_*_DIR`.

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
| `NEXUS_DELEGATION_MODE` | não | `OFF` | delegação é opt-in: `AUTO`/`ASK` precisam ser explícitos; `OFF` mantém execução direta/provider-owned e impede promoção implícita para Mission | `NEXUS_DELEGATION_MODE=ASK` |
| `NEXUS_DATA_DIR` / `AI_CLI_DATA_DIR` | não | diretório de dados da plataforma | substitui o local de dados; útil em teste/isolamento | `NEXUS_DATA_DIR=/tmp/nexus-data` |
| `NEXUS_CONFIG_DIR` / `AI_CLI_CONFIG_DIR` | não | diretório de configuração da plataforma | substitui o local de configuração | `NEXUS_CONFIG_DIR=/tmp/nexus-config` |
| `AI_CLI_LANG` | não | locale detectado | escolhe idioma da CLI | `AI_CLI_LANG=pt-BR` |
| `NEXUS_ORQUESTRADOR_DIR` | não | descoberta padrão | local da integração Maestro opcional | `/opt/orquestrador` |
| `NEXUS_CODEX_APP_SERVER` | não | comportamento normal do adapter | controla a consulta app-server do Codex; use apenas para diagnóstico | `NEXUS_CODEX_APP_SERVER=0` |
| `NEXUS_DEBUG` / `NEXUS_AGY_DEBUG` | não | desativado | ativa logs de depuração | `NEXUS_DEBUG=1` |
| `NEXUS_TERMINAL_TRACE` | não | desativado | grava diagnóstico JSONL sanitizado do caminho de bytes do terminal (xterm/WebSocket/scrubber/SessionHost/PTTY), sem texto bruto de prompt | `NEXUS_TERMINAL_TRACE=1` |
| `NEXUS_UPDATE_REGISTRY_URL` | não | `https://github.com/IAPro-Community/nexus/releases/download/channel-stable/update-manifest.json` (ou `channel-beta`) | URL completa do manifesto assinado; sobrescreve o default GitHub | `https://updates.example.invalid/manifest.json` |
| `NEXUS_RELEASE_REPO` | não | `IAPro-Community/nexus` | owner/name do GitHub usado por installers e pelos ponteiros `channel-*` de update | `NEXUS_RELEASE_REPO=acme/fork` |
| `NEXUS_TUNNEL_HOSTNAME` | não | Quick Tunnel (hostname aleatório) | hostname público estável do Cloudflare Named Tunnel; exige `NEXUS_TUNNEL_TOKEN` | `nexus.exemplo.com` |
| `NEXUS_TUNNEL_TOKEN` | não | Quick Tunnel (hostname aleatório) | token do Named Tunnel; chega ao `cloudflared` pelo ambiente do processo filho, nunca em argv | `eyJhIjoi…` |
| `NEXUS_REMOTE_ORIGIN` | não | nenhum | origem privada e estável que o operador compartilha com o aparelho pareado (por exemplo, um nome de tailnet do Tailscale ou um DNS interno servido por proxy confiável); URL absoluta `https://host` | `https://nexus.tail1234.ts.net` |

Com `NEXUS_TUNNEL_TOKEN` e `NEXUS_TUNNEL_HOSTNAME` definidos, o acesso remoto
sobe um Cloudflare Named Tunnel de hostname fixo; sem eles, usa o Quick Tunnel
(hostname sorteado a cada início, o que invalida um PWA instalado). O túnel
nomeado exige **porta local fixa** — `nexus web --port 13000` — porque a regra de
ingress do dashboard aponta para um `http://localhost:<porta>` específico.
Detalhe em [acesso remoto](../product/remote-access.md).

`NEXUS_REMOTE_ORIGIN` é a peça da opção privada (Tailscale): um endereço de
tailnet é um nome, não um IP, e a política de origem exige que o host do
request seja idêntico à origem do navegador. Publicar a origem abre essa
porta sem desligar a regra de host exato. Com ela definida, o botão **QR Code**
do acesso remoto emite convite para esse endereço **sem** subir o túnel do
Cloudflare — o túnel seria redundante. O botão **Ativar Acesso Remoto** do
Cloudflare continua disponível e é o caminho da opção pública.

A publicação descreve **um endereço exato**: mesmo esquema, mesmo hostname e
mesma porta efetiva. Publicar `https://nexus.tail1234.ts.net` não autoriza
`http://nexus.tail1234.ts.net` nem `https://nexus.tail1234.ts.net:8443` — outra
porta é outro serviço. Se o proxy TLS usa uma porta própria, publique a
origem com essa porta (`https://nexus.tail1234.ts.net:8443`).

A publicação também é o que autoriza um proxy confiável que reescreve o
`Host` do request (`tailscale serve` e equivalentes): nesse caso o `Origin` do
navegador é o endereço publicado, enquanto o `Host` recebido é o serviço
local. A regra de host exato continua valendo em todos os outros casos, e um
request que chegue por interface pública não é coberto por uma publicação
privada.

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

# ADR: URL estável para acesso remoto (Gate 0 do PWA)

## Status

Aceito (transporte implementado; ativação exige credencial do maestro)

## Contexto

O acesso remoto usa Cloudflare **Quick Tunnel**, que sorteia um hostname
`https://<aleatório>.trycloudflare.com` a cada `StartTunnel`. Três consequências
quebram um app instalável:

1. O PWA instalado tem `scope`/`start_url` no hostname antigo; após reiniciar o
   túnel, o ícone abre uma origem morta e o usuário precisa reinstalar.
2. O cookie de sessão (`HttpOnly`, `Secure`, host-bound) e o CSRF são emitidos
   para o hostname antigo; a sessão não é reaproveitada.
3. A política de origem (`originpolicy.RegisterTunnelHost`) é revogada quando o
   processo cloudflared morre (`trackTunnelExit` → `SetTunnelActive(false)`).

Ou seja: sem URL estável, "PWA + notificações + dispositivo lembrado" não é
projetável — é frágil por construção.

## Decisão

Suportar **Cloudflare Named Tunnel** como transporte preferencial, mantendo o
Quick Tunnel como fallback explícito:

- `NEXUS_TUNNEL_TOKEN` — token do túnel criado no dashboard Cloudflare.
- `NEXUS_TUNNEL_HOSTNAME` — hostname público estável (`nexus.exemplo.com`).
- Com ambos definidos, `handleTunnelStart` chama `StartNamedTunnel`
  (`cloudflared tunnel --no-autoupdate run`, token via env `TUNNEL_TOKEN`).
- Sem eles, o comportamento atual (Quick Tunnel) permanece inalterado.

O ingress do dashboard deve apontar para `http://localhost:<porta>` do Nexus, e
a porta precisa ser fixa: com token, o `cloudflared` não carrega regra de
ingress própria, mas aplica a do dashboard. Um túnel nomeado iniciado sobre
porta efêmera não tem como funcionar, então o Nexus recusa essa combinação em
vez de publicar um host que responde `502`.

### Princípios Chave

1. **Nenhuma credencial em argv, log ou repositório**: o token chega ao
   cloudflared pela variável de ambiente do processo filho.
2. **Mesmo modelo de autorização**: URL estável **não** autentica ninguém. O
   convite/código de 8 caracteres, o TTL de 15 min, o uso único e a revogação ao
   parar o túnel continuam valendo sem alteração.
3. **Fail-closed preservado**: `SetTunnelActive(false)` no exit do processo
   continua revogando sessões remotas; a diferença é apenas o hostname.
4. **Configuração, não código**: ativar o modo estável não exige recompilar.
5. **A estabilidade é declarada, não presumida**: o status do túnel carrega
   `stable_url` e `named_hostname`, e a interface mostra "URL Estável" ou
   "URL Temporária". O Quick Tunnel nunca é reportado como estável, mesmo com
   uma configuração nomeada presente — o hostname sorteado é outro. Um operador
   não pode descobrir que o app instalado quebrou só depois do reinício.
6. **A origem privada é um endereço exato**: `NEXUS_REMOTE_ORIGIN` publica um
   endereço, não uma família. Mesmo esquema, mesmo hostname e mesma porta
   efetiva; outra porta é outro serviço. A publicação também é o que autoriza
   um proxy confiável que reescreve o `Host` do request (`tailscale serve`),
   caso em que a regra de host exato não se aplica — mas só quando o `Origin`
   do navegador é exatamente o endereço publicado e o request chegou a um host
   local ou privado. Um request em interface pública nunca é coberto por uma
   publicação privada.

## Alternativas Consideradas

- **Tailscale (`nexus.<tailnet>.ts.net`)**: URL estável e nada exposto
  publicamente; exige app Tailscale no celular e não atende "abrir de qualquer
  navegador". Recomendado quando privacidade for prioridade máxima.
  Implementado como segunda opção: `NEXUS_REMOTE_ORIGIN` publica a origem
  privada, e o Nexus passa a emitir convite para ela sem subir o túnel do
  Cloudflare. Publicar a origem tira o servidor da classe `LOOPBACK`: o
  bootstrap deixa de ser reutilizável, o cookie exige `Secure` e a sessão
  deixa de ser persistida.
- **LAN + proxy TLS próprio**: sem dependência externa, mas só funciona na mesma
  rede e exige certificado confiável no dispositivo.
- **Manter só Quick Tunnel**: rejeitado como base do PWA — hostname aleatório por
  start invalida instalação, cookie e CSRF.

## Consequências

- O PWA passa a ter origem estável quando o modo nomeado está ativo; sessão,
  service worker e futuras notificações push ficam vinculados a um hostname real.
- O papel do convite (`viewer`/`controller`) é fixado na emissão; `viewer` é
  somente leitura. Um `controller` ainda concede acesso amplo ao workspace
  (ferramentas locais e execução de comandos), então recomenda-se Cloudflare
  Access (MFA) na frente do hostname e/ou Tailscale para uso privado.
- Evidência obtida: `npm --prefix web run test:e2e:remote-access` prova o
  caminho de dois dispositivos em túnel HTTPS real (convites `controller` e
  `viewer`, pareamento em 390px, recusa do `viewer`, registro de dispositivos e
  revogação isolada). O mesmo roteiro reinicia o Nexus e compara a origem
  pública: no Quick Tunnel ela **muda** (hostname novo sorteado a cada start),
  o que mede em execução o defeito que a URL estável remove; com
  `NEXUS_TUNNEL_TOKEN`/`NEXUS_TUNNEL_HOSTNAME` o roteiro exige origem
  idêntica após o reinício e falha se divergir. A cobertura do caminho do
  Named Tunnel é unitária (`internal/control/web/tunnel_named_test.go`):
  configuração completa/ausente/parcial, token fora do argv, URL derivada do
  hostname e recusa de entradas inválidas. Ainda falta o reingresso no
  hostname estável com credencial real, o ensaio em aparelho físico e a
  execução no SHA publicado. Até lá, `NO_GO`.

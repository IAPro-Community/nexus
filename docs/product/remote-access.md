# Acesso remoto e autenticação — Nexus 1.0 beta

## Compartilhamento por túnel HTTPS

Em **Configurações → Acesso remoto**, ative o Cloudflare Quick Tunnel. Clique em **QR Code** para emitir um convite individual. Use **Copiar convite de acesso** para compartilhar a URL completa, incluindo o fragmento `#nexus_bootstrap=`. A URL pública do túnel, sem convite, não autentica nenhum dispositivo.

Cada convite tem um token aleatório próprio, aceito apenas no hostname do túnel que o emitiu. Ele é válido por **15 minutos contados da emissão**, serve para **um único login** e é invalidado quando o túnel é encerrado. Clique em **Gerar outro convite** para autorizar um segundo dispositivo ou substituir um convite expirado. Gerar um convite não reinicia nem troca a sessão de quem já está autenticado.

O fragmento não é enviado em URLs de requisição; o navegador o remove da barra de endereços antes da troca via POST. O cookie resultante utiliza `HttpOnly`, `Secure` e `SameSite=Strict`, com prazo máximo de 24 horas. Ele também pode sobreviver ao fechamento e reabertura do navegador no mesmo hostname, desde que a sessão continue válida no servidor. A sessão tem TTL absoluto de 24 horas e TTL por inatividade de 4 horas; isso não é acesso permanente. Reiniciar o Nexus **não** restaura sessões remotas.

**Papéis do convite:** o convite é emitido com um papel fixo, escolhido no momento da emissão (`controller` ou `viewer`) e nunca elevado depois. O papel `viewer` produz uma sessão somente leitura: o servidor recusa qualquer requisição mutante com `403` antes do handler rodar, recusa a listagem de dispositivos pareados (`403`) e bloqueia o terminal. O papel `controller` exige consentimento explícito na interface antes de gerar o convite. Um celular pareado não consegue enumerar nem revogar os outros dispositivos: essa operação exige a sessão do host.

**Aviso de segurança:** nesta beta o papel `controller` concede acesso amplo ao workspace, incluindo ferramentas locais e execução de comandos. Um convite `viewer` é restrito a leitura, mas revela o estado do workspace. Compartilhe apenas com pessoas de confiança. O acesso não é automaticamente autorizado ao abrir somente a URL pública.

## URL estável para o app instalável (Named Tunnel)

O Quick Tunnel sorteia um hostname novo a cada início. Isso invalida o app
instalado no celular: o ícone aponta para uma origem morta, e o cookie de sessão
(`HttpOnly`, `Secure`, host-bound) não é reaproveitado.

Para uma URL fixa, crie um **Named Tunnel** no dashboard Cloudflare, aponte o
ingress público para `http://localhost:<porta>` do Nexus e inicie o Nexus com:

- `NEXUS_TUNNEL_TOKEN` — token do túnel (nunca versionar; não colocar em argv).
- `NEXUS_TUNNEL_HOSTNAME` — hostname público, por exemplo `nexus.exemplo.com`.

Com as duas variáveis definidas, o botão de acesso remoto sobe o túnel nomeado e
o hostname permanece o mesmo entre reinícios. Sem elas, o Quick Tunnel continua
sendo o padrão. A URL estável **não** autentica ninguém: convite, código de 8
caracteres, TTL de 15 minutos, uso único e revogação ao parar o túnel seguem
idênticos. Ainda assim, recomenda-se Cloudflare Access (MFA) na frente do
hostname. Alternativa privada: Tailscale (`nexus.<tailnet>.ts.net`), sem
exposição pública, exigindo o app Tailscale no celular. Decisão registrada em
`docs/architecture/ADR-remote-access-stable-url.md`.

**A porta local precisa ser fixa.** Um túnel nomeado não carrega regra de
ingress própria: com token, o `cloudflared` aplica a ingress configurada no
dashboard, que aponta para um `http://localhost:<porta>` específico. Por isso o
Nexus deve ser iniciado com `nexus web --port <porta>` em uma porta conhecida
(por exemplo `13000`), e não em porta efêmera (`--port 0`). O Nexus recusa
iniciar o túnel nomeado em porta efêmera, porque o resultado seria um host
público respondendo `502` — e, antes dessa verificação, o túnel apareceria como
ativo na interface.

## Como ativar a URL estável (runbook)

Dois caminhos, conforme a escolha do maestro.

**Opção A — Cloudflare Named Tunnel (recomendada para "abrir de qualquer lugar")**

1. No dashboard Zero Trust, crie um túnel do tipo Cloudflared na conta que
   DETém o domínio.
2. Crie um Public Hostname: subdomínio (por exemplo `nexus.exemplo.com`) com
   serviço `HTTP` e porta **igual à porta local do Nexus** (por exemplo `13000`).
   A porta do dashboard e a do processo precisam ser a mesma.
3. Copie o token do túnel (é a credencial; trate como segredo).
4. Suba o Nexus com a porta fixa e as duas variáveis:

   ```bash
   NEXUS_TUNNEL_TOKEN='<token>' \
   NEXUS_TUNNEL_HOSTNAME='nexus.exemplo.com' \
   nexus web --port 13000 --no-open
   ```

5. Em **Configurações → Acesso remoto**, ative o túnel. A URL exibida deve ser
   `https://nexus.exemplo.com` — igual ao hostname configurado.
6. Pare e suba o Nexus de novo: a URL deve ser **byte-idêntica**. É essa
   propriedade que mantém o ícone do PWA e o cookie de sessão vivos.
7. Opcional e recomendado: coloque Cloudflare Access (MFA) na frente do
   hostname. O acesso remoto do Nexus não autentica pelo hostname.

**Opção B — Tailscale (recomendada para uso privado)**

1. Instale o Tailscale no servidor e no celular e una os dois à mesma tailnet.
2. Exponha o Nexus em uma porta fixa local. O `tailscale serve` termina o TLS
   na tailnet e fala com o processo local, então o Nexus continua em loopback:

   ```bash
   nexus web --port 13000 --no-open
   ```

   E exponha a porta pelo Tailscale:

   ```bash
   tailscale serve --bg http://127.0.0.1:13000
   ```

   A alternativa é dispense o proxy e servir direto na interface do tailnet
   (`--listen <ip-do-tailnet> --remote`), mas aí a origem publicada precisa
   apontar para um endereço que o browser usa de fato.

3. Publique a origem privada para o Nexus saber qual endereço compartilhar:

   ```bash
   NEXUS_REMOTE_ORIGIN='https://nexus.<tailnet>.ts.net' \
   nexus web --port 13000 --no-open
   ```

4. O endereço é estável e não é público. Com a origem publicada, o cartão
   **Acesso remoto** mostra **URL Privada** e o botão **QR Code** emite o
   convite para esse endereço **sem** subir o túnel do Cloudflare — o túnel
   seria redundante. O botão **Ativar Acesso Remoto** continua disponível e é
   o caminho da opção pública.

5. A publicação descreve um endereço **exato**: mesmo esquema, mesmo hostname
   e mesma porta. Se o `tailscale serve` publica em `https://nexus.<tailnet>.ts.net`
   (porta 443), publique exatamente assim. Publicar a origem com uma porta
   diferente da que o navegador usa faz o pareamento ser recusado — por
   desenho, não por falha: outra porta é outro serviço.

6. O `Host` do request pode ser reescrito pelo proxy (`tailscale serve`
   termina o TLS e fala com o serviço local). A publicação é o que autoriza
   esse caso: o `Origin` do navegador precisa ser exatamente o endereço
   publicado. Um request que chegue pela interface pública não é coberto por
   uma publicação privada.

O modo privado não é mais frouxo que o público. Publicar a origem tira o
servidor da classe `LOOPBACK`: o bootstrap deixa de ser reutilizável, o cookie
passa a exigir `Secure` e a sessão deixa de ser persistida. O convite, o
código de 8 caracteres, o TTL de 15 minutos, o uso único e os papéis
`viewer`/`controller` são os mesmos.

**Não use o Quick Tunnel como base do PWA.** Ele sorteia um hostname novo a
cada início, o que invalida a instalação, o cookie de sessão e o CSRF. Serve
para uma demonstração pontual, não para o app do celular.

### Verificação automatizada

```bash
NEXUS_TUNNEL_TOKEN='<token>' NEXUS_TUNNEL_HOSTNAME='nexus.exemplo.com' \
  NEXUS_E2E_PORT=13400 npm --prefix web run test:e2e:remote-access
```

O roteiro pareia dois dispositivos, valida o papel de cada um e exige que a
origem pública seja **idêntica** antes e depois do reinício do Nexus. Sem as
duas variáveis, ele roda no Quick Tunnel e reporta a mudança de origem.

## Notificações no celular (Web Push)

Em **Configurações → Acesso remoto**, o card **Notificações no celular** ativa o Web Push no aparelho. O navegador pede a permissão, registra a assinatura (`PushManager.subscribe`) e o Nexus guarda apenas o material criptográfico necessário — a chave pública do aparelho e o segredo de autenticação — indexado pelo hash do endpoint, nunca pelo endpoint em claro.

A notificação chega quando **um novo episódio de atenção** aparece na projeção que a própria tela de atenção lê. Um watcher de 30 s compara o estado pendente com o que já foi anunciado: o episódio novo notifica, a reconferência do mesmo episódio fica em silêncio, e um episódio resolvido e reaberto notifica de novo. O conteúdo enviado é a pergunta ou o resumo pendente — nunca o conteúdo do workspace. O payload é cifrado para o aparelho (RFC 8291, `aes128gcm`) e autenticado com VAPID (ES256).

O envio de teste (`Enviar teste`) fica disponível só na sessão do host, para que um celular pareado não use o canal para spam dos outros aparelhos.

## Acesso na rede local

O bind remoto precisa ser habilitado explicitamente. A autenticação no navegador exige **HTTPS com certificado confiável**. URLs LAN em HTTP (`http://192.168.x.x:PORTA`) não são uma forma suportada de autenticação: os cookies usam `Secure` e o bootstrap retorna uma indicação explícita de HTTPS obrigatório. Não remova o atributo `Secure` para contornar o problema.

Para acesso autenticado pela LAN, configure um proxy TLS de confiança para um endereço/IP privado aceito pela política de origem, ou utilize o túnel HTTPS. Um hostname DNS personalizado exige configuração explícita da política de hosts, sem wildcard e sem desligar as verificações de Origin/CSRF.

## Diagnóstico e homologação

- **Sessão inválida imediatamente após iniciar o túnel:** gere um novo convite na instalação local; a validade do convite não depende mais da hora de inicialização do processo.
- **Convite já utilizado ou expirado:** gere outro; o mesmo link não pode autenticar várias pessoas.
- **URL do túnel sem fragmento:** não é um link de autorização.
- **Cookie não enviado após login:** confira HTTPS, hostname e políticas do navegador; verifique os cookies HttpOnly/Secure.
- **Rede local via HTTP:** configure HTTPS ou use o túnel; não haverá login inseguro de conveniência.
- **Reinício do servidor:** sessões remotas não são restauradas por design.

Antes de publicar uma beta instalável, executar `go test ./internal/control/web/...`, verificações TypeScript/UI e smoke em Chrome/Edge no Windows e Linux. Validar dois dispositivos por túnel, troca de hostname, reinício, logout, CSRF, WebSocket do terminal, expiração e reutilização de convites.

O cenário de dois dispositivos por túnel público já tem automação: `npm --prefix web run test:e2e:remote-access` sobe um Nexus real com Quick Tunnel, emite um convite `controller` e um `viewer`, pareia dois navegadores móveis distintos em 390px, confere overflow zero na tela de pareamento, prova que o `viewer` é recusado em mutação e em listagem de dispositivos sem efeito colateral, confere o registro de dispositivos no host e valida que a revogação de um aparelho derruba apenas a sessão dele. O `cloudflared` usado é o binário oficial 2026.8.2 conferido por SHA-256 contra o mesmo digest fixado em `internal/control/web/tunnel.go`.

O mesmo roteiro reinicia o Nexus e compara a origem pública antes e depois. No Quick Tunnel a origem **muda** (hostname novo sorteado), o que mede na prática o defeito que a URL estável corrige. Definindo `NEXUS_TUNNEL_TOKEN` e `NEXUS_TUNNEL_HOSTNAME` o roteiro passa a exigir a origem idêntica após o reinício e falha se ela divergir.

Ele também confirma que o app é instalável **pela origem pública**: `manifest.webmanifest` e `sw.js` respondem `200` no hostname do túnel, o manifest pede modo de exibição instalável e o service worker fica registrado naquele escopo. Ainda falta o ensaio em aparelhos físicos e no SHA publicado.

## A interface declara se a URL é estável

O cartão **Acesso remoto** mostra, logo abaixo do endereço público, se ele é
**Estável**, **Temporário** ou **Privado**, com a consequência prática:

- **URL Estável** — o mesmo endereço depois de reiniciar. O app instalado no
  celular continua válido e o cookie host-bound é reaproveitado.
- **URL Temporária** — o endereço muda a cada reinício. O ícone do app
  instalado passa a abrir uma origem morta e é preciso reinstalar. A
  correção é configurar um túnel nomeado.
- **URL Privada** — o endereço é servido na rede privada (tailnet do Tailscale
  ou DNS interno). É estável por construção e **não precisa** do túnel do
  Cloudflare: o botão **QR Code** emite o convite para esse endereço sem
  subir o túnel. O túnel continua disponível para quem quiser o caminho
  público.

Essa distinção vem do servidor (`stable_url`, `named_hostname` e
`remote_origin` no status do túnel) e é exigida pelo roteiro de E2E nos dois
modos: no Quick Tunnel o status tem de declarar origem efêmera, e no Named
Tunnel, origem estável com o hostname configurado.

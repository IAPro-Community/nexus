# Acesso remoto e autenticação — Nexus 1.0 beta

## Compartilhamento por túnel HTTPS

Em **Configurações → Acesso remoto**, ative o Cloudflare Quick Tunnel. Clique em **QR Code** para emitir um convite individual. Use **Copiar convite de acesso** para compartilhar a URL completa, incluindo o fragmento `#nexus_bootstrap=`. A URL pública do túnel, sem convite, não autentica nenhum dispositivo.

Cada convite tem um token aleatório próprio, aceito apenas no hostname do túnel que o emitiu. Ele é válido por **15 minutos contados da emissão**, serve para **um único login** e é invalidado quando o túnel é encerrado. Clique em **Gerar outro convite** para autorizar um segundo dispositivo ou substituir um convite expirado. Gerar um convite não reinicia nem troca a sessão de quem já está autenticado.

O fragmento não é enviado em URLs de requisição; o navegador o remove da barra de endereços antes da troca via POST. O cookie resultante utiliza `HttpOnly`, `Secure` e `SameSite=Strict`. A sessão tem TTL absoluto de 24 horas e TTL por inatividade de 4 horas, e funciona durante novas navegações no mesmo hostname enquanto for válida. Reiniciar o Nexus **não** restaura sessões remotas.

**Aviso de segurança:** nesta beta o convite concede acesso amplo ao workspace, incluindo ferramentas locais e execução de comandos. Ainda não existe controle de papéis para convidados nem compartilhamento somente leitura. Compartilhe apenas com pessoas de confiança. O acesso não é automaticamente autorizado ao abrir somente a URL pública.

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

Antes de publicar uma beta instalável, executar `go test ./internal/control/web/...`, verificações TypeScript/UI e smoke em Chrome/Edge no Windows e Linux. Validar dois dispositivos por túnel, troca de hostname, reinício, logout, CSRF, WebSocket do terminal, expiração e reutilização de convites. Os testes adicionados ao repositório ainda precisam ser executados para certificar esses comportamentos.

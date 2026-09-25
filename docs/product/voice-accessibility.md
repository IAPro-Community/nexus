# Nexus Voice Accessibility — v1 beta

## Objetivo

Nexus Voice adiciona entrada por voz sem acoplar terminal, agentes ou navegação a um fornecedor
de speech-to-text. A entrega integrada ao candidato `v1.0.0-beta.2` usa APIs
remotas: **Groq / `whisper-large-v3-turbo`** como primeira opção e
**Cloudflare Workers AI / Whisper** como fallback configurável. xAI/Grok não é
provider padrão.

A feature faz parte da linha v1 beta; certificação física de microfone continua
separada por plataforma.

## Fluxos suportados

### Terminal Voice Input

O microfone do terminal não procura um campo HTML dentro de Codex, Claude, Gemini, Antigravity
ou outra TUI. Depois da transcrição, o Nexus envia o texto pelo mesmo WebSocket de input da
sessão PTY ativa:

```text
microfone -> STT -> transcript -> terminal WebSocket -> PTY stdin -> processo/TUI atual
```

A transcrição **não inclui Enter automaticamente**. Isso permite revisar ou complementar o texto
e evita transformar um erro de STT em execução de shell. O terminal exige lease `CONTROL`.

### Nexus Voice Control

O microfone global controla o Nexus por uma camada determinística, sem entregar a decisão de
ação a um LLM. Ele resolve comandos explícitos e também reutiliza o **Command Registry** do
Nexus para as ações marcadas como voice-safe. Isso cobre navegação e operações reversíveis como
Projetos, Agentes, terminais, Composer/Work, Missions, Resources, Sessions, Settings, Maestro,
layouts, modo Zen, abertura de nova sessão de IA, novo agente (abre o fluxo de criação) e Command
Palette.

Fala arbitrária ou comandos destrutivos continuam em `unknown`. A adição futura de um comando
destrutivo ao Command Registry não o torna automaticamente executável por voz: IDs aceitos por
voz usam allowlist/prefixos explícitos.

O microfone global também entende ditado explícito para o terminal ativo:

```text
"escreva no terminal git status"
            ↓
transcrição STT
            ↓
terminal ativo com lease CONTROL
            ↓
PTY stdin recebe "git status"
```

O Nexus **não acrescenta Enter**. Dizer "git status" sozinho no microfone global não executa
shell; é necessário usar a intenção explícita de terminal. Se não houver terminal ativo, conexão
PTY ou lease CONTROL, nada é enviado.


## Invariante P0 da migração Wails v3

A migração do shell nativo não pode alterar o roteamento STT do produto:

```text
captura Web/native
  -> Groq / whisper-large-v3-turbo (prioridade 1)
  -> Cloudflare Workers AI / @cf/openai/whisper-large-v3-turbo (fallback)
  -> transcript
  -> Nexus Voice Control OU terminal ativo
```

`automatic_fallback=true` permanece padrão. Rate limit/quota, autenticação, falha de rede e
indisponibilidade podem avançar ao próximo provider. Áudio/request inválido (400/413/415) não é
cascateado. O contrato tem regressão automatizada no backend e o bridge Wails v3 tem teste próprio
para Start/Stop/Cancel de captura.

## Providers e failover

A configuração fica em **Settings > Accessibility > Voz & Acessibilidade**.

- providers podem ser habilitados, desabilitados e reordenados;
- API keys são gravadas no backend em `voice-secrets.json`, nunca em `localStorage`;
- `voice-settings.json` contém apenas metadados não secretos;
- GET de settings retorna apenas `has_api_key` / `configured`, nunca o segredo;
- variáveis de ambiente continuam possíveis (`GROQ_API_KEY`,
  `CLOUDFLARE_API_TOKEN`, `CLOUDFLARE_ACCOUNT_ID`);
- 429/quota, falhas de autenticação, rede e 5xx podem avançar ao próximo provider;
- 400/413/415 são tratados como problema do request/áudio e **não** são reenviados
  automaticamente para todos os fornecedores;
- a UI mostra somente uso de áudio observado pelo Nexus. Não inventa quota global restante.

O backend impõe limite de upload de 25 MiB e duração configurável de 5 a 120 segundos.

## Segurança e privacidade

- voz vem desligada por padrão;
- captura é push-to-talk/click-to-talk, sem microfone continuamente aberto;
- o browser nunca recebe credenciais dos providers;
- áudio só deixa a máquina depois de o usuário iniciar a captura;
- fallback não executa shell e não adiciona Enter;
- erros de provider retornam status tipado sem incluir tokens;
- nenhuma transcript é tratada como evidência de sucesso de Mission.

## Multiplataforma

O frontend usa `navigator.mediaDevices.getUserMedia` + `MediaRecorder` e escolhe o MIME de
acordo com a capacidade real do WebView/browser (`webm/opus`, WebM, Ogg/Opus ou MP4). Não há
dependência de binário Whisper local.

| Superfície | Estratégia |
| --- | --- |
| Browser | APIs Web de captura; permissão do browser |
| Linux Desktop | Wails v3/WebKitGTK tenta Web capture; se a captura Web não estiver disponível, usa fallback nativo `pw-record` (PipeWire) ou `arecord` (ALSA), gerando WAV mono/16 kHz |
| Windows Desktop | Wails v3/WebView2 usa a permissão nativa do WebView2; nenhuma API key fica no WebView |
| macOS Desktop | Wails v3/WKWebView + permissão de microfone no bundle nativo |

Se `getUserMedia` ou `MediaRecorder` não estiver disponível, o Desktop consulta a capability
`nativeVoiceCapture`. No Linux ela só é anunciada quando `pw-record` ou `arecord` existe no
PATH. Se nenhuma rota real de captura estiver disponível, a interface desabilita o botão e informa
indisponibilidade em vez de simular funcionamento.

A compilação cross-platform não substitui homologação nativa de microfone: Windows e macOS
precisam passar a campanha same-SHA da linha v1 beta em runners nativos antes de serem anunciados
como certificados. Linux também precisa de um teste de dispositivo real para fechar o gate de
captura, pois CI headless não fornece um microfone físico confiável.

## Arquivos de estado local

```text
<NEXUS_CONFIG_DIR>/
  voice-settings.json   # não secreto
  voice-secrets.json    # 0600/best effort no Windows
  voice-usage.json      # métricas observadas localmente
```

Os arquivos de voz não alteram o schema principal de config e não criam um segundo Capacity
Monitor. Uma integração futura pode publicar health/cost STT no modelo canônico, sem duplicar
a lógica de Capacity.

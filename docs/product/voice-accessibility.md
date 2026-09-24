# Nexus Voice Accessibility — 1.1

## Objetivo

Nexus Voice adiciona entrada por voz sem acoplar terminal, agentes ou navegação a um fornecedor
de speech-to-text. A primeira entrega da linha 1.1 usa APIs remotas: **Groq /
`whisper-large-v3-turbo`** como primeira opção e **Cloudflare Workers AI / Whisper** como
fallback configurável. xAI/Grok não é provider padrão.

A V1 estável permanece congelada em `release/v1.0.0`; esta feature pertence à linha 1.1.

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

O microfone global usa uma gramática determinística para navegação não destrutiva (projetos,
agentes, terminais, Composer/Work, Missions, Resources, Sessions, Settings e Command Palette).
Fala arbitrária ou comandos destrutivos ficam em `unknown`; esta versão não envia voz global
para um LLM para decidir ações.

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
| Linux Desktop | Wails/WebKitGTK; captura e codec detectados em runtime |
| Windows Desktop | Wails/WebView2; mesma camada Web, sujeito à permissão de microfone do Windows |
| macOS Desktop | Wails/WKWebView + `NSMicrophoneUsageDescription` em `Info.plist` e `Info.dev.plist` |

Se `getUserMedia` ou `MediaRecorder` não estiver disponível, a interface desabilita o botão
e informa indisponibilidade em vez de simular funcionamento.

A compilação cross-platform não substitui homologação nativa de microfone: Windows e macOS
precisam passar a campanha same-SHA da linha 1.1 em runners nativos antes de serem anunciados
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

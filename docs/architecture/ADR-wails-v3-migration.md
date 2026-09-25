# ADR: Migração integral da camada nativa para Wails v3

## Status

Aceito para a linha beta. Merge condicionado a evidência nativa same-SHA.

## Decisão

Migrar Nexus Desktop e Nexus HUD juntos para `github.com/wailsapp/wails/v3 v3.0.0-beta.25` e `@wailsio/runtime 3.0.0-beta.25`.

O Core, REST/WebSocket loopback, Agents, Missions, Capacity, Cost e voz permanecem canônicos e compartilhados. Wails continua sendo somente a camada de apresentação nativa.

## Contrato

- Desktop e HUD usam `application.NewService` e o mesmo `OpenSharedBackend`.
- O frontend usa `Call.ByName` com FQN completo do service: `github.com/IAPro-Community/nexus/internal/desktop.<Service>.<Method>`.
- Bindings gerados e `wails.json` da v2 são removidos.
- Linux usa Wails v3 com tag `gtk3` para preservar o baseline WebKitGTK 4.1.
- Windows usa WebView2; macOS usa WKWebView.
- Tray, autostart, Screen API, work area e DPI vêm do Wails v3.

## Voz P0

A migração não altera o fluxo: captura -> Groq/Whisper -> Cloudflare Workers AI fallback -> transcript. Controle global só executa comandos voice-safe e ditado para terminal nunca adiciona Enter.

## Gate

A migração só é concluída quando Linux, Windows amd64 e macOS arm64 executarem os shells no mesmo SHA. Build cruzado não substitui esse gate.

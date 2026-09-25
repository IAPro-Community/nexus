# ADR: Nexus Desktop Multiplataforma com Wails v2 (histórico)

## Status
**Superseded** pela migração integral para Wails v3 (`ADR-wails-v3-migration.md`).

## Contexto
O IAPro Nexus foi concebido como um Autonomous Multi-Agent Workspace OS completo. Inicialmente acessível via navegador web (`nexus web`), o produto necessitava de uma superfície nativa de execução para Windows, macOS e Linux que oferecesse capacidades de integração ao sistema operacional (janelas nativas, system tray, notificações de sistema, seletores nativos de diretório/arquivo, deep links `nexus://` e inicialização no login).

## Decisão
Adotar o **Wails v2 estável** (`v2.15.0`) exclusivamente como uma casca nativa (shell) para o Nexus Desktop.

### Princípios Chave:
1. **One Nexus Core**: O mesmo Nexus Core escrito em Go (`internal/app`, `internal/control`, `internal/nexus`, etc.) gerencia tanto a execução web quanto desktop.
2. **One React Frontend**: Um único SPA React compilado em `web/src -> web/dist`, copiado para o diretório gerado `internal/control/web/embedded`, é embarcado no servidor web e na casca Wails (`web.EmbeddedDistFS()`).
3. **One Control API & Contract**: A comunicação entre a interface do usuário e o domínio Nexus é realizada via REST e WebSocket autenticado em loopback.
4. **Wails is a Shell, not a Second Backend**: O Wails expõe apenas capacidades nativas do SO (`internal/desktop`), não duplicando regras de negócio, runtimes de terminal (PTY/ConPTY), escalonador ou updater.
5. **No Electron, No Tauri**: Electron foi rejeitado por introduzir um runtime Node duplicado de 150MB+ e overhead desnecessário de memória. Tauri foi rejeitado por exigir uma stack Rust paralela. Wails utiliza os webviews nativos do SO (WebView2 no Windows, WKWebView no macOS, WebKitGTK no Linux) com Go unificado.

## Consequências
- A paridade entre Web e Desktop é intrínseca e garantida por construção.
- O mesmo Git SHA, versão e ciclo de release produzem todos os artefatos.
- Esta decisão foi substituída em setembro de 2026 pela migração integral de Desktop e HUD para Wails v3 beta.25.

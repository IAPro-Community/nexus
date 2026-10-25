# Validação do HUD Linux

## Desktop XWayland

O Nexus Desktop Wails e o HUD final foram iniciados com GTK 3/WebKitGTK na sessão GNOME 49 Wayland, por XWayland. Interações XTest e workspace/terminal foram exercitados no overlay visual anterior 6476b22e5ae0b69e6f7340d13915ad013e0151dd5119fe91142cbef69441b02e. Após as correções do Core, os binários exatos finais foram novamente abertos usando o D-Bus real da sessão, com HOME/XDG/perfil Nexus isolados: Desktop 1280×800 e HUD 224×158 ficaram visíveis; os dois Cores responderam HTTP 200; os processos encerraram com exit 0. A captura do HUD confirma um NEX. O protocolo está em NATIVE-004.

| Fluxo | Resultado | Evidência |
| --- | --- | --- |
| Inicialização do Desktop e HUD final | PASS | logs/native-exact-final-visual-evidence.txt; screenshots/native-desktop-exact-final-xwayland.png; screenshots/native-hud-exact-final-xwayland.png |
| Alternância Barra e Mascote | PASS | screenshots/native-single-mascot-hud-bar-xwayland.png e screenshots/native-single-mascot-hud-final-xwayland.png |
| Expansão de controles do HUD | PASS | screenshots/native-single-mascot-hud-expanded-xwayland.png |
| Clique na área vazia atravessa para a janela inferior | PASS | logs/native-clickthrough-observations.md |
| Clique no mascote é interceptado pelo controle | PASS | logs/native-clickthrough-observations.md |
| Clique-through com popup expandido | PASS | logs/native-clickthrough-observations.md |
| Bandeja nativa, auto-hide, foco e múltiplos monitores | NOT_RUN | Não exercitado no host disponível |
| Persistência de posição e geometria após reinício do HUD | NOT_RUN | Não há evidência final desse ciclo no mesmo overlay |
| Bolha nativa alimentada por execução real | BLOCKED | Sem sessão de agente autenticada no perfil isolado |
| Transparência visual do canal alpha | BLOCKED | Captura do compositor não preserva alpha de forma confiável |
| Interação direta sob Wayland | BLOCKED | Desktop executado em XWayland; sessão host Wayland |

Dimensões observadas nas capturas do HUD: Barra compacta 220×60 DIP; Mascote compacto 224×158 DIP; painel expandido 334×496 DIP.

## Limite de plataforma

Este resultado valida GTK/WebKit no XWayland sob GNOME/Wayland. Não valida posição global, click-through, foco, tray ou popup independente com Wayland nativo. A ausência de acelerador 3D utilizável também limita qualquer conclusão de fluidez ou consumo GPU de animações em hardware de distribuição.


Captura exata final: `screenshots/native-hud-resilience-final-xwayland.png`; binário SHA-256 e screenshot SHA-256 em `logs/native-final-resilience-visual-evidence.txt`. A janela visível contém um único NEX. O fundo preto da captura XWayland não é evidência confiável do canal alpha. A screenshot anterior de `6476b22e` permanece em `native-hud-exact-final-xwayland.png` como evidência dos testes interativos anteriores.

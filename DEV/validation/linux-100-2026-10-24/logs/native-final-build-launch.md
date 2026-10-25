# Observação manual — abertura do Desktop final

Tipo de evidência: registro manual, não stdout bruto.

Binário: `/tmp/nexus-linux-100-2026-10-24/bin/nexus-desktop`
Commit/fingerprint: `bf6adf0c80ac3531d7291844512a14a017ab0002` + `15e9eb89de6477f32456752f1dea97ea07d479d8f63793dfe2b89003cf5c754d`.
Host: Ubuntu GNOME 49 em sessão Wayland.

O executável foi iniciado depois do rebuild final. O argumento `--help` não é tratado como modo de ajuda pelo entrypoint Wails; o aplicativo iniciou o Core embutido e permaneceu ativo junto ao HUD vinculado. O log observável do Core informou uma porta loopback efêmera. O processo final foi identificado pelo caminho exato do binário de homologação e encerrado ao terminar o smoke. O processo anterior `__control-host` permaneceu ativo.

Limites: esta abertura usou o ambiente HOME padrão do shell e não foi isolada; nenhum botão de importação, login ou execução de agente foi acionado. Não foi capturada uma screenshot nesta abertura. As capturas nativas do HUD e workspace desta pasta foram produzidas no smoke XWayland anterior ao ajuste isolado do comando de login; o código do HUD e do workspace não mudou nesse ajuste. Esta evidência cobre inicialização do binário final, não interação visual completa da tela final.

Registro manual das observações coletadas durante o teste XTest; este arquivo não é stdout bruto.
Teste: click-through nativo do HUD compacto e expandido
Commit base: bf6adf0c80ac3531d7291844512a14a017ab0002
Fingerprint da árvore de fontes: 113c9202aecd849e054ce46e38a22f4e884c0a02b7e3dd5d5031ab2b6fadc15e
Plataforma: Ubuntu GNOME 49 / sessão host Wayland / app e fixture GTK em XWayland

Pré-condição: Desktop e HUD foram iniciados com HOME, XDG_CONFIG_HOME, XDG_DATA_HOME e NEXUS_DATA_DIR apontados para /tmp/nexus-linux-100-2026-10-24. Uma janela GTK fixture foi colocada abaixo do HUD.

1. Clique em área vazia do HUD compacto, coordenada aproximada de tela (55,37).
Esperado: evento chega à janela abaixo.
Observado: fixture registrou clicked.
Resultado: PASS.

2. Clique no controle NEX, coordenada aproximada (165,132).
Esperado: o HUD expande e o clique não chega à janela inferior.
Observado: HUD expandiu para o painel de controles; fixture não registrou clique.
Resultado: PASS.

3. Clique em área vazia do HUD expandido, coordenada aproximada (55,35).
Esperado: evento atravessa para a janela inferior.
Observado: fixture registrou clicked.
Resultado: PASS.

Esse teste demonstra click-through funcional em XWayland. Não mede o canal alpha da captura e não valida interação direta no compositor Wayland.

# Revisão visual — HUD e mascote NEX — 2026-10-26

## Escopo

- Intenção: revisar visualmente o HUD nativo corrente e preservar a identidade de um mascote NEX, sem introduzir modo ou personagem Companion 2D.
- Classificação: `REVIEW / PRESERVE`.
- Design system local descoberto em `web/src/design-system` (resolução `@../design-system`, confiança 0,95; registro em `logs/frontend-design-system-discovery-2026-10-26.json`). Não foi encontrado arquivo `design-profile.*` na raiz examinada.
- Nenhum componente ou estilo de produto foi alterado durante esta revisão visual.

## Evidência examinada

- Captura real da janela Wails HUD executada em Xvfb: [NEX único](screenshots/native-hud-current-build-xvfb-2026-10-26/hud-mascot.png), 224×158, RGB, SHA-256 `08ac39298d10e04c54d9af4ec90bc29fa351d8a1328f655554026575600e437d`.
- Captura real do Desktop Wails atual em Uso/Contas: [tela após importar](screenshots/native-current-desktop-xvfb-1440-2026-10-26/usage-after-import.png), RGB, SHA-256 `d0a0c9ad3f303e7828997ed5c404a44b5fb71065ca830698772d3df65add4f4c`. O screenshot mostra a tela nativa atual, resumo de importação e o mesmo NEX no HUD linkado.
- Captura de tela após clique que abriu os controles: [HUD e popover](screenshots/native-hud-current-build-xvfb-2026-10-26/hud-expanded-root.png), SHA-256 `2d84ed45e6495f4e4abc38b92d18212eeac305281f1a3b8571fdb2d87e9197ea`.
- Execução: `tools/native_hud_current_build_xvfb.sh`; log aprovado: `logs/native-hud-current-build-xvfb-2026-10-26-retest2.log`.
- Fingerprint do overlay compilado: `751cfc1a160c38d950ac5b9807e1febb9746d03981c861c722e5bd965e1ef012`; o manifesto de 1.428 fontes foi conferido sem divergências.

## Resultado

| ID | Verificação | Resultado | Evidência / limite |
| --- | --- | --- | --- |
| VIS-001 | Identidade do personagem no HUD | PASS | A janela Wails mostra uma única arte NEX (`NexusHud → GuardianMascot → NexPixelSprite`). Nenhum Companion 2D ou segundo mascote aparece. |
| VIS-002 | Leitura da imagem e posição dentro da janela | PASS parcial | A figura está nítida e legível no HUD de 224×158, inclusive sobre a tela Desktop Wails de Uso/Contas. A revisão cobre a apresentação observada; não comprova todos os temas, escalas ou monitores. |
| VIS-003 | Transparência, borda e click-through no compositor real | BLOCKED | Xvfb não fornece captura RGBA do compositor do host. A imagem RGB mostra o que o X server compôs e não prova alpha; executar sob Wayland/XWayland capturável ainda é necessário. |
| VIS-004 | Estados de atenção gerados por um agente real | BLOCKED | Fixtures anteriores mostram o balão e ações nativas, mas esta captura idle não valida evento real de agente. |

## Veredito

O build atual confirma visualmente um mascote NEX único e a interação básica do HUD. O Companion 2D separado não está implementado no caminho ativo e não existe uma segunda apresentação a remover. A aprovação do acabamento visual completo permanece parcial até validar alpha e estados reais no compositor suportado; este relatório não declara `GO_LINUX`.

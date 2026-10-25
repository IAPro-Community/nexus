# Validação do mascote NEX único

## Reconfirmação da decisão de produto — 2026-10-25

- Resultado: um único personagem NEX. Não há opção, renderer ou atlas Companion 2D nesta branch; `GuardianMascot` envolve exatamente um `NexPixelSprite`. Barra/Mascote/Bandeja mudam a apresentação do HUD, não a identidade do personagem.
- Reteste: cinco suítes focadas passaram, 25/25; log: [mascot-single-nex-confirmation-2026-10-25.log](logs/mascot-single-nex-confirmation-2026-10-25.log), SHA-256 `2453ee9895d4675715c48715cfaba7e0712ea67cca8d26f4cb2c25f2c39e4a9c`.
- Evidência visual nativa atual: [captura XWayland](screenshots/native-hud-current-tree-mascot-xwayland.png), SHA-256 `1c28a68052d77727269cd7e63bc612efab8207da0bf8785e7cdb23719c65e3df`. Mostra uma única arte NEX.
- Nenhum renderer ou apresentação Mascote foi removido porque não existe um segundo personagem/modo separado no código integrado. Atenção e humor são estados funcionais do mesmo NEX.

## Contrato confirmado

O Nexus deve exibir um único mascote NEX, com arte consistente e estados funcionais. O checkout de `release/v1.0.0` não contém `NexusGuardian.tsx`, atlas de Companion, segundo renderer ou modo Companion 2D. A PR #55 não foi integrada.

A apresentação Mascote do HUD renderiza o NEX por `GuardianMascot` → `NexPixelSprite`. Barra, Mascote e Bandeja são apresentações do HUD. `companionModel` fornece estado, atenção e adiamento para o mesmo NEX; não cria uma segunda personagem.

## Evidência e status

| Caso | Procedimento | Status | Observado |
| --- | --- | --- | --- |
| MASCOT-001 | Vitest de `GuardianMascot`, `GuardianModeSelector`, apresentação e modelos | PASS | Reexecutado no overlay `516b706a…` em 2026-10-25: 25/25 testes em cinco arquivos. O seletor oferece somente Barra/Mascote/Bandeja como apresentações do mesmo HUD; não há opção 2D separada. Log: `logs/mascot-no-2d-mode-regression-2026-10-25.log`. |
| MASCOT-002 | Executar HUD nativo recompilado em XWayland e inspecionar captura | PASS | Build Wails pós-FIX-012 abriu como `nexus-hud` em 224×158 e mostrou um único NEX; binário SHA-256 `a70789f8c5e6afa45515c84abab8a0a311070821719712227360d1ed6137a0d6`. Captura atual SHA-256 `98b4320b329512b349af90284f5851b9080b20be3d97fc4bccf19520ac16cd3c`: `screenshots/native-hud-gdk-fix-mascot-xwayland.png`. |
| MASCOT-003 | Renderizar estados e conclusão de missão em fixtures | PASS | 25/25 testes em cinco arquivos. `COMPLETED` e `COMPLETED_VERIFIED` deixam o NEX neutro; sucesso visual não é inferido sem telemetria própria verificada. |
| MASCOT-004 | Testar ações de atenção em fixture | PASS | Abrir, adiar e dispensar têm cobertura automatizada. |
| MASCOT-005 | Mostrar evento real de agente no balão nativo e acionar Abrir/Adiar | BLOCKED | A Mission real passou pelo runtime Direct, mas não foi conectada a uma captura do Attention Center no HUD. O caso separado com fixture nativa passou em MASCOT-006. |
| MASCOT-006 | Capturar balão de atenção e preferências na janela nativa | PASS | Capturas reais do HUD Wails recompilado: balão `REQUIRE_USER` acionável ao lado do NEX e painel de preferências; XTest acionou “Adiar”, recolhendo o balão sem apagar a tarefa do Attention Center. Setup e resultado: `logs/native-attention-fixture-setup.txt` e `logs/native-attention-fixture-click-snooze.txt`; build/binário: `logs/native-attention-current-hud-build.log` e `logs/native-attention-current-hud-binary.txt`. |
| MASCOT-007 | Medir transparência alpha confiável no compositor | BLOCKED | Captura XWayland disponível é RGB e não prova transparência. |
| MASCOT-008 | Validar o HUD do mascote em Wayland direto | PASS | O HUD abriu em Wayland; NEX expandiu/recolheu, Barra/Mascote/Bandeja foram identificados e a apresentação Mascote foi restaurada pelo tray nativo. Após reiniciar, o HUD voltou a exibir o controle do mesmo NEX. Captura de pixels e transparência seguem bloqueadas. |
| MASCOT-009 | Validar persistência completa dos estados reais após reinício | NOT_RUN | Não foi executada a matriz completa de reinício com eventos reais. |

## Capturas

- [HUD Wails após FIX-012](screenshots/native-hud-gdk-fix-mascot-xwayland.png): janela 224×158 com uma arte NEX; screenshot SHA-256 `98b4320b329512b349af90284f5851b9080b20be3d97fc4bccf19520ac16cd3c`; binário HUD SHA-256 `a70789f8c5e6afa45515c84abab8a0a311070821719712227360d1ed6137a0d6`; fingerprint `516b706a2c3cdca1a6d76200833a785186063350fa4d41db4494fcb67bdb7f7a`.
- [Auditoria do caminho do renderer](logs/mascot-single-renderer-current-audit.txt) e [verificação da identidade NEX](logs/nex-visual-verify-current-2026-10-25.log).
- [Balão de atenção nativo com fixture](screenshots/native-hud-attention-fixture-xwayland.png): janela HUD Wails 310×245, estado `REQUIRE_USER`, botão “Abrir” e ação “Adiar”; screenshot SHA-256 `b2bff6d18ab8214d682430c3f097e93bf1e338a1924af7ccb7b25f661c7ba08a`. Após “Adiar”, [captura recolhida](screenshots/native-hud-attention-fixture-snoozed-xwayland.png), 224×158, SHA-256 `cbad75a5d16d15e17d92b5d99447f33955281a9f35b3d36f4682dc822c159d5e`. O endpoint de atenção ainda continha a fixture (`HTTP 200`, `needs_you_count=1`); log: [clique e verificação](logs/native-attention-fixture-click-snooze.txt).
- [Preferências do HUD](screenshots/native-hud-expanded-single-nex-auth-fix-xwayland.png): painel real com Barra/Mascote/Bandeja e configuração do popover.
- [HUD recompilado após FIX-010](screenshots/native-hud-single-nex-mascot-fix-xwayland.png): uma arte NEX, janela 224×158; screenshot SHA-256 `6d7bf609f6926c104fba5628775d1eb5d7cfab2f36e2094c1ab73bde0521cb97`; fingerprint do overlay `eccc97fea75002ce905239be463cc5d37f095e7d7b1b524f2abe19a7c3568bbc`.
- [HUD nativo atual: um NEX](screenshots/native-hud-xwayland-position-fix.png) — binário HUD raw SHA-256 `95137b3af9324a4590e19d6b14106e6391b2bdcc97594bab54835a96fa62b984`; fingerprint do overlay `996e1063540370263158e70903477f4cbbc3035dbb33b93161e603bffbe3c133`.
- [HUD do tar Linux anterior](screenshots/native-hud-packaged-linux-position-fix.png): uma arte NEX.
- [HUD do tar no overlay atual](screenshots/native-hud-packaged-current-overlay-xwayland.png): uma arte NEX em 224×158; payload SHA-256 `cf00cee5b82c6f856625c87bee8aef1891e39ad975e3fbbe1a491b76457e1dfd`, tar SHA-256 `6552ca3c7162c2b6c1c7322e8814c1f461da14a71e7377ad2dc08d4635afda8f`, fingerprint `dac0d00b8410f70ed02c8b4c324780349655fed8f7e580d1944da1ec3eaecefa`.
- [Foco do controle de arraste](screenshots/native-nexus-hud-keyboard-drag-focus-xwayland-position-fix.png), [após a seta](screenshots/native-nexus-hud-keyboard-moved-xwayland-position-fix.png) e [posição restaurada após reinício](screenshots/native-nexus-hud-position-persisted-after-restart-xwayland-position-fix.png).
- [HUD expandido após clique no mascote](screenshots/native-nexus-hud-expanded-final-overlay-xwayland.png): 334×496 com controles Barra/Mascote/Bandeja.
- [Desktop do overlay final](screenshots/native-desktop-final-gemini-fix-xwayland.png), [importação isolada](screenshots/native-desktop-final-session-import-xwayland.png) e [reinício com quatro provedores](screenshots/native-desktop-final-after-restart-xwayland.png); hashes em `logs/native-final-overlay-smoke.txt`.
- [Barra nativa](../linux-100-2026-10-24/screenshots/native-single-mascot-hud-bar-xwayland.png) e [controles/preferências](../linux-100-2026-10-24/screenshots/native-single-mascot-hud-expanded-xwayland.png): o frontend é byte a byte igual ao build atual; as oito alterações posteriores estão no backend Go.
- A captura inicial escura do Desktop foi substituída pela [captura final visível](screenshots/native-nexus-desktop-final-xwayland.png). A imagem escura não é usada como aprovação.
- A execução funcional Wayland direta está resumida em [wayland-hud-functional-summary.txt](logs/wayland-hud-functional-summary.txt), [wayland-hud-atspi-validation.log](logs/wayland-hud-atspi-validation.log) e [wayland-hud-restart.log](logs/wayland-hud-restart.log). O compositor não permitiu obter uma captura direta de pixels.

## Resultado

A identidade visual validada continua sendo um único NEX; não há segundo mascote ou modo Companion 2D. No código atual, `NexusHud → GuardianMascot → NexPixelSprite` é um único caminho de renderização; a apresentação Mascote foi mantida porque exibe esse NEX. A captura nativa pós-FIX-012 confirma uma arte NEX, e os cinco arquivos de teste passaram em 25/25. O HUD também exibiu balão de fixture e a ação “Adiar”; a ligação de uma tarefa real de agente continua BLOCKED. Em Wayland direto, teclado físico/posicionamento global, alpha e persistência de todos os estados seguem em aberto.

## Reconfirmação no overlay do Attention Center

No follow-up de 2026-10-25, inspecionei novamente a captura nativa `screenshots/native-nexus-hud-single-mascot-xwayland.png`: ela mostra uma única arte NEX, SHA-256 `e58c1dbc6dc1c32705a627c85a2e86ef53b9ed9f280c41e10f65ca19a0be0127`. A suíte completa Web passou em 108 arquivos/672 testes e inclui a regressão do seletor; o overlay deste follow-up está identificado por `logs/source-provenance-p2-mission-attention.txt` (fingerprint `7368531977b7422eccdf00e7c676f57d440e8d736d9e114bcdc071b9d5885fe6`). O Companion 2D continua `NOT_PRESENT` e fora do escopo. O detalhe do Attention Center e os limites da fixture estão em [MISSION_ATTENTION_VALIDATION.md](MISSION_ATTENTION_VALIDATION.md).

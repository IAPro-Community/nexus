# Validação do mascote NEX único

## Decisão de produto

O Nexus deve ter **um mascote único**, bem acabado e funcional. Um Companion 2D como personagem ou modo adicional não faz parte do produto. A PR #55 permanece fora da release e não foi incorporada. O HUD ativo mostra a arte oficial NEX. Barra e Bandeja são apresentações do HUD e não mascotes.

A lógica em companionModel.ts contém projeção de atenção, humor e adiamento usados pelo mesmo mascote. O caminho ativo rende uma única GuardianMascot com NexPixelSprite; não há NexusGuardian, atlas de Companion, renderer ou modo visual alternativo na branch testada. O seletor apresenta somente Barra, Mascote e Bandeja; um teste impede a introdução de rótulo ou modo “2D”.

## Evidência

| Caso | Resultado | Evidência |
| --- | --- | --- |
| Seletor sem personagem ou modo Companion 2D | PASS | web/src/features/hud/GuardianModeSelector.test.tsx |
| NEX único renderizado no HUD nativo — overlay exato final | PASS visual no XWayland | screenshots/native-hud-resilience-final-xwayland.png; logs/native-final-resilience-visual-evidence.txt |
| Alternância Barra → Mascote no app nativo | PASS em XWayland | screenshots/native-single-mascot-hud-bar-xwayland.png; screenshots/native-single-mascot-hud-final-xwayland.png; screenshots/native-hud-exact-final-xwayland.png |
| Controles expandidos | PASS | screenshots/native-single-mascot-hud-expanded-xwayland.png |
| Estados e ações cobertos por fixtures | PASS automatizado | GuardianMascot.test.tsx; companionModel.test.ts; guardianModel.test.ts |
| Abrir, adiar e dispensar atenção em fixture | PASS automatizado | GuardianMascot.test.tsx |
| Preferência de movimento reduzido | PASS automatizado | GuardianMascot.test.tsx |
| Evento real do runtime apresentado no mascote nativo | BLOCKED | RUNTIME_VALIDATION.md; não foi iniciada sessão Nexus autenticada |
| Persistência nativa e ciclo completo de estados reais após reinício | NOT_RUN | Matriz final de reinício não foi executada |
| Transparência alpha medida no compositor | BLOCKED | Captura disponível não preserva alpha confiável |
| Screenshot nativa do balão de atenção com preferências visíveis | NOT_RUN | Testes de componente usam fixture, mas nenhuma fixture foi injetada na janela Wails para captura |


O XTest mostrou que o botão do NEX recebe cliques e que regiões vazias do HUD deixam o clique chegar à janela GTK abaixo. O fundo preto nas capturas vem do caminho de captura, portanto não prova nem reprova a transparência.

## Resultado

A renderização visual do mascote único e a alternância Barra/Mascote passaram no escopo exercitado. As fixtures validam estados e ações cobertos. Não está homologado como o mascote reage a runtime real, nem a persistência completa, Wayland direto ou transparência alpha.


A screenshot exata do HUD final em `screenshots/native-hud-resilience-final-xwayland.png` corresponde ao binário SHA-256 `57a30228ed3060c134a91d6ee7cf5202ae941799faae7960729f2ac54b2dca7d` e fingerprint 84bef2759af6b975d73e0488ab80e5fced2df70f17855597d512dcdef1f29ec8. A janela nativa de 224×158 mostra um único NEX; o Core respondeu HTTP 200 e o processo encerrou com exit 0. A tentativa anterior em D-Bus privado não exibiu a janela e foi substituída pelo reteste sob a sessão real (detalhes no log). Alpha, Wayland direto e estados produzidos por runtime real seguem sem validação.


Checksums e hashes exatos dos binários das capturas finais: `logs/native-exact-final-visual-evidence.txt`; evidência bruta de cada execução em `logs/native-desktop-exact-final-build-sha256.txt` e `logs/native-hud-exact-final-build-sha256.txt`.

# NEX — catálogo completo de movimentos e giro 360° (plano de produção)

**Data da revisão:** 28/10/2026  
**Identidade:** o robô frontal branco/azul do PNG escolhido, sem redesenhar capacete, visor, aletas, torso, núcleo ou quatro módulos.  
**Estado verificado em 29/10/2026:** o player 360° já existe no `IAPro-Community/Nexus-Site`, mas as **16 vistas reais NÃO foram produzidas**. A [PR #147](https://github.com/IAPro-Community/nexus/pull/147) incorporou na release os PNGs originais, camadas e atlas experimental de 96 quadros. O master OpenRaster é recuperável byte a byte a partir de seis partes rastreadas no Git por `node scripts/restore-nex-openraster.mjs`. **Esses assets ainda são candidatos, não aprovados para o HUD/site público.**

## Classificação de viabilidade

- **A — já alcançável com as camadas reais v1.1:** movimento de olhos isolados, flutuação, módulos independentes, brilho, piscadas; a fidelidade deve ser avaliada durante a ação.
- **B — alcançável após novo rig e underpainting:** pescoço/cabeça independente, orelhas articuladas, rotação do casco, verdadeira abertura das aletas, expressões desenhadas no visor, módulos passando para trás do corpo.
- **C — exige desenhos de novos ângulos:** frente, perfis e costas (360°), inclinações fortes, acenos em perspectiva e órbitas com oclusão completa.
- **D — não disponível sem nova geometria ou atlas adicional:** câmera livre arbitrária, poses 3D sem restrição, deformações de peças rígidas não previstas.

## 1. Matriz de movimentos

| Movimento | Grupo afetado | Classe | Uso | Observação |
| --- | --- | --- | --- | --- |
| Flutuar/respirar | corpo/sombra | A | Idle, todo produto | ±2–5px na versão Hero; reduzir no HUD |
| Piscada simples/dupla | olhos | A | Idle, site | Base do visor corrigida na v1.1 |
| Olhar para esquerda/direita/cima | olhos | A | Idle | Apenas limites pequenos sem escorregar para fora do visor |
| Foco e curiosidade | olhos | B | Idle, execução | Precisam de sprites de expressão |
| Surpresa, aprovação e alegria | visor | B | Ilustração | Sucesso somente com evento real |
| Cochilar/acordar | olhos/energia | B | Idle prolongado | Nunca confundir `offline` com sono |
| Inclinar a cabeça | cabeça real | B | Idle, hover | No protótipo atual o *corpo todo* inclina; requer recorte da cabeça |
| Ajustar aletas | aletas | B | Idle | O protótipo atual anima brilho, não articulação geométrica |
| Pulso do reator | núcleo | A | Idle/working | Nunca inventar progresso percentual |
| Brilho e reflexos | núcleo/visor/casco | A | Site | Sempre discretos, sem flashes agressivos |
| Órbita de quatro módulos | módulos | A/B | Idle e working | Pequenos deslocamentos A; cruzar atrás do corpo exige B |
| Inspecionar um módulo | olhos + módulo | A/B | Idle | Microação com cooldown; sem telemetria falsa |
| Reorganizar os módulos | 4 módulos e órbita | B | Idle raro | Manter identidade e ordem dos quatro |
| Núcleo em alerta (laranja) | luz | A | Warning | Apenas com evidência canônica |
| Prioridade crítica (vermelho) | olhos/luz | A | Critical | Sem estroboscópio |
| Offline/pausa | todas | A | Offline/reduced-motion | Nenhuma animação |
| Reação ao mouse (parallax) | corpo e módulos | A/B | Site Hero | No máximo ±8–12° **simulados**; não são vistas reais |
| Aceno/gesto de cumprimento | corpo/cabeça | B | Interação | Sem braços humanos inventados |
| Giro 45°–90° com anatomia real | ângulo de câmera | C | Site interativo | Precisa de novas vistas consistentes |
| Giro 360° completo | todos os volumes | C | Site, on demand | 16–24 imagens, não espelhamento da frente |
| Rolagem contínua de câmera livre | geometria inteira | D | Não planejado | Exigiria rig/mesh e textura 3D mais completos |

## 2. Biblioteca idle e prioridade

Continuar com os 12 IDs em [RETRO_GAME_IDLE_DIRECTION.md](RETRO_GAME_IDLE_DIRECTION.md):

**P0 — seis ações:** `idle_hover`, `idle_blink`, `idle_glance`, `idle_orbit_breathe`, `idle_head_tilt`, `idle_fin_twitch`.

**P1 — quatro ações:** `idle_module_inspect`, `idle_orbit_shuffle`, `idle_curiosity`, `idle_micro_stretch`.

**P2 — duas ações:** `idle_drowse`, `idle_wake`.

**O que está funcional hoje:** protótipo HTML P0 do Hero híbrido, **não** as 12 ações no HUD. Inclinação no P0 é do corpo inteiro e aleta só tem oscilação de reflexo. Os seis primeiros nomes não significam seis masters artísticos prontos.

### Diretor de microações (`IdleDirector`)

1. Apenas `idle` real permite sorteio de microações. As ações visuais não expandem estados do Core.
2. Uma ação especial por vez; no máximo duas ações especiais por minuto, com intervalo variável/cooldown por tipo e seed de sessão em testes.
3. Trabalho/atenção/crítico/offline interrompem qualquer idle e restauram a pose canônica.
4. Não ativar se janela está oculta, no tray, fora do viewport, ou `prefers-reduced-motion`/preferência do usuário está desligado.
5. Para `idle_drowse`, usar tempo local de inatividade, nunca afirmar que o agente está offline.
6. Limpar timers/RAF ao desmontar; não sobrepor Maestro, não invocar LLM, provider nem novo IPC.

## 3. Arquitetura recomendada para Nexus e site

**Nexus HUD (64–128px):** sprites raster Compact, opticamente redesenhados, em atlas pequeno (8–12fps nos gestos). Tempo todo: uma textura estática/loop leve; dormir e pausar quando oculto. Evitar carregar 16/24 frames Hero.

**Nexus-Site:** os PNGs fiéis são o master visual Hero. O site pode usar HTML/CSS/Web Animations API para controlar camadas; `PixiJS` somente se a quantidade de sprites, atlas, batching e interações justificarem. Não introduzir WebGL, Rive ou Spine como dependência apenas para pequenas piscadas e deslocamentos. Para o giro 360°, usar frames bidimensionais reais, movidos por drag, clique/teclado ou scroll, com preload somente das vistas vizinhas.

**Fases de integração:** preservar arte aprovada → integrar raster fiel no laboratório QA → homologar seis idles → gerar novas vistas → ativar giro sob demanda → comparar LCP/CLS/FPS mobile → promover Hero público.

## 4. Atlas real de rotação: contrato

**Primeiro aceite técnico:** 16 vistas distintas em passos de **22,5°**, resolução artística inicial **512×512 RGBA**, mesmo centro/pivô, transparência e iluminação. A troca deve manter olhos, contorno, módulos, encaixe das aletas e oclusão correta entre frente, 3/4, perfil e costas.

**Alternativa:** 24 vistas em passos de **15°** se 16 parecerem saltadas, sem mudar o contrato do player. Também é aceitável testar oito vistas para composição **antes** da versão final, mas oito vistas não habilitam o player final.

**Tamanho descomprimido de referência:** 16 quadros RGBA 512² ocupam cerca de 16 MiB de pixel data bruto; 24 ocupam cerca de 24 MiB. Transferência de arquivo depende do conteúdo e codec. Carregar inicialmente apenas a vista 0°; demais sob interação, com duas vizinhas pré-carregadas; não fazer 360° contínuo em autoplay.

**Arquivos de produção:**
- `/nex/turntable/manifest.json` com schema 1.0, `view_count=16`, 16 `yaw_degrees`, SHA-256, tamanho, pivô e `status=APPROVED` **somente após revisão humana e presença dos 16 binários**.
- `/nex/turntable/frames/yaw-000.webp` até `yaw-337.5.webp` (nomes são exemplos; manifesto autoritativo).
- Todos os arquivos com mesma referência de personagem e *yaw* inequívoco, sem inventar partes ocultas ou copiar a frente como costa.
- Teste de regressão em quatro pontos cardeais 0°/90°/180°/270° e enquadramento invariável.
- Sem os binários, manifest `PENDING_ARTWORK` com caminhos nulos, e player se recusa corretamente a animar.

**Projeto de implementação:** `IAPro-Community/Nexus-Site`, `public/nex/turntable/player.mjs`, `public/nex/turntable/manifest.json`, `src/pages/nex-turntable.astro`, `scripts/nex-turntable.node-test.mjs`, `scripts/verify-nex-turntable.mjs`.

### Origem das vistas

A opção mais fiel e controlável é desenhar as vistas diretamente sobre **um guia de volumes**, mantendo a identidade aprovada. Como alternativa de produção, pode-se criar um modelo 3D *somente no pipeline de autoria*, gerar os 16 PNGs e retocá-los para o acabamento pixel art. Isso **não** torna o site ou o HUD 3D: o runtime continua exibindo imagens 2D. O modelo 3D de autoria não pode substituir o original sem aprovação de fidelidade.

**Não utilizar** `rotateY(360deg)` do PNG de frente como rotação real: no meio do efeito aparecem o visor distorcido e depois a mesma frente espelhada, jamais uma vista traseira legítima.

## 5. Gates e responsabilidade

**Não afirmar "entregue" enquanto:** PNGs da v1.1 não tiverem sido commitados; o laboratório Astro não executar com eles; não houver revisão visual da piscada; as vistas do 360° estiverem ausentes; o HUD não tiver sido homologado em Windows/macOS/Linux.

### Referências

- [Personagem selecionado](identity-lock.json)
- [Hero raster em camadas v1.1](hybrid-v1/README.md)
- [QA Chromium](hybrid-v1/BROWSER_QA_20261028.md)
- [Giro 360° planejado na issue #129](https://github.com/IAPro-Community/nexus/issues/129)
- [PixiJS Spritesheet](https://pixijs.com/)
- [Rive](https://rive.app/) — útil se houver migração de rig para estado visual; exige avaliar licenciamento/ciclo de edição.
- [Spine Runtimes](https://esotericsoftware.com/spine-runtimes) — mais específico para rig skeletal, com requisitos próprios de licenciamento; não necessário agora.

**Decisão para v1:** CSS/WAAPI + raster fiel + atlas 2D; sem motor 3D em execução, sem dependências visuais pesadas e sem alterar o Core.

# NEX — Direção escolhida: Retro Game Deluxe + biblioteca de idles

**Decisão do proprietário (27/10/2026):** manter como personagem principal **o robô da prancha NEX enviada pelo usuário nesta conversa** (capacete branco arredondado, visor largo azul-preto, duas aletas técnicas, duas orelhas/acoplamentos, núcleo azul no peito, quatro módulos esféricos separados e anel de estabilização). O **tratamento de produção** desejado é **pixel art de game retrô com alto detalhamento e muitas ações de idle**.

**Status:** `CHARACTER_SELECTED_PIXEL_MASTER_PENDING`. A escolha de **personagem** está feita, mas **ainda não** existe sprite master em camadas homologado, nem atlas de animações na release. Não congelar `identity-lock.json` como `FROZEN` sem os binários, hashes e aprovação do master final. Essa escolha **substitui explorações de personagem X-Fold/Flux/N-Core como candidatas principais**, sem apagar seu histórico.

## Novas pranchas de apoio (27/10/2026)

Criei um **pacote visual de dez imagens separadas** para consulta de outros agentes e para orientar desenho dos sprites: personagem principal, turnaround, expressões, anatomia, catálogo de 12 idles, cinco estados operacionais, ações ilustrativas, contextos de uso, paleta e prancha original integral do novo estudo. Consulte [inventário e proveniência](reference-pack/README.md) e [SHA-256 de cada recorte](reference-pack/manifest.json).

**Estas imagens não são o arquivo original selecionado pelo usuário:** foram geradas posteriormente e os nove recortes vêm de **uma única composição**. A autoridade do personagem continua em `identity-lock.json.selection_record`. O pacote está marcado **`PENDING_BINARY_INGEST`** enquanto os PNGs não estiverem commitados; não divulgar a referência como master em sprites, atlas de animação ou assets que já existem no runtime. Para incorporar os arquivos originais do pacote, usar o importador com validação de hash documentado na pasta.

## Referência realmente selecionada

- Original enviado nesta conversa: imagem PNG 1536 × 1024, 2.130.804 bytes, SHA-256 `08dcb5446a1f12a34c913104e2f9991be8b94595e1e6ccb76d1e5e2c0b351b66`.
- A prancha é **referência visual editorial selecionada**, não uma sprite sheet nem o master pixelado. O arquivo enviado não está versionado aqui; a verificação e os campos de proveniência constam do `identity-lock.json`.
- O que preservar: silhouette de cabeça ampla, formato e posição dos olhos em visor, aletas verticais, acoplamentos, quantidade/distribuição dos módulos, torso curto, núcleo central, anel inferior e proporcionalidade relativa.
- O que reinterpretar: materiais, textura e reflexos passam para **pixel art desenhada** com sombras e highlights em blocos, sem aparência de render 3D amaciado ou filtro de pixelização.
- Aparência base **frontal** para o HUD, com vistas auxiliares 3/4/perfil/costas consistentes. O personagem não vira outro robô ao trocar ângulo ou estado.

## Linguagem visual final proposta

**High-detail 32-bit retro-game sprite**, explicitamente diferente de ilustração vetorial lisa e diferente de fotografia 3D:

- Master artístico em grade **256×256 ou 320×320**, PNG RGBA, com arquivo de trabalho **editável por camada** (`.aseprite`, `.ora` ou equivalente aberto) e mapa JSON de pivôs.
- Pixel clusters intencionais, contorno controlado de 1–3px em escala do master, `4–7` tons por material principal, sombra fria navy, metal branco com azuis baixos e ciano saturado apenas em áreas emissivas.
- Dithering comedido em sombreados de metal, brilho pixelado escalonado no núcleo e nos olhos; anel de energia pode ter partículas de 1–3px. **Não** usar blur sobre o corpo inteiro, soft-vector antialias, efeito CRT obrigatório, falso pixelation filter, ruído indiscriminado ou arte nova gerada separadamente para cada frame.
- O NEX não recebe membros humanos ou boca; expressões ocorrem no visor, inclinando a cabeça e orbitando módulos, sempre retornando à silhueta inicial.
- A cena de publicidade pode ter glow e fundo, mas o **asset base é transparente**, sem cenário, texto, logotipo do produto ou marcas gravadas permanentemente no capacete.

## Escalas derivadas — não ampliar indiscriminadamente

| Uso | Canvas orientativo | Exigência |
| --- | --- | --- |
| HUD Compact | **64×64** e teste também 48px / 92px CSS | versão redesenhada opticamente, olhos e núcleo legíveis e quatro módulos identificáveis |
| Standard | **128×128** | clusters mais ricos, visor expressivo; não reduzir puro Hero |
| Hero | **256×256 ou 320×320** (export 2×/4× em nearest-neighbor) | detalhe máximo de metal, reflexos e silhueta sem perder leitura em 64px |
| Micro | **32×32** | variante simplificada ou somente símbolo reconhecível; não prometer quatro módulos distintos se não forem legíveis |

A estética “32 bits” é uma **direção artística**, não profundidade de cor do PNG. O uso de paleta RGBA completa é permitido.

## Biblioteca obrigatória de idle

Todas as ações abaixo são **comportamentos não operacionais** que acontecem somente com `idle` real, nunca simulam execução, conclusão, sucesso, erro ou análise. Suas expressões devem manter o personagem do mesmo master e todas as quatro peças independentes.

| ID | Ação | Ciclo visual | Frequência / condição alvo | Primeiro pacote |
| --- | --- | --- | --- | --- |
| `idle_hover` | Respiração e flutuação | Corpo sobe/desce 1–3px, sombra responde, anel baixa intensidade | loop 3–5s | P0 |
| `idle_blink` | Piscada simples ou dupla | Olhos fecham em 2–3 frames, retornam sem mudar formato do visor | a cada 4–9s | P0 |
| `idle_glance` | Olhar para os lados | Pupilas/olhos se deslocam ou expressam curiosidade; sem girar corpo todo | espaçado, 12–25s | P0 |
| `idle_orbit_breathe` | Órbita viva | Quatro módulos oscilam em fases independentes e retornam ao pivô, partículas sutis | contínuo leve, 4–7s | P0 |
| `idle_head_tilt` | Inclinar a cabeça | Cabeça inclina 2–4°, olhos acompanham; corpo permanece frontal | raro, 25–50s | P0 |
| `idle_fin_twitch` | Microajuste das aletas | Aletas espelham um movimento curto e assimétrico para depois centralizar | raro, 18–40s | P0 |
| `idle_module_inspect` | Inspecionar módulo | Um módulo aproxima-se até 5–7px; olhar o segue e retorna | raro, 45–90s | P1 |
| `idle_orbit_shuffle` | Reorganização curta | Módulos seguem arcos limitados, sem trocarem de identidade/ordem | raro, 60–110s | P1 |
| `idle_curiosity` | Curiosidade | Olhos ganham expressão curiosa e cabeça faz leve inclinação | raro, 35–75s | P1 |
| `idle_micro_stretch` | Espreguiçada tecnológica | Corpo se alonga sutilmente, aletas abrem alguns pixels e retornam | muito raro, 90–180s | P1 |
| `idle_drowse` | Sonolência de espera | Olhos caem por 2–4s, núcleo fica baixo, sem `offline` falso | só após >3min sem interação | P2 |
| `idle_wake` | Acordar | Olhos reabrem, núcleo volta ao nível normal | após `idle_drowse` ou interação | P2 |

**Regra de coreografia:** no máximo uma ação especial sobre a base `idle_hover`/órbita leve por vez. Evitar cabeçadas, flashes, loops hiperativos ou grandes rotações, especialmente durante leitura de código. 1–2 ações especiais por minuto é teto de experiência, não obrigação de executar.

### Agendador de microações

- Implementar uma única pequena máquina visual local `IdleDirector` **sem dependências do Core, do Maestro nem chamadas a LLM**; estado é `idle` somente enquanto a projeção real permitir.
- Intervalos **não mecânicos** escolhidos por PRNG local com seed de sessão, mas reprodutíveis em testes por seed injetável; cooldown por ação e limite global anti-repetição.
- `working`/`warning`/`critical`/`offline` **interrompem imediatamente** qualquer idle e restauram pose neutra sem teleportar peças. O retorno deve ser suave e determinístico.
- Pausar totalmente quando HUD não estiver visível, janela oculta/tray, `document.visibilityState=hidden`, sistema com redução de movimento ou preferência `motion=off`.
- Economizar GPU/CPU: atlas compartilhado, sem polling, sem timers duplicados por mount, sem WebGL obrigatório; considerar loop visual 8–12fps e animação em camadas para 92px HUD.
- Nunca usar cor laranja/vermelha durante idle como se fosse atenção real; usar ciano da identidade e neutros apenas.
- Sem áudio por padrão, sem alteração da área clicável, hit region ou janela nativa.

## Regras de produção dos frames

1. **Uma arte base em camadas, reutilizada:** helmet, visor/máscara, cada aleta, duas orelhas, torso, núcleo, módulos 1–4, anel dianteiro/traseiro, sombra e partículas.
2. Cada sequência tem `action_id`, `frames`, `fps`, `loop`, `anchor`, `bounds`, `cancel_policy`, `cooldown` e `return_to_idle` em manifesto; não gerar frames independentes por IA.
3. Entre 6–16 quadros por microação conforme complexidade, com pivôs fixos e sem jitter subpixel; referência sugerida 12fps para idles ricos, mantendo CSS apenas onde não comprometer pixel precision.
4. Entregar atlas com PNG transparente e JSON estruturado + source editável + preview GIF/APNG/WebP **somente como demonstração**, não como master obrigatório.
5. Para cada ação, registrar before/after em tamanho HUD real; requer aprovação humana com comparativo ao PNG de referência selecionado.

## Hero SVG alternativo para animação do site

Um asset vetorial de grande resolução, `web/public/nex/nex-site-hero.svg`, foi criado como **candidato animável** para o site. Ele permite idle, olhos, núcleo, aletas e quatro módulos independentes, mas mantém frontalidade e **não** equivale ao master artístico em pixel art HD (nem ao atlas 360°). A documentação [SVG Hero + pseudo-3D + giro real](SITE_HERO_SVG_AND_360_MOTION.md) separa implementação atual de metas de arte futura; o master retrô da issue #119 continua pendente.

## Fases e critérios de pronto

- **P0 — identidade de game:** master frontal Hero/Standard/Compact do mesmo personagem, paleta, silhueta, export sem fundo e seis primeiros idles testáveis no HUD, todos sem movimento quando solicitado.
- **P1 — personalidade:** quatro interações idles adicionais, emissões/microexpressões, sorteio com cooldown, preempção por evento real, revisão em Wayland/X11/Windows/macOS.
- **P2 — descanso e retomada:** `idle_drowse` e `idle_wake`; variantes de tema/site e exports em maior resolução se a base estiver homologada.

**Bloqueadores para afirmar “implementado”:** master editável e atlas real versionados, hashes no `identity-lock.json`, teste de pixel fidelity, execução da prévia, comparação visual humana e smoke do HUD nativo. O atual `NexPixelSprite.tsx` e animações CSS são **fallback temporário do runtime**; não são a versão retro HD pixel art final.

## Governança

- [Identidade / freeze](IDENTITY_GOVERNANCE.md)
- [Lock machine-readable](identity-lock.json)
- [Contrato visual anterior](../NEX_32BIT_VISUAL_CONTRACT.md)
- [Animações presentes em código](../NEX_ANIMATION_CONTRACT.md)

A seleção deste personagem **não autoriza declarar os binários de v1.0.0 homologados** nem alterar o logo institucional do Nexus.

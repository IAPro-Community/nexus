# NEX — Identidade visual oficial do mascote IAPro Nexus

**Decisão de design aprovada:** 23/10/2026  
**Produto:** IAPro Nexus, linha 1.0 (arte oficial no source, não equivalente a binário homologado)  
**Estética:** pixel art retrofuturista *32-bit-inspired*; pose **frontal** como padrão.  
**Mascote:** NEX. **Marca institucional:** logotipo Nexus permanece independente.

## Personagem canônico

NEX é um robô flutuante, simpático sem ser infantil, de silhueta simétrica e cabeça dominante.

1. Cabeça grande e arredondada em metal branco-frio com contorno navy, sempre frontal na pose neutra.
2. Um visor escuro arredondado, com olhos emissivos azuis que comunicam os estados operacionais.
3. Duas aletas verticais laterais e dois acoplamentos circulares azuis nas orelhas.
4. Corpo compacto, flutuante, **um** núcleo circular de energia no centro.
5. **Quatro módulos magnéticos destacados**: dois laterais e dois inferiores. Não converter em braços/pernas humanos.
6. Uma órbita/anel energético ciano-azul na zona inferior, sem cruzar rosto e visor.
7. Elementos de pixel shading claros, reflexos discretos, sem backgrounds presos no asset ou transparências falsas.

Os mesmos olhos, proporções, módulos, cores e perfil devem permanecer constantes entre estados, temas e superfícies. Vistas 3/4, perfil e costas são **apoio** para motion/marketing, não pose padrão do HUD.

## Identidade versus implementação

| Artefato | Status de decisão | Fonte e finalidade |
| --- | --- | --- |
| NEX Compact | **Arte oficial da v1 em source** | `web/src/features/hud/NexPixelSprite.tsx` — SVG em grade 128x128, exibido em ~92px no HUD |
| NEX Standard | **Estilo oficial, master específico ainda não publicado** | versão futura 128–256px otimizada para interface, onboarding e docs |
| NEX Hero | **Estilo oficial, master específico ainda não publicado** | versão futura 256px+ para site e marketing |
| Ilustrações conceituais | Referências aprovadas de linguagem, **não sprites exportáveis** | pranchas do processo de design |
| Logo/ícone Nexus | Marca institucional preservada | `assets/brand/`, `web/public/nexus-icon.png` |

A expressão “32 bits” é uma referência à estética dos jogos, **não** uma imposição de profundidade de cor, nem um limite de resolução. Não ampliar o sprite Compact e chamar de versão Hero: a versão Hero exige desenho óptico específico, consistente com o mesmo personagem.

## Paleta

- Navy profundo: `#0A1830` / superfície do visor `#030B21`.
- Azul Nexus institucional: `#3B72E1`.
- Ciano emissivo: `#31D5FF`; highlights `#DFFFFF`.
- Branco metálico: `#F2F8FF`, sombras de aço `#AAC4E2`.
- Alerta comprovado: `#FF8A45`; offline: cinza-azulado `#71839D`.

Não introduzir uma nova identidade visual substituindo a marca principal, nem permitir skins de provedores alterarem a anatomia de NEX.

## Contrato operacional da v1

O *Guardian* permanece um **modo de apresentação do HUD**. NEX não é um agente, não executa tarefas nem determina routing. Estados honestos derivados da projeção canônica:

- `idle`: olhos contentes, flutuação sutil, pulsação lenta do núcleo.
- `working`: olhos concentrados, núcleo e módulos com pulso mais rápido.
- `warning`: olhos laranja, gesto curto e balão de intervenção baseado em sinal real.
- `offline`: brilho reduzido, olhos neutros, **sem loop que simule atividade**.

As animações CSS são de apresentação, não barras de progresso de execução; o estado `verified` não é inferido de `working`. Estados como `thinking`, `routing` e `success` somente poderão ser ligados a telemetria própria futuramente.

A implementação atual usa componentes visuais CSS/SVG, sem WebGL ou rede; favorece `prefers-reduced-motion: reduce`, renderização pixelada e mantém o texto acessível no controle pai. O HUD conserva BAR/MASCOT/TRAY, ponte nativa, preferências e geometria.

## Governança da arte e consistência

- `NexPixelSprite.tsx` é a fonte editável de NEX Compact, inclusive as variações por mood. Ajustes de anatomia devem atualizar testes.
- Criações futuras de Standard/Hero partem das **proporções e silhueta acima**; não gerar cada estado independentemente e publicar como arte final.
- Preserve um atlas ou master em camadas se o Compact migrar de SVG para PNG/sprite sheets; pivôs, coordenadas e quantidade de módulos não variam.
- Os ícones de sistema operacional, tray e favicon **continuam exibindo o símbolo institucional do Nexus**, não uma cabeça pixelada pequena e ilegível.
- Nos sites, a identidade visual pode ser mais rica, mas a pose institucional de NEX é frontal.

## Gates de homologação (independentes da aprovação visual)

O visual foi **aprovado como direção de produto**, mas a release binária **não está automaticamente homologada**.

1. `bun run --cwd web quality:full` e build frontend no mesmo SHA.
2. HUD nativo Windows/macOS/Linux (X11 e Wayland), transparência real, hit testing, alça de arraste e troca BAR↔MASCOT↔TRAY.
3. Escalas DPI 100%, 125% e 150%: legibilidade, recorte, aliasing, dimensões ~108x112 DIP.
4. Estados idle/working/warning/offline e `prefers-reduced-motion`: ausência de flicker, loops artificiais ou falsos alertas.
5. Documentação de evidência, SHA, artefatos e rollback. **Sem certificar Windows/macOS/Linux por inspeção estática de source.**

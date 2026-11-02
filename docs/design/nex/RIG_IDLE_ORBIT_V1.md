# NEX Rig v0.1 — primeiro corte: Idle premium + Orbit

**Status:** IMPLEMENTAÇÃO DE PROTÓTIPO, não master aprovado. Não integrar no HUD público nem alterar o gate de release.

## Referências e integridade

- Seleção visual do usuário: prancha `Guia Visual Futurista do Mascote NEX.png`, 1536×1024, SHA256 `08dcb5446a1f12a34c913104e2f9991be8b94595e1e6ccb76d1e5e2c0b351b66`. Não está versionada em Git como fonte binária.
- Fonte usada **provisoriamente** neste protótipo: `web/public/nex/hybrid-v1/nex_source_original.png`, 1254×1254, SHA256 `ef7a4aeb7b24d2ab5f199f913d351414b8fafb0b6208b3ab2b665013adcb9118`. Versão detalhada, mas **não é a prancha oficial**; issue #174 bloqueia promoção.
- Não utilizar os SVGs antigos 128×128 como substituto visual.
- Rive não está integrado porque não há rig editável nem arquivo `.riv` real. Não criar host que apenas falha para um arquivo inexistente. Esta fase usa Canvas2D sem dependência, e o runtime fica isolado para futura migração para Rive ou Spine.

## Código entregue

```text
web/public/nex/rig-v1/
  index.html            # laboratório isolado, 2 botões, comparação de PNG
  nex-motion-core.mjs    # funções puras: idle, blink e órbita
  nex-rig-player.mjs     # renderer Canvas 2D e lifecycle
web/scripts/nex-rig-v1.node-test.mjs
```

O Nexus-Site mantém uma **cópia exata** dos três arquivos em `public/nex/rig-v1/` para o laboratório de staging. A arte binária está em `public/nex/hybrid-v1/`, sincronizada por manifesto SHA. Qualquer alteração no core/player precisa sincronizar ambas as cópias; preferir futuramente empacotar a biblioteca comum para remover o espelhamento.

## Rig atual baseado em pixels, não em promessas

| Camada | Estado | Coordenada/pivô |
| --- | --- | --- |
| `body_base.png` | presente, contém cabeça + corpo + anel, **não articulável** | canvas 1254×1254 |
| `eye_left.png`, `eye_right.png` | presente, independente; blink por escala de Y e look por deslocamento | manifest do híbrido |
| `module_1..4.png` | presente, cada módulo independente; órbita e perspectiva simples | manifest do híbrido |
| `fin_left.png`, `fin_right.png` | presente, mas apenas reflexos, **não** aletas completas | manifest do híbrido |
| Núcleo | ainda embutido no `body_base.png`; apenas halo aditivo externo | aprox. (627,869) |
| Anel energético | embutido no `body_base.png`; ainda não acompanha a órbita | arte estática |

A animação de órbita usa uma elipse central `(627,800)`, raios `(446,186)`, ciclo ~6,4 s, quatro fases, perspectiva de escala/alpha e ordenação real traseiro/corpo/dianteiro. Não é 360° do robô, nem rotação física dos módulos em 3D. O idle combina hover, dois períodos de balanço, blink curto simples/duplo, eye-tracking dentro do visor e core-glow discreto.

Em dispositivos com movimento reduzido, abas ocultas e preview desativado, mostra-se **o PNG original byte-fiel**, sem reconstituir novos pixels. O renderer pausa o `requestAnimationFrame` quando está fora da viewport. A lógica operacional do Nexus nunca é consultada pelo laboratório.

## Separação de arte para o Rive profissional (próxima etapa)

É tecnicamente incorreto apenas recortar os pixels visíveis e chamar isso de rig. Para uma versão realmente articulável devem existir regiões ocultas reconstruídas e fonte editável PSD/ORA. Ordem sugerida:

1. **Fundo:** halo traseiro, metade traseira do anel, sombra, partículas.
2. **Torso:** chassis atrás, corpo frontal, pescoço/encaixe, núcleo e glow em camadas.
3. **Cabeça:** capacete traseiro, frontal, visor negro base, moldura de vidro, brilho, 2 olhos, 2 pálpebras, 2 acoplamentos, 2 aletas inteiras, detalhes LED — pivô no encaixe do pescoço.
4. **Orbitais:** 4 corpos de módulo com carcaça, olho/LED, brilho, máscara/oclusão independente — pivôs definidos pelo eixo central da peça.
5. **Energia:** anel traseiro e frontal, flashes auxiliares, sombra.
6. **Transformações:** hierarquia `root > torso > neck/head/eyes/fins`; orbitais ficam fora do pai `head` para não herdar inclinação de cabeça; olhos herdam pose de cabeça; manter máscaras de visor.
7. **Art direction:** não adicionar braços/pernas, substituir cor/forma, mudar quatro módulos ou construir uma geometria 3D falsa. Gravar SHA do novo master depois de aprovação humana.

## QA

```bash
cd web
npm run test:nex:rig
# Para revisão manual com um servidor estático que inclua web/public:
# abrir /nex/rig-v1/ (não via file:// — fetch e módulos requerem HTTP)
```

**Gates de aceitação nesta fase:**

- [x] módulo em quatro trilhas elípticas, profundidade frente/trás e âncoras estáveis
- [x] idle + blink e foco de olhar no visor
- [x] isolamento visual sem segunda máquina de estados operacional
- [x] fallback original em redução de movimento ou falha de renderer
- [ ] browser QA com screenshots 390/768/1280px e nenhuma falha de asset
- [ ] percepção/art direction humana da órbita e aparência
- [ ] rig de cabeça, pescoço, corpo e aletas individualizados
- [ ] master editável pixel-art de alta fidelidade aprovado e `.riv` real
- [ ] integração/QA nativo HUD Wails

**Não declarar os checkboxes não homologados como concluídos.**

## Adendo v0.2 — polimento incremental (2026-11-01)

- Microgestos independentes em `idleModulePose(index, t)`: quatro módulos conservam personalidade própria, sem sincronização robótica nem exceder ~4,4px da arte de 1254px.
- `orbitFrontWeight(z, blend)`: transição em `smoothstep` entre passes de composição traseiro e frontal; remove a troca discreta de `z-index` durante o crossover e no retorno ao idle.
- Renderer continua um único Canvas2D com fonte pixel-original e um RAF pausável; nada de novos `setInterval`, LLM, modelo de status ou efeito global.
- Modo QA opcional **Mostrar pivôs e órbita** desenha elipse e IDs dos quatro módulos sem alterar arquivos PNG nem manter HUD telemetry.
- Testes acrescentados para continuidade de peso de oclusão, âncoras estáticas e microdrifts independentes.
- **Limite artístico permanece**: o Canvas faz `crossfade` em cruzamentos; isso pode apresentar semi-transparência transitória. Com master/rig final editável, substituir por máscaras de oclusão ou malha para manter opacidade física sem pop. Não certificar como render físico 3D.
- **Não** implementa nova articulação cabeça/torso/aletas porque a fonte atual não contém peças completas nem regiões ocultas restauradas. Separação correta exige novo master supervisionado, bloqueada por #174.

### Testes / revisão pendente
Rodar `npm run test:nex:rig`, conferir PR do Nexus-Site com deploy e comparar fisicamente as capturas ao master. Testar frame crossing a 25/50/75% da órbita nos modos 390px, 768px e 1280px; verificar reduced-motion, aba suspensa, contexto Canvas perdido e native Wails.

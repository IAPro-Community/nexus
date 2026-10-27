# Nex Motion Lab (experimental)

Prova de conceito **isolada** de um mascote Nex com geometria fluida em forma de X, carcaça branca e visor robótico. **Não substitui** o componente oficial `src/features/hud/NexPixelSprite.tsx` e não é distribuída no HUD/release por padrão.

## Executar

Abra `index.html` diretamente em um navegador moderno; não há servidor, build nem dependência de terceiros. Opcionalmente, a partir da raiz do repositório:

```bash
python3 -m http.server 8765 --directory web/experiments/nex-motion-lab
# http://localhost:8765/
```

## Experimentos

- Estados: ocioso, curioso, pensando, trabalhando, apontando, espiando, sucesso, atenção e descansando;
- Um contorno vetorial único gerado a cada frame por funções radiais e curvas quadráticas, interpolando comprimentos, ângulos e proporções dos quatro lobos em X;
- Interpolação amortecida (mola), visor e olhos animados, gesto para a janela simulada;
- Controles de sequência, elasticidade, movimento reduzido e seleção de estados;
- `requestAnimationFrame` pausado logicamente quando a aba está oculta.

## Limites e critérios para adoção

Protótipo de **comportamento**, não identidade visual aprovada nem renderer de produção. Os estados e a telemetria são sintéticos; nenhuma integração com Maestro, runtimes ou agentes. Não confundir com as moods canônicas do HUD (`idle | working | warning | critical | offline`).

Antes da integração, são necessários: design canônico aprovado; testes cross-browser e mobile; medição de CPU, memória e frame pacing em máquinas modestas; testes automatizados de acessibilidade e movimento reduzido; um adaptador de eventos para o HUD **sem duplicar** sua máquina de estados; testes de convivência com terminal/TUI; degradação para SVG estático.

Para avaliar, compare especialmente o trajeto **Ocioso → Apontando → Espiando → Descansando → Ocioso**, verificando continuidade de silhueta e legibilidade do visor.

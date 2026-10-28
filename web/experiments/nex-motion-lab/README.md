# Nex Motion Lab — experimento v0.2

Protótipo **isolado** da metamorfose do Nex: um robô branco com visor escuro, em forma-base **X**, cuja geometria contínua é deformada a cada frame. Não substitui o `NexPixelSprite` do HUD; não integra Maestro nem recebe eventos reais de agentes.

## Abrir

Abra [index.html](./index.html) em um navegador moderno, sem dependências de execução, ou sirva a pasta localmente:

```sh
python3 -m http.server 8765 --directory web/experiments/nex-motion-lab
# abrir http://localhost:8765/
```

## O que avaliar

1. Use os **nove estados** da barra lateral. Compare Ocioso → Apontando → Espiando → Descansando → Ocioso.
2. **Arraste uma das extremidades** do Nex e solte: o contorno se expande radialmente, permanece contínuo e retorna elasticamente à forma original.
3. Pressione **Enter/Espaço** com o canvas focado ou clique no personagem.
4. No estado **Apontando**, uma extremidade mira a borda do botão da janela simulada, usando sua posição efetiva no layout (não um alvo fixo). Redimensione a janela para comparar.
5. Ative **Reduzir movimento**: estados passam a mudanças estáticas, sem autoplay ou loop permanente de renderização. Alterações da preferência de acessibilidade do sistema são observadas.
6. Compare no mobile (390px) e verifique o desempenho; renderização limitada a cerca de 30fps no mobile e 60fps no desktop.

## Teste de fumaça automatizado

Requer `playwright` e Chromium do ambiente de desenvolvimento do Nexus:

```sh
bun install --cwd web --frozen-lockfile
cd web
bunx playwright install chromium
node experiments/nex-motion-lab/smoke.mjs
```

O teste cobre seleção dos estados, teclado, captura de arrasto, modo de movimento reduzido e layout mobile. O resultado depende da disponibilidade do browser no ambiente de execução.

## Arquitetura e limites

A demo usa HTML, CSS e Canvas 2D sem dependências de runtime. Um contorno paramétrico único é gerado por função radial, com interpolação amortecida dos parâmetros, deformação local por gesto e visor independente. **Não é uma simulação física de líquidos nem arte final aprovada.** Os movimentos e os eventos são sintéticos; nenhuma ação do Nexus real é executada.

O HUD atual usa as moods canônicas `idle | working | warning | critical | offline`. Uma integração futura exigiria um adaptador de **apresentação** desses eventos, mantendo a origem da verdade no HUD e sem criar uma segunda máquina de estados de negócio. As ações do personagem devem ser opt-in e não roubar o foco do terminal.

### Ainda necessário para integração
- Aprovação da silhueta e do acabamento visual em tamanhos de 32px, 64px e grandes.
- Testes de desempenho em dispositivos reais, profiling de CPU e GPU e budget de recursos.
- Testes cross-browser (Chromium/WebKit/Firefox) e de leitores de tela.
- Política de prioridade e frequência para sugestões, tratamento de foco e múltiplas janelas.
- Estado inativo que reduza a atividade de renderização, fallback estático e suporte a preferência de movimento reduzido.
- Testes E2E de não regressão com o HUD/terminal nativo, antes de integrar à release.

**Não mesclar na release enquanto a linguagem de movimento e identidade visual não forem aprovadas.**

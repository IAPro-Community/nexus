# Nexus Companion — HUD opcional 2D (versão 1.1)

> Status: implementação candidata no PR #55. Não homologada em Windows, Linux ou macOS.
> Base: `release/v1.0.0`. A release 1.0 permanece com a Barra como padrão.

## Objetivo

O usuário escolhe **Barra** (experiência compacta existente) ou **Mascote 2D**.
Ambas usam a **mesma** projeção autenticada do Core. O personagem apenas
representa o estado, avisa sobre inputs reais e encaminha a uma ação; ele
não coleta quota, roteia modelos nem troca contas.

## Identidade visual e sprites

Direção: guardião/núcleo 2D com inspiração leve em slime; visual flat;
baseada na marca canônica Nexus: janelas/terminal `>_`, grafite
`#182234`, azul Nexus `#3b72e1`, céu `#70a4ea`, branco.
O personagem não deve ser confundido com um redesign da logomarca.

Arquivos em `web/public/companion`:
- `nexus-guardian-sprites.svg`: sprite atlas 576 × 576, **6 colunas × 6 linhas**,
  cada quadro 96 × 96.
- `nexus-task-capsule.svg`: cápsula separada para tarefas, 64 × 70.

Linhas do atlas, nesta ordem: **idle**, **working**, **attention**,
**critical**, **offline**, **success**. Cada linha possui seis quadros.
A CSS `steps(6,end)` muda apenas a posição horizontal do atlas,
sem depender de imagens 3D, canvas ou runtimes de animação externos.

Microações adicionais (tilt/hop/peek) são curtas e sorteadas com intervalo
entre 16 e 32 segundos. Não são executadas com popup ou balão aberto,
em estado crítico, ou se a opção de animação / `prefers-reduced-motion`
for desativada.

O atlas e a cápsula possuem fundos transparentes. Não colocar sombra de
janela retangular nem blur infinito no próprio personagem.

## Ações

- Clique no corpo: abre/recolhe **o popup operacional existente**, sem
  criar segunda implementação de HUD.
- Clique no balde: abre a lista de pendências do popup.
- Balão de atenção: botões **Abrir** e **Depois**; nunca abre navegador
  ou executa ação sem o usuário clicar.
- Selecione Barra/Mascote no popup ou nas configurações do Web OS.
- Arraste pela alça Wails já existente; teclado move pela mesma ponte nativa.
- O tray nativo continua permitindo ocultar/recuperar o HUD.
- Ações contextuais no Desktop abrem rotas canônicas do Nexus:
  `/p/:projectId/missions/:runId`,
  `/p/:projectId/agents/:agentId`, ou
  `/p/:projectId/resources`. Sem projectId verificado, abrem a URL
  base do Nexus, nunca uma deep link inventada.

## Fontes de atenção

`projectCompanionAttention` lê a projeção de HUD:
1. Missions **não terminais** com `needs_user=true` (prioridade crítica).
2. Agents em estados explicitamente aguardando input do usuário.
3. Alertas Capacity WARNING/CRITICAL ainda não reconhecidos e não snoozed.

As notificações são deduplicadas por ID em uma sessão, o balão fecha
automaticamente em 8,5 segundos, e “Depois” silencia o *balão* por até
uma hora no armazenamento local. Não marca a Mission concluída nem
reconhece alertas do Core. O balde continua mostrando pendências reais.

Os títulos dos balões são genéricos para reduzir vazamento acidental de
dados durante compartilhamento de tela. Detalhes somente no Nexus.

## Persistência e geometria

Preferências de apresentação: `HudPresentationPreferences` / storage
`iapro:nexus:hud:presentation:v1`; usuários antigos continuam em
**Barra**. O tamanho da Barra é preservado independentemente do mascote.

Preset nativo `companion`:
- Compacto: **224 × 158 DIP**
- Expandido: **380 × 560 DIP**
- Balão transitório: **310 × 245 DIP**, sem substituir o preset persistido

A geometria permanece limitada à área útil do monitor quando a plataforma
possui suporte a posicionamento global; Wayland pode restringi-lo.
Windows e macOS usam a configuração existente sem sombra nativa de janela.

**Importante:** o viewport temporário do balão ainda é retangular para
hit-testing em algumas plataformas. A transparência visual por pixel
não implica click-through dos pixels invisíveis. A futura separação
física de janelas (personagem e popup) é uma melhoria de arquitetura que
depende de certificação específica; não fingir que ela está pronta.

## Gates obrigatórios para promover PR #55

1. `cd web && bun install --frozen-lockfile && bun run quality:full`
2. `go test ./internal/desktop ./cmd/nexus-hud`
3. UI Web: selecionar ambos modos, popup, balões, “Depois”, lista de
   pendências, `prefers-reduced-motion`, modo offline e sem eventos.
4. Smoke Wails Windows: sprite transparente, arraste, sombra, popup, tray,
   DPI 125/150%, monitores múltiplos e navegador/contexto.
5. Smoke Linux X11/Wayland: comportamento do GTK e limites do compositor
   documentados separadamente.
6. Smoke macOS: translucidez, ausência de sombra retangular, dock,
   mudança de modo e popup.
7. Uso com 10+ contas e missões concorrentes: limite visual e deduplicação.
8. Capturas reais e logs de cada SO vinculados **ao mesmo SHA**;
   nenhum teste `NOT_TESTED` pode ser classificado como `PASS`.

A certificação geral não pode ser inferida de build Web ou dos testes
unitários; o popup Wails deve funcionar ao clique real em cada plataforma.

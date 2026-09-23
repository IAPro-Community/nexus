# ADR — Nexus HUD como projeção, não domínio paralelo

**Status:** Accepted for implementation  
**Base auditada:** `5a61d3152b23b002b39e8b4b8988924db39956b2`  
**Branch:** `feat/nexus-hud`

## Contexto

O Nexus v1 já possui autoridades canônicas para Capacity, seleção de recursos,
custo, Agents, Missions, eventos e evidência. Criar um monitor ou scheduler
específico para o HUD produziria divergência de estado, probes duplicados,
políticas inconsistentes e maior custo operacional.

O Desktop atual usa Wails v2.15.0. O shell principal consome o mesmo Core que o
Web e já prefere anexar a uma instância loopback existente.

## Decisão

Criar `HUD Projection` como read model read-only, composto a partir dos serviços
canônicos existentes.

```text
canonical services/state
        ↓
HUDApplicationService
        ↓
HUDState (DTO estável)
   ┌────┴────┐
   ↓         ↓
Web HUD   Desktop HUD
```

A projeção pode ordenar/priorizar informação para apresentação, mas não:

- coletar quota;
- avaliar thresholds;
- calcular routing alternativo;
- inferir custo ausente;
- alterar estado de Agent/Mission;
- decidir sucesso;
- publicar uma segunda família de eventos.

Ações como acknowledge/snooze continuam nos endpoints/serviços canônicos.

## Limiting window

O HUD não implementará fórmula de janela limitante. A seleção deve usar a
semântica já exposta pelo Capacity/ResourceScheduler. Se a fonte não identificar
uma janela limitante confiável, a projeção não inventa uma.

## Resource recommendation

O HUD não chama `ResourceApplicationService.List` nem executa discovery para
obter uma recomendação. Em uma transição degradada, o Capacity Monitor chama o
recommender canônico do ResourceScheduler uma única vez, publica a mesma
recomendação no event bus e a persiste no `capacity_event` durável.

`HUDApplicationService` apenas lê esse evento e exige que seu `fingerprint`
continue ativo em `WARNING`, `CRITICAL` ou `EXHAUSTED`. Recovery, ack e
snooze removem a recomendação da projeção. Ausência de evidência resulta em
ausência de recomendação, não em fallback calculado pelo HUD.

## Desktop

Wails v2 é single-window por aplicação. Para satisfazer a exigência de um gadget
real independente da janela principal sem trocar de framework, o Desktop HUD é
um shell Wails separado e leve.

Isto é uma separação de **processo de apresentação**, não uma duplicação de
domínio. O shell utiliza o mesmo Core/API. Nenhuma regra de Capacity, scheduler,
cost ou Mission é copiada para ele.

## Preferências

`HUDPresentationPreferences` permanece separado de
`CapacityMonitorPreferences`. Preferências Web que são específicas do browser
podem usar storage local versionado. Geometria/capabilities nativas pertencem ao
adapter Desktop.

## Eventos e refresh

O alvo é event-driven. Enquanto uma fonte canônica não expuser stream para todo
o read model, a revalidação deve ser única, deduplicada, pausada quando invisível
e nunca executar probes diretamente pela UI.

## Plataforma

Capabilities nativas são consultadas por um adapter único. A UI não espalha
condicionais por sistema operacional. Suporte público segue fail-closed:
`UNVERIFIED` não é `PASS`.

## Consequências

### Positivas

- uma fonte de verdade;
- Web e Desktop compartilham contrato;
- risco menor de divergência;
- possível migração futura de Wails sem alterar domínio;
- testes determinísticos da projeção.

### Trade-offs

- Wails v2 exige um processo de apresentação adicional para a janela HUD
  independente;
- algumas capacidades Wayland permanecem compositor-dependent;
- o resumo de recomendação geral não substitui routing contextual de uma
  Mission.

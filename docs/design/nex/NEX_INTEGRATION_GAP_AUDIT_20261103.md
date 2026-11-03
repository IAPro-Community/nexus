# NEX — Auditoria de lacunas para integração (2026-11-03)

**Escopo aprovado:** Companion 2D de oito comportamentos sobre cinco moods operacionais reais, catálogo técnico de doze gestos idle; **360° real permanece futuro**. A publicação é atômica HUD + Nexus-Site, com master fiel, QA real e aprovação humana no SHA exato.

**Natureza:** auditoria de fonte e PRs, não homologação de desenho ou executável. Comparação realizada com `release/v1.0.0`, PR [#193](https://github.com/IAPro-Community/nexus/pull/193), PRs de contrato [Nexus #194](https://github.com/IAPro-Community/nexus/pull/194) e [Nexus-Site #15](https://github.com/IAPro-Community/Nexus-Site/pull/15), issues [#141](https://github.com/IAPro-Community/nexus/issues/141), [#174](https://github.com/IAPro-Community/nexus/issues/174), [#119](https://github.com/IAPro-Community/nexus/issues/119) e [#120](https://github.com/IAPro-Community/nexus/issues/120).

## Estado verificado (não confundir candidate com PASS de release)

| Capacidade | Estado técnico | Evidência / limite |
|---|---|---|
| Projeção dos 5 moods | IMPLEMENTADO em código | `mascotModel.ts` e mapeamento attention→warning; UI nativa e fidelidade visual ainda requerem QA |
| Diretor de 12 IDs idle | IMPLEMENTADO em código | `nexIdleDirector.ts` e testes; animações CSS/SVG, não doze artes pixel-art individualmente homologadas |
| 8 behaviors contextualizados | IMPLEMENTADO em branch / NÃO MERGED | PR #193: `nexCompanionBehavior.ts`, `GuardianMascot.tsx`; 25 testes focados PASS; testes de 8 gestos no produto real ainda pendentes |
| Evento `COMPLETED_VERIFIED` | IMPLEMENTADO em branch / NÃO MERGED | PR #193 projeta conclusão recém-observada via Attention Center com baseline e dedupe; ainda falta E2E de atualização real, troca de projeto e retomada de sessão |
| Interação mouse/foco/click | PARCIAL | `GuardianMascot` reage a pointer/focus; QA de touch, foco, cooldown de gestos e click-through nativo ainda pendente |
| HUD Web | PARCIAL | `shouldRenderHudMascot` retorna somente para `native && nativeView==='capsule' && appearance==='MASCOT'`; Web HUD em modo cápsula **não mostra automaticamente o mascote**. Definir UX esperada e cobrir fluxo Web explicitamente |
| HUD nativo Wails | NÃO HOMOLOGADO | Não inferir Windows/macOS/Linux a partir do TypeScript Windows; testar drag real, mouse passthrough, DPI, tray, redução de motion, transparência |
| Master pixel-art aprovado | BLOCKED_ART | PNG selecionado 1536×1024 SHA-256 `08dcb5446a1f12a34c913104e2f9991be8b94595e1e6ccb76d1e5e2c0b351b66` não está versionado; raster 1254×1254 no repositório é **candidato diferente**; não reimaginar mascote |
| Rig independente + variantes ópticas | BLOCKED_ART | Master em camadas preciso, underpainting de partes ocultas, pivôs, atlas 64/92/128px e Hero, hashes e comparação lado a lado ainda não aprovados |
| Site laboratório de 8 behaviors | PARCIAL / NÃO MERGED | `main` possui `/nex-lab` noindex, mas seus assets podem ficar indisponíveis pelo gate hash; branch `feat/nex-companion-lab-20261101` implementa controles ainda não integrados; antiga PR Site #11 encerrada **sem merge** |
| Publication gates 2D | EM PR DRAFT | Nexus #194 e Nexus-Site #15 alinham os 11 gates, preservam BLOCKED/0 PASSED; não promover antes de checks e revisão |
| Qualidade Web (PR #193) | TESTADA PARCIALMENTE | GitHub workflow 41628946757: 784 testes em 125 arquivos, build, lint e 10 capturas Web sintéticas PASS; **capturas genéricas não certificam todos os gestos do mascote** |
| Giro 360° verdadeiro | DEFERRED | Issue #129 permanece aberta; não é gate do MVP nem pode ser apresentado como implementado |

## Sequência recomendada

### P0 — Consolidar contratos e paridade visual

1. Revisar Nexus #194 e Nexus-Site #15 como par; garantir `BLOCKED` e que `--require-ready` falhe até os onze gates reais estarem aprovados. Não fazer merge automático.
2. Consolidar o motor da PR #193 sobre o SHA atual da release somente após revisão e CI; manter o mesmo `GuardianMascot` e `mascotModel` como autoridades.
3. Recuperar a referência **exata** selecionada (sem exigir upload do usuário como pré-condição arbitrária); versionar bytes, licença, hash e comparar ao material de origem. Se bytes inacessíveis, reportar `BLOCKED_ART`, sem declarar candidato como aprovado.
4. Criar rig editável fiel com cabeça, visor, aletas, tronco, núcleo, quatro módulos e anéis separados, preenchimento de oclusões e pivôs. Gerar poses derivadas, não colagens/invenções IA. Solicitar revisão visual do artefato real no tamanho de uso.

### P1 — Completar experiência e cobertura

5. Criar laboratório **do próprio HUD** com demonstração controlada de cinco moods × oito behaviors × doze IDs, distinção explícita de eventos simulados, screenshots/vídeo e controle de reduced-motion. Não atribuir isso a telemetria de produção.
6. Decidir e implementar apresentação do mascote no **Web HUD** (sem transportar hitbox/drag da janela nativa); escrever testes de `shouldRenderHudMascot` e fluxo acessível para o Web.
7. Implementar QA E2E de `COMPLETED_VERIFIED` com primeiro fetch baseline, troca de projeto, dedupe, trabalho/critical preemptando, backdrop visível e retomada após aba oculta; keyboard/pointer/touch/snooze.
8. Integrar laboratório Site 2D noindex de 8 behaviors após comparar assets por SHA; derivar Hero PT/EN com mesmo master e movimento leve. Não confundir arquivo disponível com render realmente verificável.

### P2 — Homologação e publicação

9. Medir CPU/RAM, quadros, repaint, LCP e transferência de assets em 64/92/128px, 390/768/1280px, redução de movimento, Chrome/Firefox/Safari; atribuir PASS apenas a ambientes executados.
10. Homologar Wails em Windows/macOS/Linux X11/Wayland onde disponível, incluindo DPI 100/125/150%, click-through, drag, foco e TRAY. Plataformas não executadas = NOT_RUN.
11. Registrar evidências/hashes no manifesto dos onze gates e aprovação humana **no SHA exato** para HUD e Site; somente então considerar `READY` e publicação simultânea.

## Não objetivos nesta etapa

Não instalar Three.js, não construir um segundo motor, não produzir falso giro 360°, não adicionar braços/membros humanos, não forçar merge de PRs encerradas sem revisão, não promover colagens de conceito a spritesheets, não publicar no Nexus-Site prematuramente. Não alterar Maestro/Core ou fabricar métricas de execução. A issue #129 permanece backlog real.

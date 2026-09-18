# Composer Conversation Workspace Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Reformular o Composer para um workspace conversacional alinhado à referência visual, sem criar um segundo chat e sem alterar contratos de API.

**Architecture:** `ConversationSurface` permanece domain-agnostic (assistant-ui ExternalStore) e ganha slots (`threadEnd`, `composerLeading`, `composerTrailing`, inspector drawer). A lógica de Composer (briefing, readiness, destinations, unknowns) fica em `features/work`. O App Shell existente não é duplicado; o header de página do WorkSurface encolhe e os chips operacionais entram no header do workspace.

**Tech Stack:** React 18, TypeScript, SCSS Modules, `--nx-*` tokens, `@assistant-ui/react` 0.15, Radix Dialog (`ContextDrawer`), vitest (helpers), i18n `en`/`ptBR`/`es`.

## Global Constraints

- Preservar `@assistant-ui/react` + ExternalStore; proibido segundo runtime de chat.
- Sem dados fake (readiness, memórias, intelligence, timestamps).
- Sem botões sem backend; `@` abre o picker de contexto existente; sem seletor de modelo inventado.
- i18n obrigatório; SCSS Modules; sem `!important`; sem CSS inline estático.
- Preservar: sessions, unknowns/remember/dismiss, attachments PNG/JPEG/WEBP, skills, destinations/gates, bootstrap activity, tools.
- Commits pequenos Conventional Commits (`feat(composer)`, `refactor(conversation)`, `style(composer)`, `test(composer)`).
- Branch: `feat/nexus-maximum-delivery`.

## Files

Create:

- `web/src/features/work/ComposerWorkspaceHeader.tsx` + `.module.scss`
- `web/src/features/work/ComposerInspector.tsx` + `.module.scss`
- `web/src/features/work/ComposerBriefOverview.tsx`
- `web/src/features/work/ComposerReadiness.tsx`
- `web/src/features/work/ComposerActionDock.tsx`
- `web/src/features/work/ComposerPendingDecision.tsx`
- `web/src/features/work/ComposerContextTab.tsx`

Modify:

- `web/src/features/work/composerModel.ts` + `.test.ts`
- `web/src/features/conversation/ConversationSurface.tsx` + `.module.scss`
- `web/src/features/conversation/AgentActivity.module.scss`
- `web/src/features/work/ComposerSurface.tsx` + `.module.scss`
- `web/src/features/work/WorkSurface.tsx`
- `web/src/features/work/ClarificationPrompt.tsx` + `.module.scss`
- `web/src/features/work/ComposerSkills.tsx`
- `web/src/i18n/resources.ts`

Reuse: `ComposerStart`, `ComposerSessionHeader` (histórico/select), `ComposerPromptPreview`, `ComposerDestinations`, `ComposerContextChips`, `ClarificationPrompt`, `ContextDrawer`, `Badge`, `Button`, `IconButton`.

---

### Task 1: Readiness/stats helpers (TDD)

**Files:** `composerModel.ts`, `composerModel.test.ts`

- [x] Add `composerReadinessDisplay`, `composerBriefStats`, `formatComposerRelativeTime`.
- [x] Tests: score > 0 uses bar; score 0/missing uses semantic state; stats count real unknowns/decisions/risks.
- [x] Commit: `test(composer): cover readiness display without fake scores`

### Task 2: ConversationSurface layout + slots

**Files:** `ConversationSurface.tsx`, `.module.scss`

- [x] Add `threadEnd`, `composerLeading`, `composerTrailing`, `hideInspectorChrome`, `inspectorVariant: 'split' | 'drawer'`.
- [x] Messages: avatar + meta; user = subtle bubble; assistant = open content (borda só em data/tools).
- [x] Composer sticky, rounded, paperclip inside bar; no full-bleed top border.
- [x] Grid ~68/32, inspector 340–430px; drawer via `ContextDrawer` <768px.
- [x] Commit: `refactor(conversation): add workspace slots and message chrome`

### Task 3: Composer header + inspector + dock

**Files:** new work components + `ComposerSurface.tsx` + `WorkSurface.tsx`

- [x] Compact header: title, Histórico, Nova, session meta, operational chips (props reais).
- [x] Inspector tabs Briefing / Contexto / Prompt.
- [x] Clarification in `threadEnd`; pending bar + contextual actions.
- [x] Collapse WorkSurface page H1; pass intelligence/memory/context into Composer.
- [x] Commit: `feat(composer): add conversation workspace layout`

### Task 4: i18n, a11y, polish, gates

- [x] Keys `en`/`ptBR`/`es` parity.
- [x] Keyboard tabs, focus-visible, no fake buttons.
- [x] `make web-verify` + Go not required unless API changes.
- [x] Screenshots desktop/tablet/mobile vs referência.
- [x] Commit: `style(composer): align conversation workspace with nexus design`

## Clarification-in-thread

Do **not** persist a fake turn. Inject `ClarificationPrompt` via `threadEnd` inside `ThreadPrimitive.Viewport` after messages so it is chronological in the thread without a second source of truth. Footer no longer hosts the primary question.

## Readiness rule

If `brief.readiness.score` is a finite number `> 0`, show percentage + bar. Otherwise show semantic state (`BLOCKED` / `EXPLORING` / `READY_WITH_ASSUMPTIONS` / `READY`) — never invent 82%.

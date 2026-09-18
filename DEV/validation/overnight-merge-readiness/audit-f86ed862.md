## BLOCKER

**1. Violação massiva de i18n — `PlanBuilderSurface.tsx` (work/Flow)**  
Dezenas de strings visíveis hardcoded (PT/EN misturados), sem `t(...)`, incluindo labels de UI, placeholders, alertas, notices e mensagens de erro de geração:

```308:308:web/src/features/work/PlanBuilderSurface.tsx
    setPlanError('Geração cancelada.');
```

```350:376:web/src/features/work/PlanBuilderSurface.tsx
      setGenerateStage('Checando Intelligence…');
      // ...
          setPlanError(probe.detail || probe.error || 'Intelligence probe failed');
      // ...
      setGenerateStage('Pedindo o rascunho ao modelo…');
```

```875:921:web/src/features/work/PlanBuilderSurface.tsx
        <strong>Clarification required before autonomous planning</strong>
        // ...
                placeholder="Answer required"
        // ...
          <ArrowRight size={14} /> Continue planning
```

```933:993:web/src/features/work/PlanBuilderSurface.tsx
            <span>{flowDraft && selectedPlan ? 'Flow Rascunho Ativo' : 'Sua solicitação'}</span>
            // ...
                placeholder="Descreva o objetivo do Flow…"
            // ...
                <Sparkles size={14} /> {generating ? 'Refinando…' : 'Refinar'}
            // ...
                  {flowDirty ? 'Alterações não salvas' : `Revisão ${selectedPlan.current_revision}`}
            // ...
                  <Route size={14} /> {preflightBusy ? 'Verificando…' : 'Preflight'}
            // ...
                  <Play size={14} /> {runBusy ? 'Iniciando…' : 'Approve & Run'}
```

```1128:1141:web/src/features/work/PlanBuilderSurface.tsx
          <InlineAlert tone="danger" title="Nexus Intelligence could not generate the plan">
            {planError}. Direct AI sessions remain available; configure a real Intelligence provider
            in Settings to use AI Planning.
          </InlineAlert>
        // ...
          <InlineAlert tone="warning" title="Flow needs attention">
            {flowErrors.join(' ')}
```

Contraria regra obrigatória AGENTS.md §2.5 (texto visível deve usar `react-i18next`).

**2. Violação massiva de i18n — `FlowRunSurface.tsx` (Flow)**  
Superfície inteira sem `useTranslation`; todo copy user-facing em inglês hardcoded:

```148:148:web/src/features/work/FlowRunSurface.tsx
        <Spinner label="Loading Flow Run…" />
```

```162:165:web/src/features/work/FlowRunSurface.tsx
            <Activity size={13} /> FLOW RUN
          </span>
          <h1>{flow?.title || plan?.title || `Run ${run.id}`}</h1>
          <p>Durable Mission Runner execution · Project {project.name}</p>
```

```175:182:web/src/features/work/FlowRunSurface.tsx
        <InlineAlert tone="danger" title="Flow Run action failed">
          {error}
        </InlineAlert>
      {run.state === 'BLOCKED_NEEDS_USER' && (
        <InlineAlert tone="warning" title="Human decision required">
          {run.paused_reason ||
            'Execution stopped fail-closed and will not redispatch automatically.'}
```

```304:360:web/src/features/work/FlowRunSurface.tsx
          <RefreshCw size={12} /> Refresh
        // ...
            <Pause size={12} /> Pause
        // ...
            <TerminalSquare size={12} /> Take Control
        // ...
            <Play size={12} /> Resume / Return
        // ...
            <Square size={12} /> Cancel
```

---

## HIGH

**3. `scan.error` exibido cru, sem chave i18n nem rótulo acessível — `ProjectIntelligenceInspector.tsx`**  
Erro de backend renderizado diretamente (colapsado e expandido):

```106:109:web/src/features/work/ProjectIntelligenceInspector.tsx
      {(error || scan?.error) && !expanded ? (
        <p className={styles.error} role="status">
          {error || scan?.error}
```

```124:124:web/src/features/work/ProjectIntelligenceInspector.tsx
          {scan?.error ? <p className={styles.error}>{scan.error}</p> : null}
```

**4. Corrida async no poll de Project Intelligence — `ProjectIntelligenceInspector.tsx`**  
Intervalo dispara `refresh()` a cada 1,5s sem guard de montagem/sequência; respostas concorrentes podem sobrescrever estado stale:

```26:48:web/src/features/work/ProjectIntelligenceInspector.tsx
  const refresh = useCallback(async () => {
    try {
      setError('');
      setView(await nexus.getProjectIntelligence(projectId));
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }, [projectId]);
  // ...
    const timer = window.setInterval(() => {
      void refresh();
    }, 1500);
```

**5. Corrida async na geração de plano — `PlanBuilderSurface.tsx`**  
`finally` sempre faz `setGenerating(false)` mesmo se uma geração mais nova já estiver em curso (abort/refire):

```405:411:web/src/features/work/PlanBuilderSurface.tsx
    } finally {
      window.clearTimeout(timeout);
      if (generateAbortRef.current === controller) generateAbortRef.current = null;
      setGenerating(false);
      setGenerateStage('');
      setGenerateHeartbeat(false);
    }
```

**6. Corrida async ao trocar plano selecionado — `PlanBuilderSurface.tsx`**  
Fetch de revisões sem token/disposed; resposta de plano A pode aplicar após seleção de plano B:

```178:191:web/src/features/work/PlanBuilderSurface.tsx
  useEffect(() => {
    if (!selectedPlan) {
      setRevisions([]);
      return;
    }
    nexusApi
      .getPlan(selectedPlan.id)
      .then((detail) => {
        setRevisions(detail.revisions || []);
        if (detail.plan.current_revision !== selectedPlan.current_revision)
          setSelectedPlan(normalizeWorkPlan(detail.plan));
      })
```

**7. Corrida async no refresh de Flow Run — `FlowRunSurface.tsx`**  
Polling + `refresh()` sem guard; callback depende de `plan?.id`, permitindo respostas fora de ordem:

```73:102:web/src/features/work/FlowRunSurface.tsx
  const refresh = useCallback(async () => {
    try {
      const current = await nexusApi.getRun(runId);
      setRun(current);
      const [nextEvidence, detail] = await Promise.all([
        nexusApi.getRunEvidence(runId),
        plan?.id === current.plan_id ? Promise.resolve(null) : nexusApi.getPlan(current.plan_id),
      ]);
      setEvidence(nextEvidence);
      if (detail) setPlan(detail.plan);
      // ...
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }, [runId, plan?.id]);
  // ...
    const timer = window.setInterval(() => void refresh(), 1500);
```

**8. DirectSession continua após fechar dialog — `DirectSessionLauncher.tsx`**  
`start()` não verifica `open`/mounted antes de `onStarted`/`onClose`; fechar durante `starting` ainda cria agente:

```103:128:web/src/features/work/DirectSessionLauncher.tsx
  const start = async () => {
    if (!selected || !request) return;
    setStarting(true);
    // ...
      await nexus.startAgent(agent.id);
      await refreshAgents();
      await onStarted(agent, request.prompt);
      onClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setStarting(false);
    }
  };
```

**9. Acessibilidade: radiogroup sem navegação por teclado — `DirectSessionLauncher.tsx`**  
Controles `role="radio"` só respondem a click; sem setas/roving tabindex (WCAG 2.2 teclado):

```186:194:web/src/features/work/DirectSessionLauncher.tsx
                    <button
                      type="button"
                      role="radio"
                      aria-checked={checked}
                      data-quota-state={quotaState}
                      className={`${resourceStyles.account} nx-direct-resource`}
                      data-selected={checked ? 'true' : 'false'}
                      key={resource.id}
                      onClick={() => choose(resource)}
```

**10. `any` introduzido em estado de produção — `PlanBuilderSurface.tsx`**

```87:87:web/src/features/work/PlanBuilderSurface.tsx
  const [compiledPrompt, setCompiledPrompt] = useState<any>(null);
```

---

## MEDIUM

**11. Plumbing `skill_ids`: dessincronia no load — `flowModel.ts`**  
`skillIds` faz fallback para `maestro_skills`, mas `maestroSkills` só lê `maestro_skills`; payload só com `skill_ids` perde alias legacy no round-trip save:

```92:94:web/src/features/work/flowModel.ts
        skillIds: clone(pkg.skill_ids || pkg.maestro_skills),
        maestroGates: clone(pkg.maestro_gates),
        maestroSkills: clone(pkg.maestro_skills),
```

**12. Corrida async em recomendações de usage — `UsageSurface.tsx`**  
`loadRecommend` sem guard de request-id; refresh rápido pode aplicar recomendação stale:

```27:52:web/src/features/usage/UsageSurface.tsx
  const loadRecommend = useCallback(async (activePolicy: string) => {
    setRecommendLoading(true);
    setRecommendError('');
    try {
      const result = await nexus.recommendResources(
        { task_kind: 'general', role: 'developer' },
        activePolicy || 'BALANCED',
      );
      setRecommend(result);
    } catch (err) {
      setRecommend(null);
      setRecommendError(err instanceof Error ? err.message : String(err));
    } finally {
      setRecommendLoading(false);
    }
  }, []);
  // ...
      void loadRecommend(nextPolicy);
```

**13. Enum cru visível ao usuário — `FlowStepInspector.tsx`**

```47:47:web/src/features/work/FlowStepInspector.tsx
        <Badge tone="brand">{step.assignmentStrategy}</Badge>
```

**14. `console.error` em superfícies de feature (não `console.log`/`debugger`) — `PlanBuilderSurface.tsx`**  
Múltiplos logs em produção (ex.: linhas 161, 175, 190, 245, 465, 642, 816+); não há `console.log`/`debugger` em `features/work` ou `features/usage`.

**15. i18n parcial em usage — `UsageSurface.tsx`**  
Fallbacks PT hardcoded inline (funciona, mas inconsistente com chaves em `resources.ts`):

```103:115:web/src/features/usage/UsageSurface.tsx
              {t('usage.loadedAt', 'Atualizado')}: {new Date(loadedAt).toLocaleTimeString()}
            // ...
              ? t('usage.refreshing', 'Atualizando…')
              : t('usage.refresh', 'Atualizar cota')}
```

---

### Limpo (sem achado MEDIUM+)

- **`console.log` / `debugger`**: ausentes em `web/src/features/work/**` e `web/src/features/usage/**`.
- **`@ts-ignore` / `@ts-nocheck`**: ausentes em código de produção work/usage (só em testes).
- **`skill_ids` Composer → API**: plumbing correto em `ComposerSurface.tsx:76-81` → `nexus.finalizeComposerSession(..., selectedSkillIds)` → `api.ts:143-146` (`skill_ids` no body). Flow editor persiste via `flowModel.ts:161-163`.

[REDACTED]
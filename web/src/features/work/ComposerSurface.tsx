import React, { useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  CheckCircle2,
  ClipboardCopy,
  FileText,
  Layers,
  Lightbulb,
  MessageCircle,
  Plus,
  RefreshCw,
  Send,
  Sparkles,
} from 'lucide-react';
import { Badge, Button, Card, Input, Select } from '../../design-system';
import { nexus } from '../../nexus/api';
import type {
  ComposerSession,
  ComposerSessionView,
  PromptArtifact,
  PromptReadinessCheck,
  Project,
} from '../../types';
import { selectResumableComposerSession } from './composerSessionModel';
import { composerNeedsGapConfirmation } from './composerModel';
import { asArray, asStringArray } from '../../lib/safeArray';

const ARCHETYPE_LABELS: Record<string, string> = {
  SOFTWARE_FEATURE: 'Feature',
  BUG_FIX: 'Correção',
  ARCHITECTURE: 'Arquitetura',
  DEVOPS: 'DevOps',
  RESEARCH: 'Pesquisa',
  SECURITY: 'Segurança',
  GENERIC: 'Genérico',
};

export const ComposerSurface: React.FC<{
  project: Project;
  onTransformFlow: (artifact: PromptArtifact) => void;
}> = ({ project, onTransformFlow }) => {
  const { t } = useTranslation();
  const [sessions, setSessions] = useState<ComposerSession[]>([]);
  const [view, setView] = useState<ComposerSessionView | null>(null);
  const [inputMode, setInputMode] = useState<'IDEA' | 'EXISTING_PROMPT'>('IDEA');
  const [draft, setDraft] = useState('');
  const [sourcePrompt, setSourcePrompt] = useState('');
  const [message, setMessage] = useState('');
  const [artifact, setArtifact] = useState<PromptArtifact | null>(null);
  const [error, setError] = useState('');
  const [confirmGaps, setConfirmGaps] = useState(false);
  const [busy, setBusy] = useState(false);
  const [refineText, setRefineText] = useState('');
  const [showRefineInput, setShowRefineInput] = useState(false);
  const [unknownAnswers, setUnknownAnswers] = useState<Record<string, string>>({});

  const selectedSkillIds = useMemo(
    () =>
      (view?.skills || [])
        .filter((skill) => skill.state === 'ACCEPTED' || skill.state === 'APPLIED')
        .map((skill) => skill.skill_id),
    [view?.skills],
  );

  const refreshSessions = async () => {
    const next = await nexus.listComposerSessions(project.id);
    setSessions(next || []);
    return next || [];
  };
  useEffect(() => {
    void (async () => {
      try {
        const next = await refreshSessions();
        const id = selectResumableComposerSession(next);
        if (id) setView(await nexus.getComposerSession(id));
      } catch (err) {
        setError(err instanceof Error ? err.message : String(err));
      }
    })();
  }, [project.id]);

  const briefItems = useMemo(
    () =>
      view
        ? [
            ['Entendido', view.brief.goal],
            [
              'Contexto',
              [
                ...asStringArray(view.brief.context),
                ...(view.brief.context &&
                typeof view.brief.context === 'object' &&
                !Array.isArray(view.brief.context)
                  ? asStringArray(
                      (view.brief.context as { existing_state?: unknown }).existing_state,
                    )
                  : []),
              ].join(' · '),
            ],
            ['Decisões', asStringArray(view.brief.decisions).join(' · ')],
            ['Critérios', asStringArray(view.brief.success_criteria).join(' · ')],
            ['Dúvidas abertas', asStringArray(view.brief.open_questions).join(' · ')],
          ].filter(([, value]) => value)
        : [],
    [view],
  );

  const create = async () => {
    if (!draft.trim()) return;
    setBusy(true);
    setError('');
    try {
      const next = await nexus.createComposerSessionWithMode(
        project.id,
        draft.trim(),
        inputMode,
        inputMode === 'EXISTING_PROMPT' ? sourcePrompt.trim() : undefined,
      );
      setView(next);
      setDraft('');
      setSourcePrompt('');
      setArtifact(null);
      await refreshSessions();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  };

  const send = async () => {
    if (!view || !message.trim() || busy) return;
    const textToSend = message.trim();
    setBusy(true);
    setError('');
    try {
      setView(await nexus.addComposerTurn(view.session.id, textToSend));
      setMessage('');
      await refreshSessions();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
      // message is preserved because setMessage('') was only called on success!
    } finally {
      setBusy(false);
    }
  };

  const finalize = async (confirmed = false) => {
    if (!view) return;
    if (!confirmed && composerNeedsGapConfirmation(view.brief)) {
      setConfirmGaps(true);
      setError(t('work.confirmGaps'));
      return;
    }
    setBusy(true);
    setError('');
    try {
      const res = await nexus.finalizeComposerSession(view.session.id, selectedSkillIds, confirmed);
      setArtifact(res);
      setView(await nexus.getComposerSession(view.session.id));
      await refreshSessions();
    } catch (err) {
      const detail = err instanceof Error ? err.message : String(err);
      setError(detail);
      setConfirmGaps(detail.includes('open questions') || detail.includes('gaps'));
    } finally {
      setBusy(false);
    }
  };

  const refine = async () => {
    if (!view) return;
    setBusy(true);
    setError('');
    try {
      const newArtifact = await nexus.refineComposerArtifact(view.session.id, refineText.trim());
      setArtifact(newArtifact);
      setShowRefineInput(false);
      setRefineText('');
      setView(await nexus.getComposerSession(view.session.id));
      await refreshSessions();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  };

  const resolveUnknown = async (unknownId: string, status: string) => {
    if (!view) return;
    setBusy(true);
    setError('');
    try {
      const answer = unknownAnswers[unknownId] || '';
      const updated = await nexus.resolveComposerUnknown(
        view.session.id,
        unknownId,
        answer,
        status,
      );
      setView(updated);
      setUnknownAnswers((prev) => {
        const next = { ...prev };
        delete next[unknownId];
        return next;
      });
      await refreshSessions();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  };

  const updateSkill = async (skillId: string, state: 'ACCEPTED' | 'REJECTED') => {
    if (!view) return;
    setBusy(true);
    setError('');
    try {
      setView(await nexus.updateComposerSkillState(view.session.id, skillId, state));
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  };

  const copy = async () => {
    if (!artifact) return;
    try {
      await navigator.clipboard.writeText(artifact.content);
    } catch {
      setError('Não foi possível copiar o prompt neste navegador.');
    }
  };

  if (!view) {
    return (
      <Card className="nx-composer-goal-bar">
        <div
          className="nx-composer-goal-bar__label"
          style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}
        >
          <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
            <Sparkles size={16} />
            <span>Comece uma elaboração</span>
          </div>
          <div style={{ display: 'flex', gap: 4 }}>
            <Button
              size="sm"
              tone={inputMode === 'IDEA' ? 'brand' : 'default'}
              onClick={() => setInputMode('IDEA')}
              disabled={busy}
            >
              <Lightbulb size={13} /> Ideia / Explorar
            </Button>
            <Button
              size="sm"
              tone={inputMode === 'EXISTING_PROMPT' ? 'brand' : 'default'}
              onClick={() => setInputMode('EXISTING_PROMPT')}
              disabled={busy}
            >
              <FileText size={13} /> Prompt Existente
            </Button>
          </div>
        </div>

        <div style={{ display: 'grid', gap: 8, marginTop: 8 }}>
          <Input
            value={draft}
            onChange={setDraft}
            placeholder={
              inputMode === 'IDEA'
                ? 'Qual ideia ou objetivo você quer transformar em um ótimo prompt?'
                : 'Qual o objetivo principal deste prompt existente?'
            }
            style={{ flex: 1 }}
            disabled={busy}
          />
          {inputMode === 'EXISTING_PROMPT' && (
            <textarea
              className="nx-textarea"
              value={sourcePrompt}
              onChange={(e) => setSourcePrompt(e.target.value)}
              placeholder="Cole aqui o prompt original para análise de gaps, completude e estruturação…"
              rows={4}
              style={{
                width: '100%',
                padding: 10,
                borderRadius: 8,
                background: 'var(--nx-surface)',
                color: 'var(--nx-text)',
                border: '1px solid var(--nx-border)',
                resize: 'vertical',
                fontFamily: 'inherit',
                fontSize: 13,
              }}
              disabled={busy}
            />
          )}
          <div style={{ display: 'flex', justifyContent: 'flex-end' }}>
            <Button
              tone="brand"
              disabled={
                !draft.trim() || (inputMode === 'EXISTING_PROMPT' && !sourcePrompt.trim()) || busy
              }
              onClick={() => void create()}
            >
              <MessageCircle size={14} />{' '}
              {busy
                ? 'Iniciando…'
                : inputMode === 'EXISTING_PROMPT'
                  ? 'Analisar Prompt'
                  : 'Conversar'}
            </Button>
          </div>
        </div>

        {sessions.length > 0 && (
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 8,
              flexWrap: 'wrap',
              marginTop: 8,
              paddingTop: 8,
              borderTop: '1px solid var(--nx-border)',
            }}
          >
            <small style={{ color: 'var(--nx-muted)' }}>Elaborações anteriores:</small>
            <Select
              placeholder="Retomar uma sessão salva…"
              value=""
              onChange={async (id) => {
                if (!id) return;
                setBusy(true);
                setError('');
                try {
                  setView(await nexus.getComposerSession(id));
                } catch (err) {
                  setError(err instanceof Error ? err.message : String(err));
                } finally {
                  setBusy(false);
                }
              }}
              options={sessions.map((s) => ({
                value: s.id,
                label: `${s.title || `Sessão ${s.id.slice(-6)}`} · ${s.state}`,
              }))}
              selectStyle={{ fontSize: 12, height: 28 }}
            />
          </div>
        )}
        {error && <div className="nx-inline-error">{error}</div>}
      </Card>
    );
  }

  const archetype = view.brief.intent?.archetype;
  const archetypeLabel = archetype ? ARCHETYPE_LABELS[archetype] || archetype : null;

  return (
    <div className="nx-composer-deliberative">
      <div className="nx-composer-deliberative__header">
        <div>
          <h2>{view.session.title || 'Elaboração'}</h2>
        </div>
        <div style={{ display: 'flex', gap: 8, alignItems: 'center', flexWrap: 'wrap' }}>
          {archetypeLabel && <Badge tone="default">Arquétipo: {archetypeLabel}</Badge>}
          {sessions.length > 1 && (
            <Select
              value={view.session.id}
              onChange={async (id) => {
                if (!id || id === view.session.id) return;
                setBusy(true);
                setError('');
                try {
                  setView(await nexus.getComposerSession(id));
                  setArtifact(null);
                } catch (err) {
                  setError(err instanceof Error ? err.message : String(err));
                } finally {
                  setBusy(false);
                }
              }}
              options={sessions.map((s) => ({
                value: s.id,
                label: `${s.title || `Sessão ${s.id.slice(-6)}`} (${s.state})`,
              }))}
              selectStyle={{ fontSize: 12, height: 28 }}
            />
          )}
          <Badge tone={view.session.state === 'FINALIZED' ? 'success' : 'brand'}>
            {view.session.state}
          </Badge>
          <Button
            size="sm"
            disabled={busy}
            onClick={() => {
              setView(null);
              setArtifact(null);
            }}
          >
            <Plus size={13} /> Nova
          </Button>
        </div>
      </div>

      <div className="nx-composer-deliberative__grid">
        <Card>
          <strong>Conversa de elaboração</strong>
          <div className="nx-composer-turns">
            {(view.turns || []).length === 0 ? (
              <p className="nx-muted-copy">
                Descreva contexto, resultado desejado e limitações. O Composer preserva esta
                elaboração.
              </p>
            ) : (
              (view.turns || []).map((turn) => (
                <div
                  key={turn.id}
                  className={`nx-composer-turn nx-composer-turn--${turn.role.toLowerCase()}`}
                >
                  <small>{turn.role === 'USER' ? 'Você' : 'Composer'}</small>
                  <p>{turn.content}</p>
                </div>
              ))
            )}
          </div>
          {view.session.state !== 'FINALIZED' && (
            <div
              className="nx-composer-goal-bar__row"
              style={{ display: 'flex', gap: 8, alignItems: 'flex-end', marginTop: 12 }}
            >
              <textarea
                className="nx-textarea"
                value={message}
                onChange={(e) => setMessage(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter' && !e.shiftKey && !e.nativeEvent.isComposing) {
                    e.preventDefault();
                    void send();
                  }
                }}
                placeholder="Adicione requisito, decisão ou resposta a uma lacuna… (Enter envia, Shift+Enter pula linha)"
                rows={2}
                style={{
                  flex: 1,
                  minHeight: 48,
                  maxHeight: 180,
                  resize: 'vertical',
                  padding: '8px 12px',
                  borderRadius: 8,
                  background: 'var(--nx-bg-elevated)',
                  border: '1px solid var(--nx-border)',
                  color: 'var(--nx-text)',
                  fontSize: 13,
                  lineHeight: 1.45,
                  fontFamily: 'inherit',
                }}
                disabled={busy}
              />
              <Button
                tone="brand"
                disabled={!message.trim() || busy}
                onClick={() => void send()}
                style={{ height: 48, alignSelf: 'stretch' }}
              >
                <Send size={14} /> {busy ? 'Enviando…' : 'Enviar'}
              </Button>
            </div>
          )}
        </Card>

        <Card>
          <strong>Briefing vivo</strong>
          <div style={{ display: 'grid', gap: 10, marginTop: 12 }}>
            {briefItems.length > 0 ? (
              briefItems.map(([label, value]) => (
                <div key={label}>
                  <small style={{ color: 'var(--nx-muted)' }}>{label}</small>
                  <p style={{ margin: '2px 0 0' }}>{value}</p>
                </div>
              ))
            ) : (
              <p className="nx-muted-copy" style={{ margin: '6px 0 0' }}>
                O briefing vivo sintetiza objetivo, contexto, critérios e decisões conforme a
                conversa avança.
              </p>
            )}
          </div>
          <div style={{ marginTop: 16 }}>
            <Badge tone="default">Maestro: sugestões reais ao refinar</Badge>
          </div>
        </Card>
      </div>

      <div className="nx-composer-deliberative__grid">
        <Card>
          <strong>Prompt Readiness</strong>
          <p>{view.brief.readiness?.summary || 'Ainda sem avaliação.'}</p>
          <Badge tone={view.brief.readiness?.state === 'READY' ? 'success' : 'warning'}>
            {view.brief.readiness?.state || 'UNKNOWN'} · {view.brief.readiness?.score ?? 0}%
          </Badge>

          {asArray<PromptReadinessCheck>(view.brief.readiness?.checks).length > 0 && (
            <div style={{ marginTop: 12, display: 'grid', gap: 6 }}>
              <small style={{ color: 'var(--nx-muted)', fontWeight: 600 }}>
                Dimensões avaliadas
              </small>
              {asArray<PromptReadinessCheck>(view.brief.readiness?.checks).map((check) => (
                <div
                  key={check.key}
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'space-between',
                    padding: '4px 8px',
                    borderRadius: 6,
                    background: 'var(--nx-surface)',
                    fontSize: 12,
                  }}
                >
                  <div style={{ flex: 1 }}>
                    <span style={{ fontWeight: 500 }}>{check.label || check.key}</span>
                    {check.summary && (
                      <small style={{ display: 'block', color: 'var(--nx-muted)' }}>
                        {check.summary}
                      </small>
                    )}
                  </div>
                  <span
                    style={{
                      fontWeight: 600,
                      color: check.score >= 80 ? 'var(--nx-accent)' : 'var(--nx-muted)',
                    }}
                  >
                    {check.score}%
                  </span>
                </div>
              ))}
            </div>
          )}

          {(view.brief.assumptions || []).length > 0 && (
            <div style={{ marginTop: 12 }}>
              <small style={{ color: 'var(--nx-muted)', fontWeight: 600 }}>Premissas ativas</small>
              {view.brief.assumptions?.map((item, index) => (
                <p key={index} className="nx-muted-copy" style={{ margin: '3px 0' }}>
                  • {typeof item === 'string' ? item : item.value}{' '}
                  {typeof item !== 'string' && item.status ? `(${item.status})` : ''}
                </p>
              ))}
            </div>
          )}
        </Card>

        <Card>
          <strong>Lacunas & Perguntas (Unknowns)</strong>
          {(view.brief.unknowns || []).length === 0 ? (
            <p className="nx-muted-copy">Nenhuma lacuna aberta.</p>
          ) : (
            view.brief.unknowns?.map((unknown) => {
              const isResolved =
                unknown.status === 'ANSWERED' ||
                unknown.status === 'CONFIRMED' ||
                unknown.status === 'DISMISSED';
              return (
                <div
                  key={unknown.id}
                  style={{
                    marginTop: 10,
                    padding: 8,
                    borderRadius: 8,
                    background: 'var(--nx-surface)',
                    border: '1px solid var(--nx-border)',
                  }}
                >
                  <div
                    style={{
                      display: 'flex',
                      justifyContent: 'space-between',
                      alignItems: 'flex-start',
                      gap: 8,
                    }}
                  >
                    <p style={{ margin: 0, fontWeight: 500, fontSize: 13 }}>{unknown.question}</p>
                    <Badge
                      tone={
                        unknown.status === 'ANSWERED'
                          ? 'success'
                          : unknown.severity === 'BLOCKING'
                            ? 'danger'
                            : 'warning'
                      }
                    >
                      {unknown.severity} · {unknown.status}
                    </Badge>
                  </div>
                  {unknown.answer && (
                    <small style={{ display: 'block', marginTop: 4, color: 'var(--nx-accent)' }}>
                      Resposta: {unknown.answer}
                    </small>
                  )}
                  {!isResolved && (
                    <div style={{ marginTop: 8, display: 'grid', gap: 6 }}>
                      <input
                        className="nx-input"
                        placeholder="Sua resposta para esta lacuna…"
                        value={unknownAnswers[unknown.id] || ''}
                        onChange={(e) =>
                          setUnknownAnswers({ ...unknownAnswers, [unknown.id]: e.target.value })
                        }
                        style={{ fontSize: 12, padding: '4px 8px' }}
                        disabled={busy}
                      />
                      <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap' }}>
                        <Button
                          size="sm"
                          tone="brand"
                          disabled={!unknownAnswers[unknown.id]?.trim() || busy}
                          onClick={() => void resolveUnknown(unknown.id, 'ANSWERED')}
                        >
                          <CheckCircle2 size={12} /> Responder
                        </Button>
                        <Button
                          size="sm"
                          disabled={busy}
                          onClick={() => void resolveUnknown(unknown.id, 'DISMISSED')}
                        >
                          Dispensar
                        </Button>
                      </div>
                    </div>
                  )}
                </div>
              );
            })
          )}
        </Card>
      </div>

      {(view.skills || []).length > 0 && (
        <Card style={{ marginTop: 12 }}>
          <strong>Maestro skills</strong>
          {(view.skills || []).map((skill) => (
            <div
              key={skill.skill_id}
              style={{ display: 'flex', alignItems: 'center', gap: 8, marginTop: 10 }}
            >
              <div style={{ flex: 1 }}>
                <strong>{skill.skill_id}</strong>
                <small style={{ display: 'block' }}>{skill.reason || skill.applicability}</small>
              </div>
              <Badge tone={skill.state === 'UNAVAILABLE' ? 'danger' : 'default'}>
                {skill.state}
              </Badge>
              {skill.state !== 'UNAVAILABLE' && skill.state !== 'APPLIED' && (
                <>
                  <Button
                    size="sm"
                    tone="brand"
                    disabled={busy}
                    onClick={() => void updateSkill(skill.skill_id, 'ACCEPTED')}
                  >
                    Aceitar
                  </Button>
                  <Button
                    size="sm"
                    disabled={busy}
                    onClick={() => void updateSkill(skill.skill_id, 'REJECTED')}
                  >
                    Dispensar
                  </Button>
                </>
              )}
            </div>
          ))}
        </Card>
      )}

      {error && <div className="nx-inline-error">{error}</div>}

      {!artifact && view.session.state !== 'FINALIZED' && (
        <div className="nx-composer-header-actions">
          <Button tone="brand" disabled={busy} onClick={() => void finalize()}>
            <Sparkles size={14} /> {busy ? 'Finalizando…' : 'Concluir elaboração'}
          </Button>
          {confirmGaps && (
            <Button tone="warning" disabled={busy} onClick={() => void finalize(true)}>
              Concluir com lacunas confirmadas
            </Button>
          )}
        </div>
      )}

      {artifact && (
        <Card style={{ marginTop: 12 }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <strong>Prompt canônico · v{artifact.version}</strong>
            <Badge tone="success">Versão imutável #{artifact.version}</Badge>
          </div>
          <pre className="nx-flow-step-compare" style={{ whiteSpace: 'pre-wrap', marginTop: 8 }}>
            {artifact.content}
          </pre>
          <div
            className="nx-composer-header-actions"
            style={{ marginTop: 12, display: 'flex', gap: 8, flexWrap: 'wrap' }}
          >
            <Button onClick={() => void copy()}>
              <ClipboardCopy size={14} /> Copiar
            </Button>
            <Button tone="brand" onClick={() => onTransformFlow(artifact)}>
              <Layers size={14} /> Transformar em Flow
            </Button>
            <Button disabled={busy} onClick={() => setShowRefineInput(!showRefineInput)}>
              <RefreshCw size={14} /> Refinar (v{artifact.version + 1})
            </Button>
          </div>

          {showRefineInput && (
            <div
              style={{
                marginTop: 10,
                display: 'grid',
                gap: 6,
                padding: 10,
                borderRadius: 8,
                background: 'var(--nx-surface)',
                border: '1px solid var(--nx-border)',
              }}
            >
              <small style={{ color: 'var(--nx-muted)', fontWeight: 600 }}>
                Instrução adicional de refinamento (opcional):
              </small>
              <Input
                value={refineText}
                onChange={setRefineText}
                placeholder="Ex: 'Adicione suporte a PostgreSQL', 'Foque apenas na API REST'…"
                disabled={busy}
              />
              <div style={{ display: 'flex', gap: 8, justifyContent: 'flex-end' }}>
                <Button size="sm" disabled={busy} onClick={() => setShowRefineInput(false)}>
                  Cancelar
                </Button>
                <Button size="sm" tone="brand" disabled={busy} onClick={() => void refine()}>
                  {busy
                    ? 'Gerando v' + (artifact.version + 1) + '…'
                    : 'Gerar nova revisão (v' + (artifact.version + 1) + ')'}
                </Button>
              </div>
            </div>
          )}
        </Card>
      )}
    </div>
  );
};

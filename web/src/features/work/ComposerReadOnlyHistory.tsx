import React, { useEffect, useState } from 'react';
import { ClipboardCopy, RefreshCw, Trash2 } from 'lucide-react';
import { Button, ConfirmDialog } from '../../design-system';
import { nexus } from '../../nexus/api';
import type { ComposerSession, ComposerSessionView } from '../../types';
import styles from './WorkSurface.module.scss';

/**
 * A provider outage must block AI-generated mutations, not erase access to
 * previously approved planning records. No edit/send/finalize controls live here.
 */
export const ComposerReadOnlyHistory: React.FC<{
  projectId: string;
  onOpenSettings?: () => void;
  onRetry: () => void;
}> = ({ projectId, onOpenSettings, onRetry }) => {
  const [sessions, setSessions] = useState<ComposerSession[]>([]);
  const [view, setView] = useState<ComposerSessionView | null>(null);
  const [selectedId, setSelectedId] = useState('');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [deleteBusy, setDeleteBusy] = useState(false);

  useEffect(() => {
    let active = true;
    setLoading(true);
    setView(null);
    setSelectedId('');
    nexus.listComposerSessions(projectId).then((items) => {
      if (!active) return;
      const next = items || [];
      setSessions(next);
      setSelectedId(next[0]?.id || '');
      setError('');
    }).catch((reason: unknown) => {
      if (active) setError(reason instanceof Error ? reason.message : String(reason));
    }).finally(() => { if (active) setLoading(false); });
    return () => { active = false; };
  }, [projectId]);

  useEffect(() => {
    if (!selectedId) return;
    let active = true;
    nexus.getComposerSession(selectedId).then((loaded) => {
      if (active) { setView(loaded); setError(''); }
    }).catch((reason: unknown) => {
      if (active) setError(reason instanceof Error ? reason.message : String(reason));
    });
    return () => { active = false; };
  }, [selectedId]);

  const deleteSavedSession = async () => {
    if (!view || deleteBusy) return;
    setDeleteOpen(false);
    setDeleteBusy(true);
    setError('');
    try {
      await nexus.deleteComposerSession(view.session.id, view.session.revision);
      const remaining = await nexus.listComposerSessions(projectId);
      setSessions(remaining || []);
      setView(null);
      setSelectedId(remaining?.[0]?.id || '');
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : String(reason));
    } finally {
      setDeleteBusy(false);
    }
  };

  return (
    <div className={`nx-surface-scroll nx-composer-surface ${styles.workbench}`}>
      <section className={styles.contextFix} aria-label="Nexus Intelligence">
        <h2>Configure uma IA para criar ou revisar planejamentos</h2>
        <p>O Composer precisa de uma IA funcional em Nexus Intelligence. Sem ela,
          suas elaborações anteriores continuam disponíveis para leitura, mas nenhuma
          nova elaboração, revisão ou execução será iniciada por esta tela.</p>
        <div className={styles.contextFix}>
          <Button tone="brand" onClick={() => onOpenSettings?.()} disabled={!onOpenSettings}>
            Configurar Nexus Intelligence
          </Button>
          <Button onClick={onRetry}><RefreshCw size={13} /> Verificar IA novamente</Button>
        </div>
      </section>
      <section className={styles.contextFix} aria-label="Histórico do Composer">
        <h3>Histórico de planejamentos</h3>
        {loading && <p role="status">Carregando sessões salvas…</p>}
        {error && <p role="alert" className="nx-inline-error">{error}</p>}
        {!loading && !sessions.length && <p>Nenhum planejamento salvo neste projeto.</p>}
        {sessions.length > 0 && (
          <div>
            <label htmlFor="composer-read-only-session">Elaboração salva</label>
            <select id="composer-read-only-session" value={selectedId}
              onChange={(event) => { setSelectedId(event.target.value); setView(null); }}>
              {sessions.map((session) => (
                <option key={session.id} value={session.id}>
                  {session.title || session.id} · {session.state}
                </option>
              ))}
            </select>
          </div>
        )}
        {view && (
          <article aria-label="Planejamento salvo">
            <h4>{view.session.title}</h4>
            <p>{view.brief.goal}</p>
            <p>Estado: {view.session.state} · Revisão: {view.session.revision}</p>
            <Button
              size="sm"
              tone="danger"
              disabled={deleteBusy}
              onClick={() => setDeleteOpen(true)}
            >
              <Trash2 size={13} /> Excluir elaboração
            </Button>
            {(view.brief.decisions || []).length > 0 && (
              <>
                <h5>Decisões registradas</h5>
                <ul>{view.brief.decisions?.map((decision, index) => (
                  <li key={index}>{decision}</li>
                ))}</ul>
              </>
            )}
            {(view.artifacts || []).length > 0 && (
              <>
                <h5>Artefatos imutáveis</h5>
                {(view.artifacts || []).map((artifact) => (
                  <div key={artifact.id}>
                    <span>Versão {artifact.version}</span>{' '}
                    <Button size="sm" onClick={() => void navigator.clipboard.writeText(artifact.content).catch(() => setError('Não foi possível copiar o artefato.'))}>
                      <ClipboardCopy size={13} /> Copiar
                    </Button>
                  </div>
                ))}
              </>
            )}
          </article>
        )}
      </section>
      <ConfirmDialog
        open={deleteOpen}
        title="Excluir elaboração"
        description={`Excluir "${view?.session.title || ''}" e o histórico de suas decisões? Ação irreversível. Se houver um Flow ou uma entrega dependente, a exclusão será bloqueada.`}
        confirmLabel="Excluir elaboração"
        onCancel={() => setDeleteOpen(false)}
        onConfirm={() => void deleteSavedSession()}
      />
    </div>
  );
};

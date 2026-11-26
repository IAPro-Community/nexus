import React, { useCallback, useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Send, Trash2, X } from 'lucide-react';
import type {
  NexV2ActionProposal,
  NexV2ActionReceipt,
  NexV2ChatTurn,
  NexV2PersonalMemoryEntry,
} from './nexV2Types';
import {
  clearNexV2Conversation,
  deleteNexV2Memory,
  loadNexV2Conversation,
  loadNexV2Memories,
  loadNexV2Receipts,
  runNexV2Action,
  saveNexV2Memory,
  sendNexV2Chat,
} from './nexV2Client';
import { NexReceiptsPanel } from './NexReceiptsPanel';
import styles from './NexChatTab.module.scss';

export interface NexChatTabProps {
  projectId?: string;
  privacyMode: boolean;
  initialHistory?: NexV2ChatTurn[];
  initialMemories?: NexV2PersonalMemoryEntry[];
  initialReceipts?: NexV2ActionReceipt[];
  initialReceiptsOpen?: boolean;
  onConfirmProposal?: (proposal: NexV2ActionProposal) => void;
}

export const NexChatTab: React.FC<NexChatTabProps> = ({
  projectId = '',
  privacyMode,
  initialHistory,
  initialMemories,
  initialReceipts,
  initialReceiptsOpen = false,
  onConfirmProposal,
}) => {
  const { t } = useTranslation();
  const [input, setInput] = useState('');
  const [sending, setSending] = useState(false);
  const [history, setHistory] = useState<NexV2ChatTurn[]>(initialHistory ?? []);
  const [memories, setMemories] = useState<NexV2PersonalMemoryEntry[]>(initialMemories ?? []);
  const [memoryOpen, setMemoryOpen] = useState(false);
  const [memKey, setMemKey] = useState('');
  const [memValue, setMemValue] = useState('');
  const [receipts, setReceipts] = useState<NexV2ActionReceipt[]>(initialReceipts ?? []);
  const [receiptsOpen, setReceiptsOpen] = useState(initialReceiptsOpen);
  const [runningAction, setRunningAction] = useState(false);
  const abortRef = useRef<AbortController | null>(null);

  useEffect(() => {
    let active = true;
    if (initialHistory === undefined) {
      void loadNexV2Conversation(projectId).then((turns) => {
        if (active && turns.length > 0) {
          setHistory(turns);
        }
      });
    }
    if (initialMemories === undefined) {
      void loadNexV2Memories().then((entries) => {
        if (active) setMemories(entries);
      });
    }
    if (initialReceipts === undefined) {
      void loadNexV2Receipts().then((items) => {
        if (active) setReceipts(items);
      });
    }
    return () => {
      active = false;
      abortRef.current?.abort();
    };
  }, [initialHistory, initialMemories, initialReceipts, projectId]);

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault();
    const trimmed = input.trim();
    if (!trimmed || sending) return;

    const controller = new AbortController();
    abortRef.current = controller;
    const userTurn: NexV2ChatTurn = { role: 'user', content: trimmed };
    const priorHistory = history;
    setHistory([...priorHistory, userTurn]);
    setInput('');
    setSending(true);

    const reply = await sendNexV2Chat(trimmed, priorHistory, projectId, {
      persist: true,
      signal: controller.signal,
    });
    if (abortRef.current === controller) {
      abortRef.current = null;
    }
    setSending(false);
    setHistory((prev) => [
      ...prev,
      {
        role: 'assistant',
        content: reply.aborted
          ? t('nexV2.chatAbortedReply')
          : reply.offline || !reply.answer
            ? t('nexV2.chatOfflineReply')
            : reply.answer,
        source: reply.source,
        proposal: reply.proposal,
      },
    ]);
  };

  const handleClear = useCallback(async () => {
    const ok = await clearNexV2Conversation(projectId);
    if (ok) setHistory([]);
  }, [projectId]);

  const handleSaveMemory = async (event: React.FormEvent) => {
    event.preventDefault();
    if (!memKey.trim() || !memValue.trim()) return;
    const saved = await saveNexV2Memory(memKey.trim(), memValue.trim());
    if (saved) {
      setMemKey('');
      setMemValue('');
      const refreshed = await loadNexV2Memories();
      setMemories(refreshed);
    }
  };

  const handleDeleteMemory = async (id: string) => {
    const ok = await deleteNexV2Memory(id);
    if (ok) {
      setMemories((prev) => prev.filter((item) => item.id !== id));
    }
  };

  const recordReceipt = useCallback((receipt: NexV2ActionReceipt | null) => {
    if (!receipt) return;
    setReceipts((prev) => {
      const idx = prev.findIndex((item) => item.receipt_id === receipt.receipt_id);
      if (idx >= 0) {
        return prev.map((item, i) => (i === idx ? receipt : item));
      }
      return [...prev, receipt];
    });
  }, []);

  const handleRunReadHud = useCallback(async () => {
    setRunningAction(true);
    try {
      const receipt = await runNexV2Action(
        'read_hud_summary',
        projectId ? { project_id: projectId } : {},
        true,
      );
      recordReceipt(receipt);
    } finally {
      setRunningAction(false);
    }
  }, [projectId, recordReceipt]);

  const clearTurnProposal = (index: number) => {
    setHistory((prev) =>
      prev.map((turn, idx) => (idx === index ? { ...turn, proposal: undefined } : turn)),
    );
  };

  const handleConfirmProposal = async (index: number, proposal: NexV2ActionProposal) => {
    onConfirmProposal?.(proposal);
    clearTurnProposal(index);
    const receipt = await runNexV2Action(
      'open_nexus_surface',
      proposal.target ? { target: proposal.target } : {},
      true,
    );
    recordReceipt(receipt);
  };

  const handleDismissProposal = async (index: number, proposal: NexV2ActionProposal) => {
    clearTurnProposal(index);
    const receipt = await runNexV2Action(
      'open_nexus_surface',
      proposal.target ? { target: proposal.target } : {},
      false,
    );
    recordReceipt(receipt);
  };

  return (
    <div className={styles.chatContainer} data-nex-chat>
      <div className={styles.toolbar}>
        <button
          type="button"
          className={styles.toolbarBtn}
          aria-expanded={memoryOpen}
          data-nex-memory-toggle
          onClick={() => setMemoryOpen((prev) => !prev)}
        >
          {t('nexV2.memoryToggleLabel')} ({memories.length})
        </button>
        <button
          type="button"
          className={styles.toolbarBtn}
          aria-expanded={receiptsOpen}
          data-nex-receipts-toggle
          onClick={() => setReceiptsOpen((prev) => !prev)}
        >
          {t('nexV2.receiptsToggleLabel')} ({receipts.length})
        </button>
        {history.length > 0 && (
          <button
            type="button"
            className={styles.toolbarBtn}
            aria-label={t('nexV2.chatClear')}
            data-nex-chat-clear
            onClick={() => void handleClear()}
          >
            <Trash2 size={12} aria-hidden="true" />
            <span>{t('nexV2.chatClear')}</span>
          </button>
        )}
      </div>

      {memoryOpen && (
        <section className={styles.memoryBox} data-nex-memory-panel>
          <form className={styles.memoryForm} onSubmit={(e) => void handleSaveMemory(e)}>
            <input
              type="text"
              className={styles.smallInput}
              value={memKey}
              onChange={(e) => setMemKey(e.target.value)}
              placeholder={t('nexV2.memoryKeyPlaceholder')}
              aria-label={t('nexV2.memoryKeyPlaceholder')}
              data-nex-memory-key
            />
            <input
              type="text"
              className={styles.smallInput}
              value={memValue}
              onChange={(e) => setMemValue(e.target.value)}
              placeholder={t('nexV2.memoryValuePlaceholder')}
              aria-label={t('nexV2.memoryValuePlaceholder')}
              data-nex-memory-value
            />
            <button type="submit" className={styles.smallBtn} data-nex-memory-save>
              {t('nexV2.memorySaveBtn')}
            </button>
          </form>
          {memories.length === 0 ? (
            <p className={styles.emptyText}>{t('nexV2.memoryEmpty')}</p>
          ) : (
            <ul className={styles.memoryList}>
              {memories.map((entry) => (
                <li key={entry.id} className={styles.memoryItem}>
                  <span className={styles.memoryText}>
                    <strong>{entry.key}:</strong>{' '}
                    {privacyMode ? t('nexV2.privacyMasked') : entry.value}
                  </span>
                  <button
                    type="button"
                    className={styles.smallBtn}
                    aria-label={t('nexV2.memoryDeleteBtn')}
                    data-nex-memory-delete={entry.id}
                    onClick={() => void handleDeleteMemory(entry.id)}
                  >
                    <X size={12} aria-hidden="true" />
                  </button>
                </li>
              ))}
            </ul>
          )}
        </section>
      )}

      {receiptsOpen && (
        <NexReceiptsPanel
          receipts={receipts}
          privacyMode={privacyMode}
          runningAction={runningAction}
          onRunReadHud={() => void handleRunReadHud()}
        />
      )}

      <div className={styles.chatMessages} data-nex-chat-messages>
        {history.length === 0 ? (
          <p className={styles.emptyText}>{t('nexV2.chatEmpty')}</p>
        ) : (
          history.map((turn, idx) => (
            <div key={`${turn.role}-${idx}`} className={styles.chatMessage} data-role={turn.role}>
              <span className={styles.chatRole}>
                {turn.role === 'user' ? t('nexV2.chatUserRole') : t('nexV2.chatAssistantRole')}
              </span>
              <p className={styles.chatText}>{turn.content}</p>
              {turn.proposal && (
                <div className={styles.proposalBox} data-nex-proposal={turn.proposal.id || 'item'}>
                  <strong>{turn.proposal.title}</strong>
                  {turn.proposal.description && (
                    <p className={styles.emptyText}>{turn.proposal.description}</p>
                  )}
                  <div className={styles.proposalActions}>
                    <button
                      type="button"
                      className={styles.smallBtn}
                      data-nex-proposal-confirm
                      onClick={() => void handleConfirmProposal(idx, turn.proposal!)}
                    >
                      {t('nexV2.proposalConfirmBtn')}
                    </button>
                    <button
                      type="button"
                      className={styles.smallBtn}
                      data-nex-proposal-dismiss
                      onClick={() => void handleDismissProposal(idx, turn.proposal!)}
                    >
                      {t('nexV2.proposalDismissBtn')}
                    </button>
                  </div>
                </div>
              )}
            </div>
          ))
        )}
      </div>

      <form className={styles.chatForm} onSubmit={(e) => void handleSubmit(e)}>
        <input
          type="text"
          className={styles.chatInput}
          value={input}
          onChange={(event) => setInput(event.target.value)}
          placeholder={t('nexV2.chatPlaceholder')}
          aria-label={t('nexV2.chatPlaceholder')}
          data-nex-chat-input
        />
        {sending ? (
          <button
            type="button"
            className={styles.sendBtn}
            onClick={() => abortRef.current?.abort()}
          >
            {t('nexV2.chatCancel')}
          </button>
        ) : (
          <button
            type="submit"
            className={styles.sendBtn}
            disabled={!input.trim()}
            data-nex-chat-send
          >
            <Send size={14} aria-hidden="true" />
            <span>{t('nexV2.chatSend')}</span>
          </button>
        )}
      </form>
    </div>
  );
};

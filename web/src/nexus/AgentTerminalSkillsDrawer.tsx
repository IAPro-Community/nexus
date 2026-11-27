import React, { useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Check,
  CornerDownLeft,
  Copy,
  MessageSquare,
  Search,
  Send,
  Sparkles,
  X,
} from 'lucide-react';
import { ContextDrawer } from '../design-system';
import type { AgentTerminalSkill } from './agentTerminalTypes';
import styles from './AgentTerminal.module.scss';

export interface AgentTerminalSkillsDrawerProps {
  open: boolean;
  onClose: () => void;
  skills: AgentTerminalSkill[];
  skillsSearch: string;
  onSearch: (value: string) => void;
  selectedCategory: string;
  onCategory: (category: string) => void;
  categories: string[];
  selectedSkills: string[];
  onToggleSkill: (id: string) => void;
  onInsertSkill: (skill: AgentTerminalSkill) => void;
  onCopySkill: (skill: AgentTerminalSkill) => void;
  copiedSkillId: string | null;
  askPrompt: string;
  onAskPrompt: (value: string) => void;
  onSend: () => void;
  asking: boolean;
  askFeedback: string;
}

/**
 * Maestro skills drawer used by the agent terminal. Pure presentation: every
 * state change is reported to the terminal, which keeps the WebSocket and PTY
 * ownership in one place.
 */
export const AgentTerminalSkillsDrawer: React.FC<AgentTerminalSkillsDrawerProps> = ({
  open,
  onClose,
  skills,
  skillsSearch,
  onSearch,
  selectedCategory,
  onCategory,
  categories,
  selectedSkills,
  onToggleSkill,
  onInsertSkill,
  onCopySkill,
  copiedSkillId,
  askPrompt,
  onAskPrompt,
  onSend,
  asking,
  askFeedback,
}) => {
  const { t } = useTranslation();
  const filtered = useMemo(() => {
    const needle = skillsSearch.trim().toLowerCase();
    return skills.filter((skill) => {
      if (selectedCategory !== 'all' && skill.category !== selectedCategory) return false;
      if (!needle) return true;
      const haystack = [skill.id, skill.name, skill.description, ...(skill.triggers || [])]
        .filter(Boolean)
        .join(' ')
        .toLowerCase();
      return haystack.includes(needle);
    });
  }, [skills, skillsSearch, selectedCategory]);

  const feedbackIsError = askFeedback.includes('erro') || askFeedback.includes('failed');

  return (
    <ContextDrawer
      open={open}
      onClose={onClose}
      title={t('terminal.skillsDrawerTitle', 'Skills do Maestro')}
      description={t(
        'terminal.skillsDrawerDesc',
        'Catálogo de capacidades e automações integradas',
      )}
      width={420}
    >
      <div className={styles.skillsDrawerBody}>
        <div className={styles.skillsSearchBar}>
          <Search size={14} className="nx-text-muted" />
          <input
            type="text"
            placeholder={t(
              'terminal.skillsSearchPlaceholder',
              'Buscar por nome, comando ou descrição...',
            )}
            value={skillsSearch}
            onChange={(event) => onSearch(event.target.value)}
          />
          {skillsSearch && (
            <button
              type="button"
              className={styles.closeButton}
              onClick={() => onSearch('')}
              title={t('common.clear', 'Limpar')}
            >
              <X size={12} />
            </button>
          )}
        </div>

        {categories.length > 1 && (
          <div className={styles.skillsCategoryBar}>
            {categories.map((category) => (
              <button
                key={category}
                type="button"
                className={styles.categoryChip}
                data-active={selectedCategory === category ? 'true' : 'false'}
                onClick={() => onCategory(category)}
              >
                {category === 'all' ? t('common.all', 'Todas') : category}
              </button>
            ))}
          </div>
        )}

        <div className={styles.skillsList}>
          {filtered.length === 0 ? (
            <div className="nx-text-muted nx-p-4 nx-text-center nx-text-sm">
              {skills.length === 0
                ? t('terminal.skillsLoading', 'Carregando catálogo dinâmico de skills...')
                : t('terminal.skillsEmpty', 'Nenhuma skill encontrada para o filtro atual.')}
            </div>
          ) : (
            filtered.map((skill) => {
              const active = selectedSkills.includes(skill.id);
              const copied = copiedSkillId === skill.id;
              return (
                <SkillCard
                  key={skill.id}
                  skill={skill}
                  active={active}
                  copied={copied}
                  onCopy={() => onCopySkill(skill)}
                  onInsert={() => onInsertSkill(skill)}
                  onToggle={() => onToggleSkill(skill.id)}
                />
              );
            })
          )}
        </div>

        <div className={styles.skillsDrawerFooter}>
          <div className={styles.composerHeader}>
            <span className={styles.composerTitle}>
              <MessageSquare size={13} /> {t('terminal.sendInstructionTitle')}
            </span>
            {selectedSkills.length > 0 && (
              <span className={styles.skillsLabel}>
                {selectedSkills.length}/3 {t('terminal.skillsSelected', 'selecionadas')}
              </span>
            )}
          </div>
          <div className={styles.composerRow}>
            <input
              type="text"
              className={`nx-input ${styles.composerInput}`}
              placeholder={t('terminal.sendInstructionPlaceholder')}
              value={askPrompt}
              onChange={(event) => onAskPrompt(event.target.value)}
              onKeyDown={(event) => {
                if (event.key === 'Enter' && !event.shiftKey) {
                  event.preventDefault();
                  onSend();
                }
              }}
            />
            <button
              type="button"
              className="nx-button"
              data-tone="brand"
              data-size="sm"
              disabled={asking || !askPrompt.trim()}
              onClick={onSend}
            >
              <Send size={12} />
              <span>{asking ? t('terminal.sending') : t('terminal.send')}</span>
            </button>
          </div>
          {askFeedback && (
            <span
              className={feedbackIsError ? styles.feedbackAlert : styles.feedbackSuccess}
              role="status"
            >
              {askFeedback}
            </span>
          )}
        </div>
      </div>
    </ContextDrawer>
  );
};

const SkillCard: React.FC<{
  skill: AgentTerminalSkill;
  active: boolean;
  copied: boolean;
  onCopy: () => void;
  onInsert: () => void;
  onToggle: () => void;
}> = ({ skill, active, copied, onCopy, onInsert, onToggle }) => {
  const { t } = useTranslation();
  return (
    <div className={styles.skillCard}>
      <div className={styles.skillCardHeader}>
        <span className={styles.skillCardName}>
          <Sparkles size={12} className="nx-text-accent" />
          {skill.name || skill.id}
        </span>
        <div className={styles.skillBadges}>
          {skill.category && <span className={styles.skillCategoryBadge}>{skill.category}</span>}
          <span className={styles.skillRiskBadge} data-risk={skill.risk || 'safe'}>
            {skill.risk || 'safe'}
          </span>
        </div>
      </div>
      {skill.description && <p className={styles.skillCardDesc}>{skill.description}</p>}
      {skill.triggers && skill.triggers.length > 0 && (
        <div className={styles.skillTriggers}>
          {skill.triggers.map((trigger) => (
            <span key={trigger} className={styles.skillTriggerTag}>
              {trigger}
            </span>
          ))}
        </div>
      )}
      <div className={styles.skillCardActions}>
        <button
          type="button"
          className={styles.skillActionBtn}
          onClick={onCopy}
          title={t('terminal.copySkillTitle', 'Copiar comando')}
        >
          {copied ? <Check size={11} className="text-[var(--nx-success)]" /> : <Copy size={11} />}
          <span>{copied ? t('terminal.copied', 'Copiado') : t('terminal.copy', 'Copiar')}</span>
        </button>
        <button
          type="button"
          className={styles.skillActionBtn}
          data-primary="true"
          onClick={onInsert}
          title={t('terminal.insertSkillTitle', 'Inserir comando no terminal')}
        >
          <CornerDownLeft size={11} />
          <span>{t('terminal.insert', 'Inserir')}</span>
        </button>
        <button
          type="button"
          className={styles.skillActionBtn}
          onClick={onToggle}
          data-active={active ? 'true' : 'false'}
          title={t('terminal.selectForPrompt', 'Vincular para prompt (máx 3)')}
        >
          {active ? `✓ ${t('terminal.selected', 'Prompt')}` : '+ Prompt'}
        </button>
      </div>
    </div>
  );
};

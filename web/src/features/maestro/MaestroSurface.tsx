import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  BrainCircuit,
  Check,
  CheckCircle2,
  Copy,
  Cpu,
  LoaderCircle,
  Layers,
  RefreshCw,
  Search,
  ShieldAlert,
  Sparkles,
  Terminal,
  X,
} from 'lucide-react';
import { nexus } from '../../nexus/api';
import type { MaestroSkill, MaestroCatalog } from '../../types';
import styles from './MaestroSurface.module.scss';

export interface MaestroStatusData {
  available: boolean;
  mode?: string;
  error?: string;
  capabilities?: {
    version?: string;
    modes?: string[];
    skills?: MaestroSkill[];
  };
}

export const MaestroSurface: React.FC = () => {
  const { t } = useTranslation();
  const [status, setStatus] = useState<MaestroStatusData | null>(null);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedCategory, setSelectedCategory] = useState('all');
  const [copiedSkillId, setCopiedSkillId] = useState<string | null>(null);
  const [catalog, setCatalog] = useState<MaestroCatalog | null>(null);

  const skillPrompt = (skill: MaestroSkill) =>
    skill.prompt?.trim() ||
    [
      `Use the Maestro skill "${skill.name || skill.id}" for this task.`,
      skill.description,
      skill.triggers?.length ? `Relevant triggers: ${skill.triggers.join(', ')}` : '',
      'Respect the skill risk level, project instructions, verification requirements, and rollback constraints.',
    ]
      .filter(Boolean)
      .join('\n\n');

  const loadData = useCallback(
    async (isManual = false) => {
      if (isManual) setRefreshing(true);
      else setLoading(true);

      try {
        const res = await Promise.race([
          Promise.all([nexus.getMaestroStatus(), nexus.getMaestroCatalog()]),
          new Promise<never>((_, reject) =>
            window.setTimeout(
              () =>
                reject(
                  new Error(t('maestroSurface.timeout', 'Tempo limite ao consultar o Maestro.')),
                ),
              10000,
            ),
          ),
        ]);
        setStatus(res[0]);
        setCatalog(res[1]);
      } catch (e) {
        setStatus({
          available: false,
          error: e instanceof Error ? e.message : String(e),
        });
      } finally {
        setLoading(false);
        setRefreshing(false);
      }
    },
    [t],
  );

  useEffect(() => {
    void loadData();
  }, [loadData]);

  const skills = useMemo(() => {
    const discovered = catalog?.operational || status?.capabilities?.skills;
    return Array.isArray(discovered) ? discovered : [];
  }, [catalog, status]);

  const categories = useMemo(() => {
    const counts: Record<string, number> = { all: skills.length };
    skills.forEach((s) => {
      const cat = (s.category || 'other').toLowerCase();
      counts[cat] = (counts[cat] || 0) + 1;
    });

    const list = Object.keys(counts)
      .filter((k) => k !== 'all')
      .sort();

    return [
      { id: 'all', label: t('common.all', 'Todas'), count: counts.all },
      ...list.map((c) => ({
        id: c,
        label: c.charAt(0).toUpperCase() + c.slice(1),
        count: counts[c] || 0,
      })),
    ];
  }, [skills, t]);

  const filteredSkills = useMemo(() => {
    const q = searchQuery.trim().toLowerCase();
    return skills.filter((s) => {
      const cat = (s.category || 'other').toLowerCase();
      const matchCat = selectedCategory === 'all' || cat === selectedCategory.toLowerCase();
      if (!matchCat) return false;

      if (!q) return true;
      const name = (s.name || s.id).toLowerCase();
      const desc = (s.description || '').toLowerCase();
      const trigs = (s.triggers || []).join(' ').toLowerCase();
      const aliases = (s.aliases || []).join(' ').toLowerCase();
      return name.includes(q) || desc.includes(q) || trigs.includes(q) || aliases.includes(q);
    });
  }, [skills, searchQuery, selectedCategory]);

  const handleCopySkillPrompt = (skill: MaestroSkill) => {
    const text = skillPrompt(skill);
    void navigator.clipboard.writeText(text);
    setCopiedSkillId(skill.id);
    setTimeout(() => setCopiedSkillId(null), 1800);
  };

  const isLoading = loading && !status;
  const isDegraded = Boolean(status && !status.available);
  const version = status?.capabilities?.version || '0.2.4';

  return (
    <main
      className={styles.surface}
      aria-labelledby="maestro-page-title"
      data-testid="maestro-surface"
    >
      <div className={styles.container}>
        {/* Hero Section */}
        <header className={styles.hero}>
          <div className={styles.heroContent}>
            <span className={styles.eyebrow}>
              <span className={styles.eyebrowDot} aria-hidden="true" />
              {t('maestroSurface.eyebrow', 'Catálogo operacional')}
            </span>
            <div className={styles.heroTitleRow}>
              <Sparkles className={styles.heroIcon} size={22} />
              <h1 className={styles.heroTitle} id="maestro-page-title">
                Maestro · Contexto e skills
              </h1>
            </div>
            <p className={styles.heroDesc}>
              {t(
                'maestroSurface.heroDesc',
                'Prepare contexto durável e contratos de skills para revisar o trabalho antes de enviá-lo ao Agente.',
              )}
            </p>

            <div className={styles.heroBadges}>
              <span className={styles.statusPill} data-degraded={isDegraded ? 'true' : 'false'}>
                {isDegraded ? <ShieldAlert size={12} /> : <CheckCircle2 size={12} />}
                <span>
                  {isDegraded
                    ? t('maestroSurface.degraded', 'Modo Standalone / Degradado')
                    : t('maestroSurface.connected', 'Catálogo disponível')}
                </span>
              </span>

              <span className={styles.versionPill} title="Versão do motor Maestro">
                <Cpu size={12} />
                <span>v{version}</span>
              </span>

              <span className={styles.versionPill} title="Total de capacidades ativas">
                <Layers size={12} />
                <span>
                  {catalog?.counts.operational ?? skills.length}{' '}
                  {t('maestroSurface.skillsLoaded', 'disponíveis agora')}
                </span>
              </span>
            </div>
          </div>

          <div className={styles.heroActions}>
            <button
              type="button"
              className={styles.refreshBtn}
              onClick={() => void loadData(true)}
              disabled={refreshing || loading}
              title={t('common.refresh', 'Atualizar')}
            >
              <RefreshCw size={13} className={refreshing ? 'nx-spin' : ''} />
              <span>
                {refreshing
                  ? t('common.loading', 'Atualizando…')
                  : t('common.refresh', 'Atualizar')}
              </span>
            </button>
          </div>
        </header>

        {/* Overview Cards */}
        <section
          className={styles.overviewGrid}
          aria-label={t('maestroSurface.overviewLabel', 'Como o Maestro funciona')}
        >
          <div className={styles.infoCard}>
            <div className={styles.infoCardTitle}>
              <BrainCircuit size={16} className="nx-text-accent" />
              <span>{t('maestroSurface.cardGovernanceTitle', 'Governança & Persistência')}</span>
            </div>
            <p className={styles.infoCardText}>
              {t(
                'maestroSurface.cardGovernanceDesc',
                'Coordena múltiplos agentes com continuidade real de contexto (DEV/WORKLOG.md), isolamento seguro de workspaces e supervisão em tempo real.',
              )}
            </p>
          </div>

          <div className={styles.infoCard}>
            <div className={styles.infoCardTitle}>
              <Sparkles size={16} className="nx-text-amber-400" />
              <span>{t('maestroSurface.cardDiscoveryTitle', 'Catálogo 100% Dinâmico')}</span>
            </div>
            <p className={styles.infoCardText}>
              {t(
                'maestroSurface.cardDiscoveryDesc',
                'As capacidades e instruções são carregadas em tempo de execução via ~/.orquestrador/SKILLS_MANIFEST.json. Nenhuma skill é codificada de forma estática no Nexus.',
              )}
            </p>
          </div>

          <div className={styles.infoCard}>
            <div className={styles.infoCardTitle}>
              <Terminal size={16} className="nx-text-emerald-400" />
              <span>{t('maestroSurface.cardExecutionTitle', 'Execução no Terminal & YOLO')}</span>
            </div>
            <p className={styles.infoCardText}>
              {t(
                'maestroSurface.cardExecutionDesc',
                'Injete triggers diretamente nas abas de terminal PTY com um clique, ou acione automações com validação de segurança Safe vs YOLO.',
              )}
            </p>
          </div>
        </section>

        {/* Catalog Section */}
        <section className={styles.catalogSection} aria-labelledby="maestro-catalog-title">
          <div className={styles.catalogHeader}>
            <div className={styles.catalogHeadingRow}>
              <div>
                <span className={styles.sectionEyebrow}>
                  {t('maestroSurface.catalogEyebrow', 'Catálogo')}
                </span>
                <h2 className={styles.catalogTitle} id="maestro-catalog-title">
                  {t('maestroSurface.catalogTitle', 'Skills disponíveis')}
                </h2>
              </div>
              <span className={styles.catalogCount} aria-live="polite">
                {t('maestroSurface.filteredCount', '{{visible}} de {{total}} disponíveis', {
                  visible: filteredSkills.length,
                  total: skills.length,
                })}
              </span>
            </div>
            <div className={styles.catalogToolbar}>
              <label className={styles.searchBox}>
                <Search size={15} className="nx-text-muted" />
                <input
                  type="text"
                  placeholder={t(
                    'maestroSurface.searchPlaceholder',
                    'Buscar skills por nome, categoria, trigger ou descrição...',
                  )}
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                />
                {searchQuery && (
                  <button
                    type="button"
                    className="nx-btn-ghost"
                    onClick={() => setSearchQuery('')}
                    title={t('common.clear', 'Limpar')}
                  >
                    <X size={14} />
                  </button>
                )}
              </label>
            </div>

            <div
              className={styles.categoryFilterRow}
              role="toolbar"
              aria-label={t('maestroSurface.categoryLabel', 'Filtrar por categoria')}
            >
              {categories.map((cat) => (
                <button
                  key={cat.id}
                  type="button"
                  className={styles.categoryButton}
                  data-active={selectedCategory === cat.id ? 'true' : 'false'}
                  onClick={() => setSelectedCategory(cat.id)}
                >
                  <span>{cat.label}</span>
                  <span className={styles.categoryCount}>{cat.count}</span>
                </button>
              ))}
            </div>
          </div>

          {isLoading ? (
            <div className={styles.emptyState} role="status">
              <LoaderCircle size={28} className={styles.loadingIcon} />
              <p>{t('maestroSurface.loadingCatalog', 'Carregando catálogo de skills…')}</p>
            </div>
          ) : status?.error ? (
            <div className={styles.emptyState} data-tone="danger" role="alert">
              <ShieldAlert size={28} />
              <strong>
                {t('maestroSurface.loadError', 'Não foi possível carregar o Maestro')}
              </strong>
              <p>{status.error}</p>
              <button
                type="button"
                className={styles.retryButton}
                onClick={() => void loadData(true)}
              >
                <RefreshCw size={13} /> {t('common.retry', 'Tentar novamente')}
              </button>
            </div>
          ) : filteredSkills.length === 0 ? (
            <div className={styles.emptyState}>
              <Sparkles size={28} />
              <p>
                {skills.length === 0
                  ? t(
                      'maestroSurface.emptyCatalog',
                      'Nenhuma skill descoberta ou manifesto inacessível.',
                    )
                  : t(
                      'maestroSurface.emptyFilter',
                      'Nenhuma skill encontrada com o termo ou categoria selecionados.',
                    )}
              </p>
            </div>
          ) : (
            <div className={styles.skillsGrid}>
              {filteredSkills.map((skill) => {
                const isCopied = copiedSkillId === skill.id;
                return (
                  <article key={skill.id} className={styles.skillItem}>
                    <div className={styles.skillItemTop}>
                      <span className={styles.skillItemTitle}>
                        <Sparkles size={14} className="nx-text-accent" />
                        {skill.name || skill.id}
                      </span>

                      <div className={styles.skillBadges}>
                        {skill.category && (
                          <span className={styles.categoryTag}>{skill.category}</span>
                        )}
                        <span className={styles.riskTag} data-risk={skill.risk || 'safe'}>
                          {skill.risk || 'safe'}
                        </span>
                      </div>
                    </div>

                    {skill.description && <p className={styles.skillDesc}>{skill.description}</p>}

                    {skill.triggers && skill.triggers.length > 0 && (
                      <div className={styles.skillTriggers}>
                        {skill.triggers.map((trig) => (
                          <span key={trig} className={styles.triggerPill}>
                            {trig}
                          </span>
                        ))}
                      </div>
                    )}

                    <details className={styles.promptDetails}>
                      <summary>{t('maestroSurface.promptSummary', 'Como usar esta skill')}</summary>
                      <pre>{skillPrompt(skill)}</pre>
                    </details>

                    <div className={styles.skillBottomRow}>
                      <button
                        type="button"
                        className={styles.copyTriggerBtn}
                        onClick={() => handleCopySkillPrompt(skill)}
                        title={t(
                          'maestroSurface.copyPromptTitle',
                          'Copiar contexto completo da skill',
                        )}
                      >
                        {isCopied ? (
                          <Check size={12} className="nx-text-emerald-400" />
                        ) : (
                          <Copy size={12} />
                        )}
                        <span>
                          {isCopied
                            ? t('common.copied', 'Copiado!')
                            : t('maestroSurface.copyPrompt', 'Copiar contexto de uso')}
                        </span>
                      </button>
                    </div>
                  </article>
                );
              })}
            </div>
          )}
        </section>
      </div>
    </main>
  );
};

import React from 'react';
import { useTranslation } from 'react-i18next';
import { Activity, Gauge, History, Scale, Shuffle } from 'lucide-react';
import { Badge, Button, Dialog } from '../../design-system';
import type { RecommendationResult } from '../../types';
import styles from './SchedulerExplainModal.module.scss';

export interface SchedulerExplainModalProps {
  isOpen: boolean;
  onClose: () => void;
  result: RecommendationResult | null;
  policy: string;
}

export const SchedulerExplainModal: React.FC<SchedulerExplainModalProps> = ({
  isOpen,
  onClose,
  result,
  policy,
}) => {
  const { t } = useTranslation();

  if (!isOpen) return null;

  return (
    <Dialog
      open={isOpen}
      onClose={onClose}
      title={t('usage.schedulerExplainTitle', 'Explicabilidade do Scheduler')}
    >
      <div className={styles.root}>
        <p className={styles.intro}>{t('usage.schedulerExplainSubtitle')}</p>

        {result?.explanation ? (
          <div className={styles.policyCard}>
            <div className={styles.policyHeader}>
              <strong>{t('usage.recommendTitle', 'Recomendação ativa')}</strong>
              <Badge tone="brand">
                Score: {result.recommended?.total_score?.toFixed(1) || '—'}
              </Badge>
            </div>
            <p>{result.explanation}</p>
          </div>
        ) : null}

        <div className={styles.factorsList}>
          <div className={styles.factorItem}>
            <Scale size={18} className={styles.factorIcon} />
            <div className={styles.factorCopy}>
              <strong>{t('usage.factorFallbackPriority')}</strong>
              <span>
                Cada provedor possui uma prioridade base configurada. Provedores com menor ordem de
                fallback são preferidos quando saudáveis.
              </span>
            </div>
          </div>

          <div className={styles.factorItem}>
            <Gauge size={18} className={styles.factorIcon} />
            <div className={styles.factorCopy}>
              <strong>{t('usage.factorQuotaRemaining')}</strong>
              <span>
                A capacidade remanescente nas janelas semanais e de 5 horas contribui diretamente
                para a pontuação do recurso.
              </span>
            </div>
          </div>

          <div className={styles.factorItem}>
            <Activity size={18} className={styles.factorIcon} />
            <div className={styles.factorCopy}>
              <strong>{t('usage.factorHealth')}</strong>
              <span>
                Taxa de erros recente, status de rate limit e penalidades de cooldown reduzem
                drasticamente o score de contas degradadas.
              </span>
            </div>
          </div>

          <div className={styles.factorItem}>
            <Shuffle size={18} className={styles.factorIcon} />
            <div className={styles.factorCopy}>
              <strong>{t('usage.factorSwitchCost')}</strong>
              <span>
                Manter o mesmo provedor ativo economiza trocas de contexto e preserva ferramentas
                locais vinculadas.
              </span>
            </div>
          </div>

          <div className={styles.factorItem}>
            <History size={18} className={styles.factorIcon} />
            <div className={styles.factorCopy}>
              <strong>{t('usage.factorContinuity')}</strong>
              <span>
                Tarefas em andamento com o mesmo Lead Agent recebem bônus de continuidade para
                manter o histórico da conversa.
              </span>
            </div>
          </div>
        </div>

        <div className={styles.policyCard}>
          <div className={styles.policyHeader}>
            <strong>{t('usage.activePolicy', 'Política ativa')}</strong>
            <Badge tone="brand">{policy || 'BALANCED'}</Badge>
          </div>
          <p>{t('surfaces.allocationBody')}</p>
        </div>

        <div className={styles.footerActions}>
          <Button tone="brand" onClick={onClose}>
            {t('common.close', 'Fechar')}
          </Button>
        </div>
      </div>
    </Dialog>
  );
};

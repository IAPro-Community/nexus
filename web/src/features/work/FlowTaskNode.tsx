import { memo } from 'react';
import { Handle, Position, type Node, type NodeProps } from '@xyflow/react';
import { Bot, CircleCheck, CircleDot } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { Badge } from '../../design-system';
import type { FlowStepModel } from './flowModel';
import styles from './FlowCanvas.module.scss';
import { asArray } from '../../lib/safeArray';

export type FlowTaskNodeData = { step: FlowStepModel; selected: boolean };

export const FlowTaskNode = memo(({ data }: NodeProps<Node<FlowTaskNodeData, 'task'>>) => {
  const { t } = useTranslation();
  const { step, selected } = data;
  return (
    <article className={styles.node} data-selected={selected ? 'true' : 'false'}>
      <Handle
        type="target"
        position={Position.Left}
        aria-label={t('flow.node.connectTo', 'Connect to {{title}}', { title: step.title })}
      />
      <div className={styles.nodeTitle}>
        {step.status === 'VERIFIED' ? <CircleCheck size={13} /> : <CircleDot size={13} />}
        <strong>{step.title || step.id}</strong>
      </div>
      <div className={styles.nodeMeta}>
        <Badge tone="default">{step.assignmentStrategy}</Badge>
        {step.parallelGroup && <Badge tone="brand">{step.parallelGroup}</Badge>}
      </div>
      <small>
        {asArray<string>(step.dependencies).length
          ? t('flow.node.after', 'after {{dependencies}}', {
              dependencies: asArray<string>(step.dependencies).join(', '),
            })
          : t('flow.node.entry', 'entry node')}
      </small>
      <span className={styles.nodeAgent}>
        <Bot size={11} />
        {step.agentId || step.role || t('flow.node.autoResource', 'Auto resource')}
      </span>
      <Handle
        type="source"
        position={Position.Right}
        aria-label={t('flow.node.connectFrom', 'Connect from {{title}}', { title: step.title })}
      />
    </article>
  );
});

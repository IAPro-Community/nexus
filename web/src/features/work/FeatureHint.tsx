import React from 'react';
import { CircleHelp } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { IconButton, Tooltip } from '../../design-system';

/** Accessible contextual help reusing the existing Radix Tooltip primitive. */
export const FeatureHint: React.FC<{
  label: string;
  content: string;
}> = ({ label, content }) => {
  const { t } = useTranslation();
  return (
    <Tooltip content={t(content)} side="top">
      <IconButton label={t(label)} title={t(label)}>
        <CircleHelp size={14} aria-hidden="true" />
      </IconButton>
    </Tooltip>
  );
};

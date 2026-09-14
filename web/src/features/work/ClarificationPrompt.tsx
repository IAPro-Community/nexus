import React, { useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Button, Input, Select } from '../../design-system';
import styles from './ClarificationPrompt.module.scss';

export type ClarificationPromptItem = {
  id: string;
  question: string;
  rationale?: string;
  severity?: string;
  suggested_options?: string[] | null;
  default_choice?: string;
};

/** Keyboard focus wrap for option listbox (ArrowUp/ArrowDown). */
export function nextOptionFocusIndex(current: number, length: number, delta: number): number {
  if (length <= 0) return 0;
  return (current + delta + length) % length;
}

export function resolveClarificationAnswer(
  useOther: boolean,
  custom: string,
  selected: string,
): string {
  return (useOther ? custom : selected).trim();
}

export const ClarificationPrompt: React.FC<{
  item: ClarificationPromptItem;
  busy?: boolean;
  onAnswer: (answer: string) => void;
  onDismiss?: () => void;
  onRemember?: (answer: string) => void;
  allowDismiss?: boolean;
  showRemember?: boolean;
}> = ({
  item,
  busy = false,
  onAnswer,
  onDismiss,
  onRemember,
  allowDismiss = false,
  showRemember = false,
}) => {
  const { t } = useTranslation();
  const options = useMemo(
    () => (Array.isArray(item.suggested_options) ? item.suggested_options.filter(Boolean) : []),
    [item.suggested_options],
  );
  const [selected, setSelected] = useState(item.default_choice || options[0] || '');
  const [custom, setCustom] = useState('');
  const [useOther, setUseOther] = useState(options.length === 0);
  const [focusIndex, setFocusIndex] = useState(0);

  useEffect(() => {
    setSelected(item.default_choice || options[0] || '');
    setCustom('');
    setUseOther(options.length === 0);
    setFocusIndex(0);
    // Depend on stable primitives — array identity from parents must not loop.
  }, [item.id, item.default_choice, options.join('\u0001'), options.length]);

  const answerValue = resolveClarificationAnswer(useOther, custom, selected);
  const canSubmit = Boolean(answerValue) && !busy;

  const submit = () => {
    if (!canSubmit) return;
    onAnswer(answerValue);
  };

  const onKeyDown = (event: React.KeyboardEvent) => {
    if (!options.length || useOther) {
      if (event.key === 'Enter' && !event.shiftKey) {
        event.preventDefault();
        submit();
      }
      return;
    }
    if (event.key === 'ArrowDown') {
      event.preventDefault();
      const next = nextOptionFocusIndex(focusIndex, options.length, 1);
      setFocusIndex(next);
      setSelected(options[next]);
    } else if (event.key === 'ArrowUp') {
      event.preventDefault();
      const next = nextOptionFocusIndex(focusIndex, options.length, -1);
      setFocusIndex(next);
      setSelected(options[next]);
    } else if (event.key === 'Enter') {
      event.preventDefault();
      submit();
    }
  };

  return (
    <section
      className={styles.root}
      data-severity={item.severity || 'RECOMMENDED'}
      aria-label={item.question}
      onKeyDown={onKeyDown}
    >
      <strong className={styles.question}>{item.question}</strong>
      {item.rationale ? <p className={styles.rationale}>{item.rationale}</p> : null}

      {options.length > 0 && !useOther ? (
        <div
          className={styles.options}
          role="listbox"
          aria-label={t('flow.chooseOption', 'Choose…')}
        >
          {options.map((option, index) => (
            <button
              key={option}
              type="button"
              role="option"
              aria-selected={selected === option}
              className={styles.option}
              data-active={selected === option ? 'true' : undefined}
              data-focused={focusIndex === index ? 'true' : undefined}
              disabled={busy}
              onClick={() => {
                setSelected(option);
                setFocusIndex(index);
              }}
            >
              {option}
            </button>
          ))}
          <button
            type="button"
            className={styles.otherToggle}
            disabled={busy}
            onClick={() => setUseOther(true)}
          >
            {t('flow.otherOption', 'Other…')}
          </button>
        </div>
      ) : options.length > 0 ? (
        <Select
          value={selected}
          onChange={(value) => {
            if (value === '__other__') {
              setUseOther(true);
              return;
            }
            setSelected(value);
            setUseOther(false);
          }}
          placeholder={t('flow.chooseOption', 'Choose…')}
          options={[
            ...options.map((option) => ({ value: option, label: option })),
            { value: '__other__', label: t('flow.otherOption', 'Other…') },
          ]}
        />
      ) : null}

      {(useOther || options.length === 0) && (
        <Input
          value={custom}
          onChange={setCustom}
          placeholder={t('flow.answerRequired', 'Answer required')}
          disabled={busy}
        />
      )}

      <div className={styles.actions}>
        {allowDismiss && onDismiss ? (
          <Button size="sm" disabled={busy} onClick={onDismiss}>
            {t('work.composer.dismiss', 'Dismiss')}
          </Button>
        ) : null}
        {showRemember && onRemember ? (
          <Button size="sm" disabled={!canSubmit} onClick={() => onRemember(answerValue)}>
            {t('work.rememberThis', 'Remember this')}
          </Button>
        ) : null}
        <Button size="sm" tone="brand" disabled={!canSubmit} onClick={submit}>
          {t('work.composer.answer', 'Answer')}
        </Button>
      </div>
    </section>
  );
};

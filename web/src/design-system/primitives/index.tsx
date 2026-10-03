import React from 'react';
import {
  AlertCircle,
  CheckCircle2,
  Info,
  LoaderCircle,
  Search,
  TriangleAlert,
  X,
} from 'lucide-react';
import { useTranslation } from 'react-i18next';

export type Tone = 'default' | 'brand' | 'success' | 'warning' | 'danger' | 'info' | 'ghost';

export const Button: React.FC<
  React.ButtonHTMLAttributes<HTMLButtonElement> & {
    tone?: Tone;
    size?: 'sm' | 'md';
  }
> = ({ tone = 'default', size = 'md', className = '', type = 'button', children, ...props }) => (
  <button
    type={type}
    data-tone={tone}
    data-size={size}
    className={`nx-button ${className}`.trim()}
    {...props}
  >
    {children}
  </button>
);

export const IconButton: React.FC<
  React.ButtonHTMLAttributes<HTMLButtonElement> & { label: string; tone?: Tone }
> = ({ label, tone = 'ghost', className = '', children, ...props }) => (
  <button
    type="button"
    aria-label={label}
    title={props.title || label}
    data-tone={tone}
    className={`nx-icon-button ${className}`.trim()}
    {...props}
  >
    {children}
  </button>
);

export const Badge: React.FC<{
  tone?: Exclude<Tone, 'ghost'>;
  children: React.ReactNode;
  className?: string;
}> = ({ tone = 'default', children, className = '' }) => (
  <span data-tone={tone} className={`nx-badge ${className}`.trim()}>
    {children}
  </span>
);

export const Card: React.FC<React.HTMLAttributes<HTMLDivElement> & { interactive?: boolean }> = ({
  interactive,
  className = '',
  children,
  ...props
}) => (
  <div
    data-interactive={interactive ? 'true' : undefined}
    className={`nx-card ${className}`.trim()}
    {...props}
  >
    {children}
  </div>
);

export const Input: React.FC<
  Omit<React.InputHTMLAttributes<HTMLInputElement>, 'onChange'> & {
    value: string;
    onChange: (value: string) => void;
    onEnter?: () => void;
    mono?: boolean;
  }
> = ({ value, onChange, onEnter, mono, className = '', ...props }) => (
  <input
    data-mono={mono ? 'true' : undefined}
    className={`nx-input ${className}`.trim()}
    value={value}
    onChange={(event) => onChange(event.target.value)}
    onKeyDown={(event) => {
      if (event.key === 'Enter') onEnter?.();
    }}
    {...props}
  />
);

export const Textarea: React.FC<
  Omit<React.TextareaHTMLAttributes<HTMLTextAreaElement>, 'onChange'> & {
    value: string;
    onChange: (value: string) => void;
  }
> = ({ value, onChange, className = '', ...props }) => (
  <textarea
    className={`nx-textarea ${className}`.trim()}
    value={value}
    onChange={(event) => onChange(event.target.value)}
    {...props}
  />
);

export const Select: React.FC<{
  value: string;
  onChange: (value: string) => void;
  options?: { value: string; label: string; disabled?: boolean }[];
  label?: string;
  className?: string;
  placeholder?: string;
  disabled?: boolean;
  id?: string;
  name?: string;
  style?: React.CSSProperties;
  selectStyle?: React.CSSProperties;
  selectClassName?: string;
  'aria-label'?: string;
  children?: React.ReactNode;
}> = ({
  value,
  onChange,
  options,
  label,
  className = '',
  placeholder,
  disabled,
  id,
  name,
  style,
  selectStyle,
  selectClassName = '',
  'aria-label': ariaLabel,
  children,
}) => {
  const selectElement = (
    <select
      id={id}
      name={name}
      disabled={disabled}
      aria-label={ariaLabel || (typeof label === 'string' ? label : undefined)}
      className={`nx-select ${selectClassName}`.trim()}
      style={selectStyle}
      value={value}
      onChange={(event) => onChange(event.target.value)}
    >
      {placeholder && !options?.some((option) => option.value === '') && (
        <option value="" disabled>
          {placeholder}
        </option>
      )}
      {options
        ? options.map((option) => (
            <option key={option.value} value={option.value} disabled={option.disabled}>
              {option.label}
            </option>
          ))
        : children}
    </select>
  );

  if (label) {
    return (
      <label className={`nx-select-wrap ${className}`.trim()} style={style}>
        <span>{label}</span>
        {selectElement}
      </label>
    );
  }

  return className || style ? (
    <div className={`nx-select-container ${className}`.trim()} style={style}>
      {selectElement}
    </div>
  ) : (
    selectElement
  );
};

export const Switch: React.FC<{
  checked: boolean;
  onChange: (checked: boolean) => void;
  label: string;
  description?: string;
}> = ({ checked, onChange, label, description }) => (
  <label className="nx-switch-row">
    <button
      type="button"
      role="switch"
      aria-checked={checked}
      className="nx-switch"
      data-checked={checked ? 'true' : 'false'}
      onClick={() => onChange(!checked)}
    >
      <span />
    </button>
    <span className="nx-switch-copy">
      <strong>{label}</strong>
      {description && <small>{description}</small>}
    </span>
  </label>
);

export const Segmented: React.FC<{
  value: string;
  onChange: (value: string) => void;
  options: { value: string; label: string }[];
  ariaLabel: string;
}> = ({ value, onChange, options, ariaLabel }) => (
  <div className="nx-segmented" role="radiogroup" aria-label={ariaLabel}>
    {options.map((option) => (
      <button
        type="button"
        role="radio"
        aria-checked={value === option.value}
        data-active={value === option.value ? 'true' : 'false'}
        className="nx-segmented__item"
        key={option.value}
        onClick={() => onChange(option.value)}
      >
        {option.label}
      </button>
    ))}
  </div>
);

export const ChoiceGroup: React.FC<
  React.HTMLAttributes<HTMLDivElement> & {
    ariaLabel: string;
  }
> = ({ ariaLabel, className = '', children, onKeyDown, ...props }) => (
  <div
    role="radiogroup"
    aria-label={ariaLabel}
    className={`nx-choice-group ${className}`.trim()}
    onKeyDown={(event) => {
      onKeyDown?.(event);
      if (event.defaultPrevented) return;

      const navigationKeys = ['ArrowRight', 'ArrowDown', 'ArrowLeft', 'ArrowUp', 'Home', 'End'];
      if (!navigationKeys.includes(event.key)) return;
      if (!(event.target instanceof HTMLElement) || event.target.getAttribute('role') !== 'radio') {
        return;
      }

      const options = Array.from(
        event.currentTarget.querySelectorAll<HTMLButtonElement>(
          'button[role="radio"]:not(:disabled)',
        ),
      );
      if (options.length === 0) return;

      const currentIndex = options.indexOf(document.activeElement as HTMLButtonElement);
      let nextIndex = currentIndex >= 0 ? currentIndex : 0;

      if (event.key === 'Home') nextIndex = 0;
      else if (event.key === 'End') nextIndex = options.length - 1;
      else if (event.key === 'ArrowRight' || event.key === 'ArrowDown') {
        nextIndex = (nextIndex + 1) % options.length;
      } else {
        nextIndex = (nextIndex - 1 + options.length) % options.length;
      }

      event.preventDefault();
      options[nextIndex]?.focus();
      options[nextIndex]?.click();
    }}
    {...props}
  >
    {children}
  </div>
);

export const ChoiceOption: React.FC<
  Omit<
    React.ButtonHTMLAttributes<HTMLButtonElement>,
    'onSelect' | 'type' | 'role' | 'aria-checked'
  > & {
    selected: boolean;
    onSelect: () => void;
  }
> = ({ selected, onSelect, className = '', onClick, children, ...props }) => (
  <button
    type="button"
    role="radio"
    aria-checked={selected}
    data-selected={selected ? 'true' : 'false'}
    className={`nx-choice-option ${className}`.trim()}
    onClick={(event) => {
      onClick?.(event);
      if (!event.defaultPrevented) onSelect();
    }}
    {...props}
  >
    {children}
  </button>
);

export const Progress: React.FC<{ value: number; label: string }> = ({ value, label }) => (
  <div
    className="nx-progress"
    role="progressbar"
    aria-label={label}
    aria-valuemin={0}
    aria-valuemax={100}
    aria-valuenow={Math.round(value)}
  >
    <span style={{ width: `${Math.min(100, Math.max(0, value))}%` }} />
    <small>{label}</small>
  </div>
);

export const Spinner: React.FC<{ label?: string }> = ({ label }) => {
  const { t } = useTranslation();
  return (
    <div className="nx-spinner" role="status">
      <LoaderCircle size={16} aria-hidden="true" />
      <span>{label ?? t('common.loading')}</span>
    </div>
  );
};

export const EmptyState: React.FC<{
  icon?: React.ReactNode;
  title: string;
  hint?: string;
  action?: React.ReactNode;
}> = ({ icon, title, hint, action }) => (
  <div className="nx-empty-state">
    {icon}
    <strong>{title}</strong>
    {hint && <p>{hint}</p>}
    {action}
  </div>
);

export const InlineAlert: React.FC<{
  tone?: 'info' | 'success' | 'warning' | 'danger';
  title?: string;
  children: React.ReactNode;
}> = ({ tone = 'info', title, children }) => {
  const Icon =
    tone === 'success'
      ? CheckCircle2
      : tone === 'warning'
        ? TriangleAlert
        : tone === 'danger'
          ? AlertCircle
          : Info;
  return (
    <div className="nx-inline-alert" data-tone={tone} role={tone === 'danger' ? 'alert' : 'status'}>
      <Icon size={15} aria-hidden="true" />
      <div>
        {title && <strong>{title}</strong>}
        <span>{children}</span>
      </div>
    </div>
  );
};

export { ContextMenu, clampMenuPosition, contextMenuFromEvent } from './ContextMenu';
export type { ContextMenuItem, ContextMenuPoint } from './ContextMenu';
export { ContextDrawer, type ContextDrawerProps } from './ContextDrawer';

import * as RadixDialog from '@radix-ui/react-dialog';
import * as RadixAlertDialog from '@radix-ui/react-alert-dialog';
import * as RadixTooltip from '@radix-ui/react-tooltip';
import * as RadixDropdownMenu from '@radix-ui/react-dropdown-menu';

export const Dialog: React.FC<{
  open: boolean;
  title: string;
  onClose: () => void;
  children: React.ReactNode;
  wide?: boolean;
  full?: boolean;
  className?: string;
}> = ({ open, title, onClose, children, wide, full, className }) => {
  const { t } = useTranslation();
  return (
    <RadixDialog.Root
      open={open}
      onOpenChange={(isOpen) => {
        if (!isOpen) onClose();
      }}
    >
      <RadixDialog.Portal>
        <RadixDialog.Overlay className="nx-dialog-backdrop" />
        <RadixDialog.Content
          className={['nx-dialog', className].filter(Boolean).join(' ')}
          data-wide={wide ? 'true' : 'false'}
          data-full={full ? 'true' : 'false'}
        >
          <header>
            <RadixDialog.Title asChild>
              <h2>{title}</h2>
            </RadixDialog.Title>
            <RadixDialog.Close asChild>
              <IconButton label={t('common.closeDialog')} onClick={onClose}>
                <X size={15} />
              </IconButton>
            </RadixDialog.Close>
          </header>
          <div className="nx-dialog__body">{children}</div>
        </RadixDialog.Content>
      </RadixDialog.Portal>
    </RadixDialog.Root>
  );
};

export interface ConfirmDialogProps {
  open: boolean;
  title: string;
  description?: string;
  confirmLabel?: string;
  cancelLabel?: string;
  tone?: Tone;
  onConfirm: () => void;
  onCancel: () => void;
}

export const ConfirmDialog: React.FC<ConfirmDialogProps> = ({
  open,
  title,
  description,
  confirmLabel,
  cancelLabel,
  tone = 'danger',
  onConfirm,
  onCancel,
}) => {
  const { t } = useTranslation();
  return (
    <RadixAlertDialog.Root
      open={open}
      onOpenChange={(isOpen) => {
        if (!isOpen) onCancel();
      }}
    >
      <RadixAlertDialog.Portal>
        <RadixAlertDialog.Overlay className="nx-dialog-backdrop" />
        <RadixAlertDialog.Content className="nx-dialog nx-dialog--confirm">
          <header>
            <RadixAlertDialog.Title asChild>
              <h2>{title}</h2>
            </RadixAlertDialog.Title>
          </header>
          <div className="nx-dialog__body">
            {description && (
              <RadixAlertDialog.Description asChild>
                <p className="nx-confirm-desc">{description}</p>
              </RadixAlertDialog.Description>
            )}
            <div className="nx-dialog-actions">
              <RadixAlertDialog.Cancel asChild>
                <Button onClick={onCancel}>{cancelLabel ?? t('common.cancel', 'Cancelar')}</Button>
              </RadixAlertDialog.Cancel>
              <RadixAlertDialog.Action asChild>
                <Button tone={tone} onClick={onConfirm}>
                  {confirmLabel ?? t('common.confirm', 'Confirmar')}
                </Button>
              </RadixAlertDialog.Action>
            </div>
          </div>
        </RadixAlertDialog.Content>
      </RadixAlertDialog.Portal>
    </RadixAlertDialog.Root>
  );
};

export const TooltipProvider = RadixTooltip.Provider;

export const Tooltip: React.FC<{
  content: React.ReactNode;
  children: React.ReactElement;
  side?: 'top' | 'right' | 'bottom' | 'left';
  delayDuration?: number;
}> = ({ content, children, side = 'top', delayDuration = 300 }) => {
  if (!content) return children;
  return (
    <RadixTooltip.Provider delayDuration={delayDuration}>
      <RadixTooltip.Root delayDuration={delayDuration}>
        <RadixTooltip.Trigger asChild>{children}</RadixTooltip.Trigger>
        <RadixTooltip.Portal>
          <RadixTooltip.Content side={side} className="nx-tooltip-content" sideOffset={5}>
            {content}
            <RadixTooltip.Arrow className="nx-tooltip-arrow" />
          </RadixTooltip.Content>
        </RadixTooltip.Portal>
      </RadixTooltip.Root>
    </RadixTooltip.Provider>
  );
};

export { RadixDropdownMenu as DropdownMenu };

export const SearchInput: React.FC<{
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  autoFocus?: boolean;
}> = ({ value, onChange, placeholder, autoFocus }) => {
  const { t } = useTranslation();
  return (
    <label className="nx-search-input">
      <Search size={15} aria-hidden="true" />
      <input
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder ?? t('common.search')}
        autoFocus={autoFocus}
      />
    </label>
  );
};

export * from './ContextDrawer';

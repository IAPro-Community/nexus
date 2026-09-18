import React, { useEffect, useRef } from 'react';
import * as RadixDialog from '@radix-ui/react-dialog';
import { X } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { IconButton } from './index';

export interface ContextDrawerProps {
  open: boolean;
  onClose: () => void;
  title: string;
  description?: string;
  children: React.ReactNode;
  width?: number | string;
  side?: 'right' | 'left';
}

export const ContextDrawer: React.FC<ContextDrawerProps> = ({
  open,
  onClose,
  title,
  description,
  children,
  width = 380,
  side = 'right',
}) => {
  const { t } = useTranslation();
  const triggerRef = useRef<Element | null>(null);

  useEffect(() => {
    if (open) {
      triggerRef.current = document.activeElement;
    }
  }, [open]);

  return (
    <RadixDialog.Root
      open={open}
      onOpenChange={(isOpen) => {
        if (!isOpen) {
          onClose();
          if (triggerRef.current && (triggerRef.current as HTMLElement).focus) {
            (triggerRef.current as HTMLElement).focus();
          }
        }
      }}
    >
      <RadixDialog.Portal>
        <RadixDialog.Overlay className="nx-drawer-backdrop" />
        <RadixDialog.Content
          className="nx-context-drawer"
          data-side={side}
          style={{ width: typeof width === 'number' ? `${width}px` : width }}
          aria-describedby={description ? 'nx-drawer-desc' : undefined}
        >
          <header className="nx-context-drawer__header">
            <div>
              <RadixDialog.Title asChild>
                <h2 className="nx-context-drawer__title">{title}</h2>
              </RadixDialog.Title>
              {description && (
                <RadixDialog.Description asChild>
                  <p id="nx-drawer-desc" className="nx-context-drawer__desc">
                    {description}
                  </p>
                </RadixDialog.Description>
              )}
            </div>
            <RadixDialog.Close asChild>
              <IconButton label={t('common.closeDrawer', 'Fechar painel')} onClick={onClose}>
                <X size={15} />
              </IconButton>
            </RadixDialog.Close>
          </header>
          <div className="nx-context-drawer__body">{children}</div>
        </RadixDialog.Content>
      </RadixDialog.Portal>
    </RadixDialog.Root>
  );
};

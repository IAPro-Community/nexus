import React, { useEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { LayoutGrid, Paintbrush, RotateCcw } from "lucide-react";
import {
  DropdownMenu,
  IconButton,
  type ContextMenuPoint,
} from "../design-system";
import { useTranslation } from "react-i18next";
import { ARRANGE_MENU_PRESETS, type ArrangeMenuPreset } from "./arrangePresets";

const WINDOW_ACCENTS = [
  "var(--nx-info)",
  "var(--nx-success)",
  "var(--nx-warning)",
  "var(--nx-accent)",
  "var(--nx-accent-text)",
  "var(--nx-danger)",
];
const WINDOW_ICONS = ["⌘", "⚡", "◆", "●", "★", "◎"];

export const WindowChromeMenu: React.FC<{
  viewId: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  customTitle: string;
  allowAiTitle: boolean;
  accent: string;
  icon: string;
  onPatch: (chrome: {
    customTitle?: string;
    allowAiTitle?: boolean;
    accent?: string;
    icon?: string;
  }) => void;
  position?: ContextMenuPoint | null;
}> = ({
  open,
  onOpenChange,
  customTitle,
  allowAiTitle,
  accent,
  icon,
  onPatch,
  position,
}) => {
  const { t } = useTranslation();
  const rootRef = useRef<HTMLDivElement>(null);
  const [draft, setDraft] = useState(customTitle);
  const [menuPos, setMenuPos] = useState<{
    top: number;
    right?: number;
    left?: number;
  } | null>(null);

  useEffect(() => {
    // Escape discards the draft; reset the suppression when the menu reopens.
    if (open) skipBlurCommit.current = false;
    setDraft(customTitle);
  }, [customTitle, open]);

  useEffect(() => {
    if (!open) {
      setMenuPos(null);
      return;
    }
    const compute = () => {
      if (position) {
        setMenuPos({ top: position.y, left: position.x });
        return;
      }
      const anchor = rootRef.current?.querySelector("button");
      if (!anchor) return;
      const rect = anchor.getBoundingClientRect();
      setMenuPos({
        top: rect.bottom + 6,
        right: Math.max(8, window.innerWidth - rect.right),
      });
    };
    compute();
    const onPointerDown = (event: PointerEvent) => {
      const target = event.target as Node | null;
      if (!target) return;
      if (rootRef.current?.contains(target)) return;
      if ((target as Element).closest?.(".nx-desktop-window__chrome-menu"))
        return;
      onOpenChange(false);
    };
    const onKey = (event: KeyboardEvent) => {
      if (event.key === "Escape") onOpenChange(false);
    };
    window.addEventListener("pointerdown", onPointerDown, true);
    window.addEventListener("keydown", onKey);
    const onResize = () => compute();
    window.addEventListener("resize", onResize);
    window.addEventListener("scroll", onResize, true);
    return () => {
      window.removeEventListener("pointerdown", onPointerDown, true);
      window.removeEventListener("keydown", onKey);
      window.removeEventListener("resize", onResize);
      window.removeEventListener("scroll", onResize, true);
    };
  }, [open, onOpenChange, position]);

  const skipBlurCommit = useRef(false);
  const commitTitle = () => {
    if (skipBlurCommit.current) return;
    const trimmed = draft.trim();
    if (trimmed !== customTitle) {
      onPatch({ customTitle: trimmed });
    }
  };

  const menu =
    open && menuPos
      ? createPortal(
          <div
            className="nx-desktop-window__chrome-menu"
            role="dialog"
            aria-label={t("workspace.customizeWindow", "Personalizar janela")}
            style={{
              top: menuPos.top,
              ...(menuPos.left != null
                ? { left: menuPos.left }
                : { right: menuPos.right }),
            }}
            onPointerDown={(event) => event.stopPropagation()}
          >
            <label>
              {t("workspace.customTitle", "Nome personalizado")}
              <input
                value={draft}
                onChange={(event) => setDraft(event.target.value)}
                onBlur={commitTitle}
                onKeyDown={(event) => {
                  if (event.key === "Enter") {
                    commitTitle();
                    onOpenChange(false);
                  } else if (event.key === "Escape") {
                    event.stopPropagation();
                    skipBlurCommit.current = true;
                    setDraft(customTitle);
                    onOpenChange(false);
                  }
                }}
                placeholder={t(
                  "workspace.windowTitlePlaceholder",
                  "Identidade da janela",
                )}
                autoFocus
              />
            </label>
            <label className="nx-desktop-window__ai-title-toggle">
              <input
                type="checkbox"
                checked={allowAiTitle}
                onChange={(event) =>
                  onPatch({ allowAiTitle: event.target.checked })
                }
              />
              <span>
                {t(
                  "workspace.allowAiTitle",
                  "Permitir que a IA atualize o título",
                )}
              </span>
            </label>
            <p className="nx-desktop-window__ai-title-help">
              {t(
                "workspace.allowAiTitleHelp",
                "O nome personalizado permanece salvo. A IA pode atualizar o título principal quando esta opção estiver ativa.",
              )}
            </p>
            <div
              className="nx-desktop-window__chrome-row"
              aria-label={t("workspace.titleColor", "Cor do título")}
            >
              <button
                type="button"
                className="nx-desktop-window__swatch nx-desktop-window__swatch--clear"
                data-active={!accent ? "true" : "false"}
                title={t("workspace.noColor", "Sem cor")}
                onClick={() => {
                  onPatch({ accent: "" });
                  onOpenChange(false);
                }}
              />
              {WINDOW_ACCENTS.map((color) => (
                <button
                  key={color}
                  type="button"
                  className="nx-desktop-window__swatch"
                  data-active={accent === color ? "true" : "false"}
                  style={{ background: color }}
                  title={color}
                  onClick={() => {
                    onPatch({ accent: color });
                    onOpenChange(false);
                  }}
                />
              ))}
            </div>
            <div
              className="nx-desktop-window__chrome-row"
              aria-label={t("workspace.icon", "Ícone")}
            >
              <button
                type="button"
                className="nx-desktop-window__icon-pick"
                data-active={!icon ? "true" : "false"}
                title={t("workspace.noIcon", "Sem ícone")}
                onClick={() => {
                  onPatch({ icon: "" });
                  onOpenChange(false);
                }}
              >
                —
              </button>
              {WINDOW_ICONS.map((glyph) => (
                <button
                  key={glyph}
                  type="button"
                  className="nx-desktop-window__icon-pick"
                  data-active={icon === glyph ? "true" : "false"}
                  onClick={() => {
                    onPatch({ icon: glyph });
                    onOpenChange(false);
                  }}
                >
                  {glyph}
                </button>
              ))}
            </div>
          </div>,
          document.body,
        )
      : null;

  return (
    <div className="nx-desktop-window__chrome" ref={rootRef}>
      <IconButton
        label={t("workspace.customizeWindow", "Personalizar janela")}
        aria-expanded={open}
        onClick={(event) => {
          event.stopPropagation();
          onOpenChange(!open);
        }}
      >
        <Paintbrush size={13} />
      </IconButton>
      {menu}
    </div>
  );
};

export const ArrangeLayoutMenu: React.FC<{
  activePreset: ArrangeMenuPreset;
  onSelect: (preset: ArrangeMenuPreset) => void;
  onReset?: () => void;
}> = ({ activePreset, onSelect, onReset }) => {
  const { t } = useTranslation();

  return (
    <div className="nx-arrange-menu">
      <DropdownMenu.Root>
        <DropdownMenu.Trigger asChild>
          <button
            type="button"
            className="nx-arrange-menu__trigger"
            title={t("workspace.arrangeWindows", "Arranjar janelas")}
            aria-label={t("workspace.arrangeWindows", "Arranjar janelas")}
          >
            <LayoutGrid size={12} />
            <span>{t("workspace.arrange", "Arranjar")}</span>
          </button>
        </DropdownMenu.Trigger>

        <DropdownMenu.Portal>
          <DropdownMenu.Content
            className="nx-arrange-menu__panel"
            aria-label={t("workspace.arrangeWindows", "Arranjar janelas")}
            align="end"
            sideOffset={6}
            collisionPadding={8}
          >
            <DropdownMenu.RadioGroup
              value={activePreset}
              onValueChange={(value) => onSelect(value as ArrangeMenuPreset)}
            >
              {ARRANGE_MENU_PRESETS.map((preset) => (
                <DropdownMenu.RadioItem
                  key={preset.id}
                  value={preset.id}
                  className="nx-arrange-menu__item"
                  data-active={activePreset === preset.id ? "true" : undefined}
                >
                  <span>{preset.label}</span>
                  <small>{preset.hint}</small>
                </DropdownMenu.RadioItem>
              ))}
            </DropdownMenu.RadioGroup>

            {onReset && (
              <>
                <DropdownMenu.Separator className="nx-arrange-menu__separator" />
                <DropdownMenu.Item
                  className="nx-arrange-menu__item"
                  onSelect={() => onReset()}
                >
                  <span className="nx-arrange-menu__reset-label">
                    <RotateCcw size={12} />
                    <span>
                      {t("workspace.resetLayout", "Redefinir layout")}
                    </span>
                  </span>
                  <small>
                    {t("workspace.restoreDefault", "Restaurar padrão")}
                  </small>
                </DropdownMenu.Item>
              </>
            )}
          </DropdownMenu.Content>
        </DropdownMenu.Portal>
      </DropdownMenu.Root>
    </div>
  );
};

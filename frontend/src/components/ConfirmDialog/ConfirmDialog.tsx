import { LoaderCircle } from 'lucide-react';
import { AlertDialog } from 'radix-ui';
import { useRef } from 'react';
import type { ReactNode } from 'react';
import { Button } from '../Button/Button';
import styles from './ConfirmDialog.module.css';

export interface ConfirmDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** The question: "Delete sample folder 'Ideas'?" */
  title: string;
  /** What will happen: "It goes to the recycle bin. You can restore it." */
  children: ReactNode;
  /** Names the action, never "OK": "Delete sample folder". */
  confirmLabel: string;
  /** Names the safe choice, which has the focus: "Cancel", or "Keep editing" for unsaved changes. */
  cancelLabel?: string;
  /** Runs the action. The dialog stays open until you close it, so it can show `busy` meanwhile. */
  onConfirm: () => void;
  /** danger: the confirm button is filled red, for actions that remove something. */
  tone?: 'danger';
  /** The action is running: a second click on the confirm button is ignored. */
  busy?: boolean;
}

/**
 * Asks before an action that cannot simply be taken back (design system §6.5). The safe choice,
 * Cancel, has the focus, so pressing Enter never deletes anything by accident.
 */
export function ConfirmDialog({
  open,
  onOpenChange,
  title,
  children,
  confirmLabel,
  cancelLabel = 'Cancel',
  onConfirm,
  tone,
  busy = false,
}: ConfirmDialogProps) {
  // Radix gives the focus back only to its own trigger button. This dialog is opened by other
  // code (a menu item, a page button), so it remembers where the focus was and returns it there.
  const focusBeforeOpen = useRef<HTMLElement | null>(null);

  return (
    <AlertDialog.Root open={open} onOpenChange={onOpenChange}>
      <AlertDialog.Portal>
        <AlertDialog.Overlay className={styles.overlay} />
        <AlertDialog.Content
          className={styles.dialog}
          onOpenAutoFocus={() => {
            focusBeforeOpen.current = focusedElement();
          }}
          onCloseAutoFocus={(event) => {
            event.preventDefault();
            focusBeforeOpen.current?.focus();
          }}
        >
          <AlertDialog.Title className={styles.title}>{title}</AlertDialog.Title>
          <AlertDialog.Description asChild>
            <div className={styles.text}>{children}</div>
          </AlertDialog.Description>
          <div className={styles.actions}>
            <AlertDialog.Cancel asChild>
              <Button>{cancelLabel}</Button>
            </AlertDialog.Cancel>
            {/* A plain button, not AlertDialog.Action: that one would close the dialog at once. */}
            <Button
              variant="primary"
              className={tone === 'danger' ? styles.danger : undefined}
              icon={busy ? <LoaderCircle className={styles.spinner} /> : undefined}
              aria-busy={busy || undefined}
              onClick={() => {
                if (!busy) {
                  onConfirm();
                }
              }}
            >
              {confirmLabel}
            </Button>
          </div>
        </AlertDialog.Content>
      </AlertDialog.Portal>
    </AlertDialog.Root>
  );
}

function focusedElement(): HTMLElement | null {
  return document.activeElement instanceof HTMLElement ? document.activeElement : null;
}

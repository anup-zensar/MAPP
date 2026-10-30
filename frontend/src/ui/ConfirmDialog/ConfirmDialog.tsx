import { LoaderCircle } from 'lucide-react';
import { AlertDialog } from 'radix-ui';
import { useRef } from 'react';
import type { ReactNode } from 'react';
import { Button } from '../Button/Button';
import { RequiredLegend } from '../RequiredLegend/RequiredLegend';
import styles from './ConfirmDialog.module.css';

interface DialogBase {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  children: ReactNode;
  /** Names the safe choice, which has the focus: "Cancel", or "Keep editing" for unsaved changes. */
  cancelLabel?: string;
}

interface Asking extends DialogBase {
  acknowledgeOnly?: false;
  confirmLabel: string;
  /** Runs the action. The dialog stays open until you close it, so it can show `busy` meanwhile. */
  onConfirm: () => void;
  tone?: 'danger';
  busy?: boolean;
}

interface Informing extends DialogBase {
  acknowledgeOnly: true;
  confirmLabel?: never;
  onConfirm?: never;
  tone?: never;
  busy?: never;
}

export type ConfirmDialogProps = Asking | Informing;

/** The safe choice, Cancel, has the focus, so Enter never deletes anything by accident. */
export function ConfirmDialog(props: ConfirmDialogProps) {
  const { open, onOpenChange, title, children, acknowledgeOnly = false } = props;
  const cancelLabel = props.cancelLabel ?? (acknowledgeOnly ? 'Close' : 'Cancel');
  // Radix returns the focus only to its own trigger; this dialog is opened by other code.
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
          <RequiredLegend />
          <AlertDialog.Description asChild>
            <div className={styles.text}>{children}</div>
          </AlertDialog.Description>
          <div className={styles.actions}>
            <AlertDialog.Cancel asChild>
              <Button>{cancelLabel}</Button>
            </AlertDialog.Cancel>
            {!props.acknowledgeOnly && <ConfirmButton {...props} />}
          </div>
        </AlertDialog.Content>
      </AlertDialog.Portal>
    </AlertDialog.Root>
  );
}

function ConfirmButton({ confirmLabel, onConfirm, tone, busy = false }: Asking) {
  // A plain button, not AlertDialog.Action: that one would close the dialog at once.
  return (
    <Button
      variant={tone === 'danger' ? 'danger' : 'primary'}
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
  );
}

function focusedElement(): HTMLElement | null {
  return document.activeElement instanceof HTMLElement ? document.activeElement : null;
}

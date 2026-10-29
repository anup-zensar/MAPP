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
  /** The question: "Delete sample folder 'Ideas'?" */
  title: string;
  /** What will happen: "It goes to the recycle bin. You can restore it." */
  children: ReactNode;
  /** Names the safe choice, which has the focus: "Cancel", or "Keep editing" for unsaved changes. */
  cancelLabel?: string;
}

interface Asking extends DialogBase {
  acknowledgeOnly?: false;
  /** Names the action, never "OK": "Delete sample folder". */
  confirmLabel: string;
  /** Runs the action. The dialog stays open until you close it, so it can show `busy` meanwhile. */
  onConfirm: () => void;
  /** danger: the confirm button is Shell red, for actions that remove something; otherwise yellow. */
  tone?: 'danger';
  /** The action is running: a second click on the confirm button is ignored. */
  busy?: boolean;
}

/** Only informs ("Can't delete 'Q4'"): one Close button, nothing to confirm (038 FR-044). */
interface Informing extends DialogBase {
  acknowledgeOnly: true;
  confirmLabel?: never;
  onConfirm?: never;
  tone?: never;
  busy?: never;
}

export type ConfirmDialogProps = Asking | Informing;

/**
 * Asks before an action that cannot simply be taken back (design system §6.5). The safe choice,
 * Cancel, has the focus, so pressing Enter never deletes anything by accident.
 */
export function ConfirmDialog(props: ConfirmDialogProps) {
  const { open, onOpenChange, title, children, acknowledgeOnly = false } = props;
  const cancelLabel = props.cancelLabel ?? (acknowledgeOnly ? 'Close' : 'Cancel');
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

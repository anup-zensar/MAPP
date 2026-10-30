import { LoaderCircle } from 'lucide-react';
import { Dialog } from 'radix-ui';
import { useRef } from 'react';
import type { FormEvent, ReactNode } from 'react';
import { Button } from '../Button/Button';
import { RequiredLegend } from '../RequiredLegend/RequiredLegend';
import styles from './FormDialog.module.css';

export interface FormDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  description?: string;
  children: ReactNode;
  submitLabel: string;
  /** Runs the action; the dialog stays open until you close it, so it can show `busy`. */
  onSubmit: () => void;
  busy?: boolean;
  cancelLabel?: string;
  /** Only the submit button: a dialog that only tells something ("Close"). */
  hideCancel?: boolean;
}

// The first field has the focus, Enter submits and the focus goes back where it was on close.
// ConfirmDialog stays for questions whose text is the content.
export function FormDialog({
  open,
  onOpenChange,
  title,
  description,
  children,
  submitLabel,
  onSubmit,
  busy = false,
  cancelLabel = 'Cancel',
  hideCancel = false,
}: FormDialogProps) {
  const focusBeforeOpen = useRef<HTMLElement | null>(null);

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!busy) {
      onSubmit();
    }
  }

  return (
    <Dialog.Root open={open} onOpenChange={onOpenChange}>
      <Dialog.Portal>
        <Dialog.Overlay className={styles.overlay} />
        <Dialog.Content
          className={styles.dialog}
          {...(description ? {} : { 'aria-describedby': undefined })}
          onOpenAutoFocus={(event) => {
            focusBeforeOpen.current =
              document.activeElement instanceof HTMLElement ? document.activeElement : null;
            // The first field, or the submit button when there is none.
            const content = event.currentTarget;
            if (!(content instanceof HTMLElement)) {
              return;
            }
            const first = content.querySelector<HTMLElement>(
              'input:not([disabled]):not([type="hidden"]), select:not([disabled]), textarea:not([disabled])',
            );
            const target = first ?? content.querySelector<HTMLElement>('button[type="submit"]');
            if (target) {
              event.preventDefault();
              target.focus();
            }
          }}
          onCloseAutoFocus={(event) => {
            event.preventDefault();
            focusBeforeOpen.current?.focus();
          }}
        >
          <Dialog.Title className={styles.title}>{title}</Dialog.Title>
          {description && (
            <Dialog.Description className={styles.description}>{description}</Dialog.Description>
          )}
          <form noValidate aria-busy={busy} onSubmit={submit}>
            <div className={styles.form}>
              <RequiredLegend />
              {children}
            </div>
            <div className={styles.actions}>
              {!hideCancel && (
                <Dialog.Close asChild>
                  <Button>{cancelLabel}</Button>
                </Dialog.Close>
              )}
              <Button
                type="submit"
                variant="primary"
                icon={busy ? <LoaderCircle className={styles.spinner} /> : undefined}
                aria-busy={busy || undefined}
              >
                {submitLabel}
              </Button>
            </div>
          </form>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}

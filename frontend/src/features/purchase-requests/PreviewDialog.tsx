import { LoaderCircle } from 'lucide-react';
import { AlertDialog } from 'radix-ui';
import type { ReactNode } from 'react';
import { Button } from '@/ui/Button/Button';
import styles from './PreviewDialog.module.css';

export interface PreviewDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  children: ReactNode;
  confirm?: { label: string; onConfirm: () => void; busy: boolean };
}

/** A ConfirmDialog whose action shows only once the server's preview allows it (032 FR-041). */
export function PreviewDialog({
  open,
  onOpenChange,
  title,
  children,
  confirm,
}: PreviewDialogProps) {
  return (
    <AlertDialog.Root open={open} onOpenChange={onOpenChange}>
      <AlertDialog.Portal>
        <AlertDialog.Overlay className={styles.overlay} />
        <AlertDialog.Content className={styles.dialog}>
          <AlertDialog.Title className={styles.title}>{title}</AlertDialog.Title>
          <AlertDialog.Description asChild>
            <div className={styles.text}>{children}</div>
          </AlertDialog.Description>
          <div className={styles.actions}>
            <AlertDialog.Cancel asChild>
              <Button>Cancel</Button>
            </AlertDialog.Cancel>
            {confirm && (
              <Button
                variant="primary"
                icon={confirm.busy ? <LoaderCircle className={styles.spinner} /> : undefined}
                aria-busy={confirm.busy || undefined}
                onClick={() => !confirm.busy && confirm.onConfirm()}
              >
                {confirm.label}
              </Button>
            )}
          </div>
        </AlertDialog.Content>
      </AlertDialog.Portal>
    </AlertDialog.Root>
  );
}

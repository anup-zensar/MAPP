import { LoaderCircle } from 'lucide-react';
import { AlertDialog } from 'radix-ui';
import { useRef, useState } from 'react';
import type { QueryKey } from '@tanstack/react-query';
import { ApiError } from '@/api/errors';
import type { components } from '@/api/schema';
import { Button } from '@/components';
import { useUndoableDelete } from '@/lib/useUndoableDelete';
import { SETTING_NOUNS } from './settingsNouns';
import type { SettingKind } from './settingsNouns';
import { useFailureToast } from './useSettingStatus';
import styles from './SettingDeleteDialog.module.css';

type DeleteResult = components['schemas']['DeleteResult'];
type Usage = components['schemas']['Usage'];

export interface SettingDeleteDialogProps {
  kind: SettingKind;
  name: string;
  usage: Usage;
  active: boolean;
  /** Where it goes: "It moves to the recycle bin; you can restore it." unless given. */
  deleteText?: string;
  /** DELETE the record; answers its recycle-bin entry for Undo. */
  remove: () => Promise<DeleteResult>;
  /** Sets it inactive (with its own message); answers whether that worked. */
  setInactive: () => Promise<boolean>;
  /** The lists to refresh after Undo. */
  invalidate: QueryKey[];
  /** After the record moved to the recycle bin. */
  onDeleted?: () => void;
  onClose: () => void;
}

/**
 * Delete of a Settings record (FR-003, FR-004): an unused record moves to the recycle bin with Undo;
 * a record in use can't be deleted — the dialog says where it is used and offers Set inactive (or,
 * when it is inactive already, only Close). A delete the API refuses as in use switches the dialog
 * to that state with the API's message.
 */
export function SettingDeleteDialog({
  kind,
  name,
  usage,
  active,
  deleteText = 'It moves to the recycle bin; you can restore it.',
  remove,
  setInactive,
  invalidate,
  onDeleted,
  onClose,
}: SettingDeleteDialogProps) {
  const nouns = SETTING_NOUNS[kind];
  // The API's message when it found the record in use after all.
  const [refusal, setRefusal] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const afterDelete = useUndoableDelete({ invalidate });
  const showFailure = useFailureToast();
  const focusBeforeOpen = useRef<HTMLElement | null>(null);
  const inUse = usage.inUse || refusal !== null;

  async function removeAndTell() {
    try {
      afterDelete(await remove());
      onDeleted?.();
      onClose();
    } catch (error) {
      if (error instanceof ApiError && error.kind === 'inUse') {
        setRefusal(error.message);
        return;
      }
      onClose();
      showFailure(error, () => void removeAndTell());
    }
  }

  async function run(action: () => Promise<unknown>) {
    if (busy) {
      return;
    }
    setBusy(true);
    try {
      await action();
    } finally {
      setBusy(false);
    }
  }

  let title: string;
  let text: string;
  let confirm: { label: string; danger: boolean; onClick: () => void } | null;
  if (!inUse) {
    title = `Delete ${nouns.noun} '${name}'?`;
    text = deleteText;
    confirm = {
      label: `Delete ${nouns.noun}`,
      danger: true,
      onClick: () => void run(removeAndTell),
    };
  } else if (active) {
    title = `'${name}' is in use`;
    text = `${refusal ?? `${usage.summary} — set it inactive instead.`} ${nouns.inactiveEffect}`;
    confirm = {
      label: 'Set inactive',
      danger: false,
      onClick: () =>
        void run(async () => {
          await setInactive();
          onClose();
        }),
    };
  } else {
    title = `'${name}' is in use`;
    text = `${usage.summary}, so it can't be deleted. It's already inactive.`;
    confirm = null;
  }

  return (
    <AlertDialog.Root open onOpenChange={(open) => !open && onClose()}>
      <AlertDialog.Portal>
        <AlertDialog.Overlay className={styles.overlay} />
        <AlertDialog.Content
          className={styles.dialog}
          onOpenAutoFocus={() => {
            focusBeforeOpen.current =
              document.activeElement instanceof HTMLElement ? document.activeElement : null;
          }}
          onCloseAutoFocus={(event) => {
            event.preventDefault();
            focusBeforeOpen.current?.focus();
          }}
        >
          <AlertDialog.Title className={styles.title}>{title}</AlertDialog.Title>
          <AlertDialog.Description className={styles.text}>{text}</AlertDialog.Description>
          <div className={styles.actions}>
            {/* The safe choice has the focus. */}
            <AlertDialog.Cancel asChild>
              <Button>{confirm ? 'Cancel' : 'Close'}</Button>
            </AlertDialog.Cancel>
            {confirm && (
              <Button
                variant="primary"
                className={confirm.danger ? styles.danger : undefined}
                icon={busy ? <LoaderCircle className={styles.spinner} /> : undefined}
                aria-busy={busy || undefined}
                onClick={confirm.onClick}
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

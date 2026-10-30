import { useQueryClient } from '@tanstack/react-query';
import type { QueryClient } from '@tanstack/react-query';
import { useCallback, useEffect, useState } from 'react';
import type { ReactElement } from 'react';
import { useBlocker } from 'react-router';
import { markUnsaved } from '@/session/lost-changes';
import { SESSION_KEY } from '@/session/useSession';
import type { CurrentUser } from '@/session/useSession';
import { ConfirmDialog } from './ConfirmDialog/ConfirmDialog';

export interface UnsavedChanges {
  dialog: ReactElement;
  /** Closes at once when nothing was typed, else only after "Discard changes" (§6.2). */
  confirmDiscard: (close: () => void) => void;
}

/** Asks "Discard changes?" before typed changes are lost; a lost session names the form. */
export function useUnsavedChanges(isDirty: boolean, title?: string): UnsavedChanges {
  const queryClient = useQueryClient();
  const [author] = useState(() => signedInPersonId(queryClient));
  // A function kept in state is set through an updater, hence "() => close" below.
  const [askedToClose, setAskedToClose] = useState<(() => void) | null>(null);

  // The session ended or another person signed in: the changes cannot be saved, so leave at once.
  const shouldBlock = useCallback(
    () => isDirty && signedInPersonId(queryClient) === author,
    [isDirty, queryClient, author],
  );
  const blocker = useBlocker(shouldBlock);

  useEffect(
    () => (isDirty ? markUnsaved(() => title ?? pageTitle()) : undefined),
    [isDirty, title],
  );

  // The browser shows its own question; it does not let pages choose the words.
  useEffect(() => {
    if (!isDirty) {
      return;
    }
    const askBrowser = (event: BeforeUnloadEvent) => event.preventDefault();
    window.addEventListener('beforeunload', askBrowser);
    return () => window.removeEventListener('beforeunload', askBrowser);
  }, [isDirty]);

  function keepEditing() {
    if (blocker.state === 'blocked') {
      blocker.reset();
    }
    setAskedToClose(null);
  }

  function discardChanges() {
    if (blocker.state === 'blocked') {
      blocker.proceed();
    }
    const close = askedToClose;
    setAskedToClose(null);
    close?.();
  }

  function confirmDiscard(close: () => void) {
    if (isDirty) {
      setAskedToClose(() => close);
    } else {
      close();
    }
  }

  const dialog = (
    <ConfirmDialog
      open={blocker.state === 'blocked' || askedToClose !== null}
      onOpenChange={(open) => {
        if (!open) {
          keepEditing();
        }
      }}
      title="Discard changes?"
      cancelLabel="Keep editing"
      confirmLabel="Discard changes"
      tone="danger"
      onConfirm={discardChanges}
    >
      Your changes haven't been saved.
    </ConfirmDialog>
  );

  return { dialog, confirmDiscard };
}

function pageTitle(): string {
  return document.title.replace(/ · SP&A One$/, '');
}

function signedInPersonId(queryClient: QueryClient): string | null {
  return queryClient.getQueryData<CurrentUser | null>(SESSION_KEY)?.id ?? null;
}

import { useQueryClient } from '@tanstack/react-query';
import type { QueryClient } from '@tanstack/react-query';
import { useCallback, useEffect, useState } from 'react';
import type { ReactElement } from 'react';
import { useBlocker } from 'react-router';
import { markUnsaved } from '@/session/lost-changes';
import { SESSION_KEY } from '@/session/useSession';
import type { CurrentUser } from '@/session/useSession';
import { ConfirmDialog } from './ConfirmDialog/ConfirmDialog';

/** What useUnsavedChanges gives a form. */
export interface UnsavedChanges {
  /** The "Discard changes?" question. Render it once, next to the form. */
  dialog: ReactElement;
  /**
   * Closes the form with `close` — on Cancel, Close or Escape — at once when nothing was typed,
   * and only after "Discard changes" when something was (design system §6.2).
   */
  confirmDiscard: (close: () => void) => void;
}

/**
 * Protects what someone typed and has not saved yet (constitution III, design system §6.4, plan
 * "Shared hooks"). While `isDirty`:
 * - leaving the page — a link, Back or Forward — asks "Discard changes?" with Keep editing
 *   (focused) and Discard changes;
 * - a reload or closing the tab makes the browser ask its own question;
 * - confirmDiscard asks the same question before the form closes.
 *
 * - a lost session names the form on the sign-in page: `title`, else the page's title (038 FR-031).
 *
 *   const { dialog, confirmDiscard } = useUnsavedChanges(name !== folder.name);
 */
export function useUnsavedChanges(isDirty: boolean, title?: string): UnsavedChanges {
  const queryClient = useQueryClient();
  // The person who opened the form: the typed changes are theirs.
  const [author] = useState(() => signedInPersonId(queryClient));
  // A Cancel, Close or Escape waiting for the answer. (A function kept in state is set through
  // an updater function, hence the "() => close" below.)
  const [askedToClose, setAskedToClose] = useState<(() => void) | null>(null);

  // Leaving waits for an answer while there are typed changes — unless the session has ended or
  // another person signed in in another tab. The changes can then no longer be saved in the
  // author's name, so the sign-in page or Home opens at once (plan "Session").
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
        // Escape and Keep editing close the question without leaving.
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

/** "New plan" from the browser tab's "New plan · SP&A One" (usePageTitle). */
function pageTitle(): string {
  return document.title.replace(/ · SP&A One$/, '');
}

/** Who is signed in right now, from the session in TanStack Query's cache; null for nobody. */
function signedInPersonId(queryClient: QueryClient): string | null {
  return queryClient.getQueryData<CurrentUser | null>(SESSION_KEY)?.id ?? null;
}

import { createContext, useContext } from 'react';

/** The one button a message can have, such as Undo. */
export interface ToastAction {
  label: string;
  /**
   * What screen readers announce instead of the button, naming another way to do the same
   * ("Restore it from the recycle bin"), because the message may close before they reach it.
   */
  altText: string;
  onClick: () => void;
}

export interface ToastOptions {
  /** Names the record: "Sample folder 'Ideas' created". */
  message: string;
  action?: ToastAction;
  /** How long the message stays: 4 seconds, or 10 seconds when it has an action (FR-026). */
  durationMs?: number;
}

export interface Toaster {
  /** Shows a message at the bottom of the screen. Returns a function that closes it early. */
  show(toast: ToastOptions): () => void;
}

/** Shared by ToastProvider (which provides it) and useToast (which reads it). */
export const ToastContext = createContext<Toaster | null>(null);

/** Shows short messages after an action, optionally with Undo (design system §5). */
export function useToast(): Toaster {
  const toaster = useContext(ToastContext);
  if (!toaster) {
    throw new Error('useToast needs the ToastProvider around the app (App.tsx).');
  }
  return toaster;
}

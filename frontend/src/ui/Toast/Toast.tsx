import { Toast } from 'radix-ui';
import { useCallback, useMemo, useRef, useState } from 'react';
import type { ReactNode } from 'react';
import { ToastContext } from './useToast';
import type { Toaster, ToastOptions } from './useToast';
import styles from './Toast.module.css';

/** A message on screen; the id tells messages apart. */
interface ShownToast extends ToastOptions {
  id: number;
}

const DEFAULT_DURATION_MS = 4_000;
/** Long enough to choose Undo (FR-026). */
const WITH_ACTION_DURATION_MS = 10_000;

/**
 * Shows the messages of useToast() at the bottom centre of the screen (design system §5), with
 * Radix Toast: a message is announced to screen readers at once, waits while the pointer rests on
 * it or the keyboard focus is in it, and F8 moves the focus to it.
 */
export function ToastProvider({ children }: { children: ReactNode }) {
  const [toasts, setToasts] = useState<ShownToast[]>([]);
  const nextId = useRef(1);

  const close = useCallback((id: number) => {
    setToasts((current) => current.filter((toast) => toast.id !== id));
  }, []);

  const toaster = useMemo<Toaster>(
    () => ({
      show(toast) {
        const id = nextId.current++;
        setToasts((current) => [...current, { ...toast, id }]);
        return () => close(id);
      },
    }),
    [close],
  );

  return (
    <ToastContext value={toaster}>
      <Toast.Provider>
        {children}
        {toasts.map((toast) => (
          <Toast.Root
            key={toast.id}
            className={styles.toast}
            // Foreground: screen readers announce the message at once.
            type="foreground"
            duration={
              toast.durationMs ?? (toast.action ? WITH_ACTION_DURATION_MS : DEFAULT_DURATION_MS)
            }
            onOpenChange={(open) => {
              if (!open) {
                close(toast.id);
              }
            }}
          >
            <Toast.Description className={styles.message}>
              {/* Radix announces the message and the action's alt text back to back; the trailing
                  space keeps them apart ("…recycle bin Restore it…", not "…binRestore it…"). The
                  dash only shows: "<message> — View" (038 FR-049). */}
              {`${toast.message} `}
              {toast.action && <span aria-hidden="true">— </span>}
            </Toast.Description>
            {toast.action && (
              <Toast.Action
                className={styles.action}
                altText={toast.action.altText}
                onClick={toast.action.onClick}
              >
                {toast.action.label}
              </Toast.Action>
            )}
          </Toast.Root>
        ))}
        <Toast.Viewport className={styles.viewport} label="Messages ({hotkey})" />
      </Toast.Provider>
    </ToastContext>
  );
}

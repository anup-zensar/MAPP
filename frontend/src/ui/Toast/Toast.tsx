import { Toast } from 'radix-ui';
import { useCallback, useMemo, useRef, useState } from 'react';
import type { ReactNode } from 'react';
import { ToastContext } from './useToast';
import type { Toaster, ToastOptions } from './useToast';
import styles from './Toast.module.css';

interface ShownToast extends ToastOptions {
  id: number;
}

const DEFAULT_DURATION_MS = 4_000;
/** Long enough to choose Undo (FR-026). */
const WITH_ACTION_DURATION_MS = 10_000;

/** Radix Toast: announced at once, waits under the pointer or focus, and F8 moves focus to it. */
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
              {/* The space keeps the message apart from the action's alt text when read out. */}
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

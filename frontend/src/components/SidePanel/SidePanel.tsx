import { X } from 'lucide-react';
import { Dialog } from 'radix-ui';
import { useRef } from 'react';
import type { ReactNode } from 'react';
import { Button } from '../Button/Button';
import styles from './SidePanel.module.css';

export interface SidePanelProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** Names the panel: "New sample folder", "Edit sample folder". */
  title: string;
  description?: string;
  /** The form or details. It scrolls when it is long. */
  children: ReactNode;
  /** The buttons that stay at the bottom: Save (primary) and Cancel. */
  footer: ReactNode;
}

/**
 * A panel that slides in from the right to create, view or edit a short record (design system
 * §5). The keyboard stays inside it while it is open; Escape, the Close button and a click
 * beside it close it.
 */
export function SidePanel({
  open,
  onOpenChange,
  title,
  description,
  children,
  footer,
}: SidePanelProps) {
  // Radix gives the focus back only to its own trigger button. This panel is opened by other
  // code (a page button, a menu item), so it remembers where the focus was and returns it there.
  const focusBeforeOpen = useRef<HTMLElement | null>(null);

  return (
    <Dialog.Root open={open} onOpenChange={onOpenChange}>
      <Dialog.Portal>
        <Dialog.Overlay className={styles.overlay} />
        <Dialog.Content
          className={styles.panel}
          // Without a description, tell Radix there is none, so nothing points to a missing text.
          {...(description ? {} : { 'aria-describedby': undefined })}
          onOpenAutoFocus={() => {
            focusBeforeOpen.current = focusedElement();
          }}
          onCloseAutoFocus={(event) => {
            event.preventDefault();
            focusBeforeOpen.current?.focus();
          }}
        >
          <div className={styles.header}>
            <div>
              <Dialog.Title className={styles.title}>{title}</Dialog.Title>
              {description && (
                <Dialog.Description className={styles.description}>
                  {description}
                </Dialog.Description>
              )}
            </div>
            {/* No tooltip: inside a dialog it would swallow the first Escape (research R14). */}
            <Dialog.Close asChild>
              <Button variant="ghost" icon={<X />} aria-label="Close" />
            </Dialog.Close>
          </div>
          <div className={styles.body}>{children}</div>
          <div className={styles.footer}>{footer}</div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}

function focusedElement(): HTMLElement | null {
  return document.activeElement instanceof HTMLElement ? document.activeElement : null;
}

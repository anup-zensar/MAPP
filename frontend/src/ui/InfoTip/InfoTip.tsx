import { Info } from 'lucide-react';
import { Popover } from 'radix-ui';
import { useRef, useState } from 'react';
import type { PointerEvent } from 'react';
import styles from './InfoTip.module.css';

export interface InfoTipProps {
  /** One or two plain sentences that explain the term (from the glossary). */
  text: string;
  /** The term it explains; the button is named "About <label>". */
  label: string;
}

/** How long a mouse that leaves the ⓘ has to reach the bubble before it closes. */
const CLOSE_DELAY_MS = 150;

/**
 * A muted ⓘ next to a label that explains it in plain words (design system §5). It opens on hover,
 * keyboard focus, click and tap, so it works with a mouse, a keyboard and a touch screen. It
 * closes on Escape, when the mouse leaves, and — through Radix — when the focus moves to another
 * control or someone clicks outside it.
 *
 * A Radix Popover rather than a tooltip, because a tooltip cannot be opened by tapping.
 */
export function InfoTip({ text, label }: InfoTipProps) {
  const [open, setOpen] = useState(false);
  const closeTimer = useRef(0);
  const name = `About ${label}`;

  function show() {
    window.clearTimeout(closeTimer.current);
    setOpen(true);
  }

  // Only a mouse hovers: a finger also "enters" and "leaves", but that is a tap.
  function showOnHover(event: PointerEvent) {
    if (event.pointerType === 'mouse') {
      show();
    }
  }

  // The short delay lets the mouse move onto the bubble without it closing (WCAG 1.4.13).
  function hideAfterHover(event: PointerEvent) {
    if (event.pointerType === 'mouse') {
      closeTimer.current = window.setTimeout(() => setOpen(false), CLOSE_DELAY_MS);
    }
  }

  return (
    <Popover.Root open={open} onOpenChange={setOpen}>
      <Popover.Trigger asChild>
        <button
          type="button"
          className={styles.button}
          aria-label={name}
          onPointerEnter={showOnHover}
          onPointerLeave={hideAfterHover}
          onFocus={show}
          onClick={(event) => {
            // Stops Radix from toggling: a click on the ⓘ only ever opens the tip.
            event.preventDefault();
            show();
          }}
        >
          <Info />
        </button>
      </Popover.Trigger>
      <Popover.Portal>
        <Popover.Content
          className={styles.bubble}
          side="top"
          sideOffset={6}
          aria-label={name}
          // The focus stays on the ⓘ while the tip is open, and is not moved when it closes.
          onOpenAutoFocus={(event) => event.preventDefault()}
          onCloseAutoFocus={(event) => event.preventDefault()}
          onPointerEnter={showOnHover}
          onPointerLeave={hideAfterHover}
        >
          {text}
          <Popover.Arrow className={styles.arrow} />
        </Popover.Content>
      </Popover.Portal>
    </Popover.Root>
  );
}

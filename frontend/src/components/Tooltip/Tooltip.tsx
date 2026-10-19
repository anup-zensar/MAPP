import { Tooltip as RadixTooltip } from 'radix-ui';
import type { ReactElement, ReactNode } from 'react';
import styles from './Tooltip.module.css';

export interface TooltipProps {
  /** The short text in the bubble. */
  content: ReactNode;
  side?: 'top' | 'right' | 'bottom' | 'left';
  /** The one control the tooltip is about: a button or a link (it must accept a ref). */
  children: ReactElement;
}

/**
 * A small dark bubble shown while the pointer rests on a control or the control has keyboard focus
 * (design system §5). It only describes: the control needs its own name (aria-label) as well.
 *
 * Never put a tooltip inside a menu, nor on a close button inside a dialog, side panel or drawer:
 * there a Radix tooltip swallows the first Escape (research R14). Needs App's Tooltip.Provider.
 */
export function Tooltip({ content, side = 'top', children }: TooltipProps) {
  return (
    <RadixTooltip.Root>
      <RadixTooltip.Trigger asChild>{children}</RadixTooltip.Trigger>
      <RadixTooltip.Portal>
        <RadixTooltip.Content className={styles.bubble} side={side} sideOffset={6}>
          {content}
          <RadixTooltip.Arrow className={styles.arrow} />
        </RadixTooltip.Content>
      </RadixTooltip.Portal>
    </RadixTooltip.Root>
  );
}

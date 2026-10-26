import { ArrowDown, ArrowUp } from 'lucide-react';
import { Button } from '@/ui/Button/Button';
import { Tooltip } from '@/ui/Tooltip/Tooltip';
import styles from './ShellAttributes.module.css';

export interface OrderButtonProps {
  id: string;
  label: string;
  direction: 'up' | 'down';
  /** Why it can't move now; the button stays focusable and its tip says why (SC-009). */
  disabledReason?: string;
  onMove: () => void;
}

/** One move button. At an edge or while filtering it is aria-disabled, never removed. */
export function OrderButton({ id, label, direction, disabledReason, onMove }: OrderButtonProps) {
  return (
    <Tooltip content={disabledReason ?? label}>
      <Button
        id={id}
        variant="tertiary"
        size="sm"
        icon={direction === 'up' ? <ArrowUp /> : <ArrowDown />}
        aria-label={label}
        aria-disabled={disabledReason ? true : undefined}
        className={disabledReason ? styles.unavailable : undefined}
        onClick={() => {
          if (!disabledReason) {
            onMove();
          }
        }}
      />
    </Tooltip>
  );
}

import { Star } from 'lucide-react';
import { Button } from '@/ui/Button/Button';
import { Tooltip } from '@/ui/Tooltip/Tooltip';
import { cx } from '@/lib/cx';
import styles from './PinButton.module.css';

export interface PinButtonProps {
  /** The plan's name, for the button's accessible name and tooltip. */
  planName: string;
  pinned: boolean;
  onToggle: () => void;
}

/**
 * The star that pins a plan to the top of the Plans list (018 FR-001, FR-003): an ink outline, and
 * filled brand yellow while the plan is pinned — the filled star stays visible on every pinned
 * card and row. Icon-only, so the accessible name carries the action and the plan, and a tooltip
 * says the same words (design system §5).
 */
export function PinButton({ planName, pinned, onToggle }: PinButtonProps) {
  const name = pinned ? `Unpin plan '${planName}'` : `Pin plan '${planName}'`;
  return (
    <Tooltip content={name}>
      <Button
        variant="tertiary"
        size="sm"
        icon={<Star aria-hidden="true" />}
        aria-label={name}
        className={cx(styles.star, pinned && styles.pinned)}
        onClick={onToggle}
      />
    </Tooltip>
  );
}

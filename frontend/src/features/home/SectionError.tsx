import { CircleAlert, RotateCw } from 'lucide-react';
import { Button } from '@/components';
import { SECTION_FAILED } from './homeText';
import styles from './SectionError.module.css';

export interface SectionErrorProps {
  /** Asks the card's data again. */
  onRetry: () => void;
}

/**
 * Shown inside a card whose data could not be loaded (FR-022): "This section couldn't load" with
 * Try again. The other cards of Home keep showing.
 */
export function SectionError({ onRetry }: SectionErrorProps) {
  return (
    <div className={styles.error}>
      <p role="alert" className={styles.message}>
        <CircleAlert aria-hidden="true" />
        {SECTION_FAILED}
      </p>
      <Button size="sm" icon={<RotateCw />} onClick={onRetry}>
        Try again
      </Button>
    </div>
  );
}

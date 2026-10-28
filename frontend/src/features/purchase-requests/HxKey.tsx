import { Copy } from 'lucide-react';
import { useId } from 'react';
import { Button } from '@/ui/Button/Button';
import { useToast } from '@/ui/Toast/useToast';
import { HX_LABEL } from './purchase-request-text';
import styles from './HxKey.module.css';

const COPIED = 'HX key copied';
const COPY_FAILED = "Couldn't copy the HX key — select it and copy it yourself.";

export interface HxKeyProps {
  value: string;
  /** In a table cell: the column header's ⓘ carries "Generated from the line-item ID". */
  compact?: boolean;
}

/** A PR line's HX key, read-only with Copy (032 FR-020): the requisitioner types it into GSAP. */
export function HxKey({ value, compact = false }: HxKeyProps) {
  const toast = useToast();
  const hintId = useId();

  async function handleCopy() {
    try {
      await navigator.clipboard.writeText(value);
      toast.show({ message: COPIED });
    } catch {
      toast.show({ message: COPY_FAILED });
    }
  }

  return (
    <span className={styles.hx}>
      <span className={styles.row}>
        <span className={styles.value} aria-describedby={compact ? undefined : hintId}>
          {value}
        </span>
        <Button
          size="sm"
          variant="tertiary"
          icon={<Copy />}
          aria-label={`Copy HX key ${value}`}
          onClick={() => void handleCopy()}
        />
      </span>
      {!compact && (
        <span id={hintId} className={styles.note}>
          {HX_LABEL}
        </span>
      )}
    </span>
  );
}

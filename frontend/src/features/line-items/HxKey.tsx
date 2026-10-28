import { Copy } from 'lucide-react';
import { useId } from 'react';
import { Button } from '@/ui/Button/Button';
import { useToast } from '@/ui/Toast/useToast';
import { copyHxLabel, HX_COPIED, HX_COPY, HX_COPY_FAILED, HX_HINT } from './swbs-text';
import styles from './SwbsCard.module.css';

/** The read-only HX key with a Copy action (030 FR-016): requisitioners type it into GSAP. */
export function HxKey({ value }: { value: string }) {
  const toast = useToast();
  const hintId = useId();

  async function handleCopy() {
    try {
      await navigator.clipboard.writeText(value);
      toast.show({ message: HX_COPIED });
    } catch {
      toast.show({ message: HX_COPY_FAILED });
    }
  }

  return (
    <span className={styles.hx}>
      <span className={styles.hxRow}>
        <span className={styles.hxValue} aria-describedby={hintId}>
          {value}
        </span>
        <Button
          size="sm"
          variant="tertiary"
          icon={<Copy />}
          aria-label={copyHxLabel(value)}
          onClick={() => void handleCopy()}
        >
          {HX_COPY}
        </Button>
      </span>
      <span id={hintId} className={styles.note}>
        {HX_HINT}
      </span>
    </span>
  );
}

import { Send } from 'lucide-react';
import { useId, useState } from 'react';
import { isShown } from '@/session/access';
import { Button } from '@/ui/Button/Button';
import { useHidesUnavailable } from '@/ui/ViewOnlyNote/view-only-context';
import type { PurchaseRequest } from './api';
import { GSAP_ORDER } from './gsap-order-text';
import { GsapOrderDialog } from './GsapOrderDialog';
import styles from './PurchaseRequestPage.module.css';

/** 033 FR-001, FR-010: offered, explained when it cannot run, hidden without finance.import. */
export function GsapOrderButton({ purchaseRequest }: { purchaseRequest: PurchaseRequest }) {
  const [open, setOpen] = useState(false);
  const reasonId = useId();
  const hidesUnavailable = useHidesUnavailable();
  const { gsapOrder } = purchaseRequest.access;
  if (!isShown(gsapOrder) || (hidesUnavailable && !gsapOrder.available)) {
    return null;
  }
  const unavailable = !gsapOrder.available;

  return (
    <span className={styles.withReason}>
      <Button
        icon={<Send />}
        disabled={unavailable}
        aria-describedby={unavailable ? reasonId : undefined}
        onClick={() => setOpen(true)}
      >
        {GSAP_ORDER}
      </Button>
      {unavailable && (
        <span id={reasonId} className={styles.reason}>
          {gsapOrder.reason}
        </span>
      )}
      {open && <GsapOrderDialog purchaseRequest={purchaseRequest} onClose={() => setOpen(false)} />}
    </span>
  );
}

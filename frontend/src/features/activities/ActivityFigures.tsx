import type { ReactNode } from 'react';
import { SummaryCard } from '@/ui/SummaryCard/SummaryCard';
import { amountText, moneyText, RateTip } from '@/features/line-items';
import { useFieldHelp } from '@/features/settings';
import type { Money } from '@/features/line-items';
import { toCents } from '@/format/decimal';
import type { Activity } from './api';
import styles from './ActivityPage.module.css';
import { lineItemsAttentionText, lineItemsText } from './activity-text';

/** Planned cost, unassigned planned cost and line items, with the display currency beside. */
export function ActivityFigures({ activity }: { activity: Activity }) {
  const { help } = useFieldHelp();
  const planned = activity.plannedCost;
  const { count } = activity.lineItems;
  const attention = activity.completeness.lineItemsNeedingAttention;

  return (
    <section className={styles.figures} aria-label="Summary">
      <SummaryCard
        label="Planned cost"
        tip={help('activity.plannedCost')}
        value={planned ? moneyText(planned) : '—'}
        detail={
          planned ? (
            <Beside primary={planned} secondary={activity.display.plannedCost} />
          ) : (
            'No planned cost yet'
          )
        }
      />
      <UnassignedCard
        unassigned={activity.unassignedPlannedCost}
        display={activity.display.unassignedPlannedCost}
      />
      <SummaryCard
        label="Line items"
        value={count}
        detail={
          count === 0
            ? 'No line items yet'
            : attention > 0
              ? lineItemsAttentionText(attention)
              : `${lineItemsText(count)}, all filled in`
        }
      />
    </section>
  );
}

function UnassignedCard({
  unassigned,
  display,
}: {
  unassigned: Money | null;
  display: Money | null;
}) {
  const tip = useFieldHelp().help('activity.unassignedPlannedCost');
  if (!unassigned) {
    return (
      <SummaryCard
        label="Unassigned planned cost"
        tip={tip}
        value="—"
        detail="No planned cost yet"
      />
    );
  }
  const cents = toCents(unassigned.amount);
  let detail: ReactNode;
  if (cents < 0n) {
    detail = (
      <span className={styles.problem}>
        Line items exceed the planned cost by {amountText(-cents, unassigned.currency)}
      </span>
    );
  } else if (cents === 0n) {
    detail = 'All of the planned cost is assigned';
  } else {
    detail = (
      <>
        <span>Not yet assigned to a line item</span>
        <Beside primary={unassigned} secondary={display} />
      </>
    );
  }
  return (
    <SummaryCard
      label="Unassigned planned cost"
      tip={tip}
      value={moneyText(unassigned)}
      tone={cents < 0n ? 'problem' : undefined}
      detail={detail}
    />
  );
}

/** "≈ 15,822.78 USD ⓘ": the display-currency amount under a card's value, when it differs. */
function Beside({ primary, secondary }: { primary: Money; secondary: Money | null }) {
  if (!secondary || secondary.currency === primary.currency) {
    return null;
  }
  return (
    <span className={styles.beside}>
      ≈ {moneyText(secondary)}
      <RateTip codes={[primary.currency, secondary.currency]} />
    </span>
  );
}

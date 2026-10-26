import type { ReactNode } from 'react';
import { SummaryCard } from '@/ui/SummaryCard/SummaryCard';
import { amountText, moneyText, RateTip } from '@/features/line-items';
import type { Money } from '@/features/line-items';
import { toCents } from '@/format/decimal';
import type { Activity } from './activitiesApi';
import styles from './ActivityPage.module.css';
import { lineItemsAttentionText, lineItemsText } from './activityText';

const UNASSIGNED_TIP =
  "The activity's planned cost minus its line items' planned costs, in the plan currency (Sys_Gen_Balance). Aim for 0.00.";

/**
 * The activity's money and progress facts under the header (FR-019, design system §2.4 B): planned
 * cost, unassigned planned cost (red words when line items exceed it) and its line items. Amounts
 * are in the plan currency, with the reader's display currency beside them when it differs (016
 * FR-013), in the one money format of this page ("120,000.00 USD").
 */
export function ActivityFigures({ activity }: { activity: Activity }) {
  const planned = activity.plannedCost;
  const { count } = activity.lineItems;
  const attention = activity.completeness.lineItemsNeedingAttention;

  return (
    <section className={styles.figures} aria-label="Summary">
      <SummaryCard
        label="Planned cost"
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
  if (!unassigned) {
    return (
      <SummaryCard
        label="Unassigned planned cost"
        tip={UNASSIGNED_TIP}
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
      tip={UNASSIGNED_TIP}
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

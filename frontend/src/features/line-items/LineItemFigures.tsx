import { SummaryCard } from '@/ui/SummaryCard/SummaryCard';
import { useFieldHelp } from '@/features/settings';
import { formatRate, toCents } from '@/format/decimal';
import { centsInput, formatMonth, moneyText } from './line-item-math';
import type { LineItem, UnassignedBalance } from './api';
import { conversionText, LABEL, NOT_SPREAD_YET, SPREAD_LABELS } from './line-item-text';
import styles from './LineItemPage.module.css';
import { RateTip } from './MoneyPair';

const UNASSIGNED_LABEL = "Activity's unassigned planned cost";

/** The summary cards: planned cost (other currencies and rates beside it), spread, unassigned. */
export function LineItemFigures({
  lineItem,
  balance,
}: {
  lineItem: LineItem;
  balance: UnassignedBalance;
}) {
  const { spread } = lineItem;
  const left = balance.balance;
  const negative = left !== null && toCents(left.amount) < 0n;
  const { help } = useFieldHelp();
  return (
    <section className={styles.figures} aria-label="Summary">
      <SummaryCard
        label={LABEL.plannedCost}
        tip={help('lineItem.plannedCost')}
        value={moneyText(lineItem.plannedCost)}
        detail={<CostBeside lineItem={lineItem} />}
      />
      <SummaryCard
        label={LABEL.spread}
        tip={help('lineItem.spread')}
        value={SPREAD_LABELS[spread.method]}
        detail={spreadDetail(spread)}
      />
      <SummaryCard
        label={UNASSIGNED_LABEL}
        tip={help('activity.unassignedPlannedCost')}
        value={left ? moneyText(left) : '—'}
        tone={negative ? 'problem' : undefined}
        detail={
          left === null
            ? 'The activity has no planned cost yet'
            : negative
              ? 'Line items exceed the planned cost'
              : toCents(left.amount) === 0n
                ? 'All of the planned cost is assigned'
                : 'Not yet assigned to a line item'
        }
      />
    </section>
  );
}

/** "= 46,000.00 GBP · ≈ 50,000.00 USD"; without a plan amount "= 50,000.00 USD at 0.92 EUR per USD". */
function CostBeside({ lineItem }: { lineItem: LineItem }) {
  const local = lineItem.plannedCost;
  const plan = lineItem.plannedCostPlan;
  const usd = lineItem.plannedCostUsd;
  if (local.currency === 'USD' && (!plan || plan.currency === 'USD')) {
    return null;
  }
  const lines: string[] = [];
  if (plan && plan.currency !== local.currency) {
    lines.push(`= ${moneyText(plan)}`);
  }
  if (usd.currency !== local.currency && (!plan || plan.currency !== usd.currency)) {
    lines.push(
      plan
        ? `≈ ${moneyText(usd)}`
        : conversionText(centsInput(toCents(usd.amount)), 'USD', [
            { rate: formatRate(lineItem.currency.ratePerUsd), code: lineItem.currency.code },
          ]),
    );
  }
  return (
    <span className={styles.beside}>
      {lines.join(' · ')}
      <RateTip codes={[local.currency, ...(plan ? [plan.currency] : [])]} />
    </span>
  );
}

/** "3 months: Mar 2026 – May 2026", counting the months that hold an amount. */
function spreadDetail(spread: LineItem['spread']): string {
  const months = spread.months.filter((month) => toCents(month.amount.amount) !== 0n);
  const first = months[0];
  const last = months.at(-1);
  if (spread.state === 'notSpread') {
    return NOT_SPREAD_YET;
  }
  if (!first || !last) {
    return 'Nothing in any month yet';
  }
  const count = months.length === 1 ? '1 month' : `${months.length} months`;
  return `${count}: ${formatMonth(first.month)}${months.length > 1 ? ` – ${formatMonth(last.month)}` : ''}`;
}

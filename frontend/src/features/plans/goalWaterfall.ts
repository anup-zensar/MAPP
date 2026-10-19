/**
 * The plan goal at every stage while the form is typed (FR-009, plan "Goal waterfall"): forwards
 * from INQ, partly backwards from MQL, backwards from Revenue — the same rounding as Go's
 * plans.planWaterfall (half up to whole numbers, revenue to the cent). What the API calculates is
 * what gets stored and shown afterwards.
 */
import {
  formatAmount,
  formatCents,
  formatPercent,
  formatWhole,
  parseDecimalInput,
  parseMoneyInput,
  toCents,
  toHundredths,
} from '@/lib/decimal';
import type { WaterfallModel } from '@/lib/waterfall';
import type { GoalStage } from './planText';

export type WaterfallStage = 'inq' | 'mql' | 'deals' | 'revenue';

/** One row of the goal table: the typed stage, a calculated one, or one that can't be worked out. */
export interface GoalRow {
  stage: WaterfallStage;
  /** "250,000", "2,500,000.00 USD"; null when there is nothing to show. */
  value: string | null;
  /** The typed stage is not calculated. */
  calculated: boolean;
  /** "MQL = INQ × 2.00 %" for a calculated stage. */
  formula?: string;
  /** Why the stage can't be worked out (a rate or the deal size is 0). */
  reason?: string;
}

export const REASON_INQ_TO_MQL = "The model's INQ→MQL rate is 0, so this can't be worked out.";
export const REASON_MQL_TO_DEAL = "The model's MQL→deal rate is 0, so this can't be worked out.";
export const REASON_DEAL_SIZE = "The model's average deal size is 0, so this can't be worked out.";
export const CHOOSE_A_MODEL = 'Choose a model to calculate';

const TEN_THOUSAND = 10_000n;

/** n × mul ÷ div, rounded half up (n, mul ≥ 0); null when div is 0. */
function ratio(n: bigint | null, mul: bigint, div: bigint): bigint | null {
  if (n === null || div <= 0n) {
    return null;
  }
  return (n * mul * 2n + div) / (2n * div);
}

/**
 * The four stages for a typed goal: the typed one as typed, the others worked out from the model.
 * Without a model the calculated stages say CHOOSE_A_MODEL (as their reason); without a goal
 * amount, or one that can't be read, every value is null.
 */
export function goalRows(
  stage: GoalStage,
  amount: string,
  model: WaterfallModel | null,
): GoalRow[] {
  const r1 = model ? (toHundredths(model.inqToMqlRate) ?? 0n) : 0n;
  const r2 = model ? (toHundredths(model.mqlToDealRate) ?? 0n) : 0n;
  const dealCents = model ? toCents(model.averageDealSize.amount) : 0n;
  const rate1 = model ? formatPercent(model.inqToMqlRate) : '';
  const rate2 = model ? formatPercent(model.mqlToDealRate) : '';
  const dealSize = model ? `${formatAmount(model.averageDealSize.amount)} USD` : '';

  let inq: bigint | null = null;
  let mql: bigint | null = null;
  let deals: bigint | null = null;
  let revenueCents: bigint | null;
  const reasons: Partial<Record<WaterfallStage, string>> = {};
  const formulas: Record<WaterfallStage, string> = {
    inq: `INQ = MQL ÷ ${rate1}`,
    mql: `MQL = INQ × ${rate1}`,
    deals: `Deals = MQL × ${rate2}`,
    revenue: `Revenue = deals × ${dealSize}`,
  };

  const text = amount.trim();
  if (stage === 'revenue') {
    const parsed = text ? parseMoneyInput(text) : null;
    revenueCents = parsed && !parsed.amount.startsWith('-') ? toCents(parsed.amount) : null;
    formulas.deals = `Deals = revenue ÷ ${dealSize}`;
    formulas.mql = `MQL = deals ÷ ${rate2}`;
    if (model && revenueCents !== null) {
      deals = ratio(revenueCents, 1n, dealCents);
      mql = ratio(deals, TEN_THOUSAND, r2);
      inq = ratio(mql, TEN_THOUSAND, r1);
      const reason =
        dealCents === 0n
          ? { deals: REASON_DEAL_SIZE, mql: REASON_DEAL_SIZE, inq: REASON_DEAL_SIZE }
          : r2 === 0n
            ? { mql: REASON_MQL_TO_DEAL, inq: REASON_MQL_TO_DEAL }
            : r1 === 0n
              ? { inq: REASON_INQ_TO_MQL }
              : {};
      Object.assign(reasons, reason);
    }
  } else {
    const whole = text ? parseDecimalInput(text, 0) : null;
    const typed = whole !== null && !whole.startsWith('-') ? BigInt(whole) : null;
    if (stage === 'inq') {
      inq = typed;
      if (model) {
        mql = ratio(inq, r1, TEN_THOUSAND);
        deals = ratio(mql, r2, TEN_THOUSAND);
      }
    } else {
      mql = typed;
      if (model) {
        inq = ratio(mql, TEN_THOUSAND, r1);
        if (mql !== null && r1 === 0n) {
          reasons.inq = REASON_INQ_TO_MQL;
        }
        deals = ratio(mql, r2, TEN_THOUSAND);
      }
    }
    revenueCents = deals === null ? null : deals * dealCents;
  }

  const values: Record<WaterfallStage, string | null> = {
    inq: inq === null ? null : formatWhole(inq),
    mql: mql === null ? null : formatWhole(mql),
    deals: deals === null ? null : formatWhole(deals),
    revenue: revenueCents === null ? null : `${formatAmount(formatCents(revenueCents))} USD`,
  };

  return (['inq', 'mql', 'deals', 'revenue'] as const).map((row) => {
    if (row === stage) {
      return { stage: row, value: values[row], calculated: false };
    }
    if (!model) {
      return { stage: row, value: null, calculated: true, reason: CHOOSE_A_MODEL };
    }
    return {
      stage: row,
      value: values[row],
      calculated: true,
      formula: formulas[row],
      reason: reasons[row],
    };
  });
}

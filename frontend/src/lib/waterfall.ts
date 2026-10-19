/**
 * The goal waterfall of a model (data-model 002 §2.1, FR-010, SC-006), the same formula as Go's
 * settings.Waterfall, for instant feedback on forms (002's worked example, 004's and 006's goals).
 * BigInt throughout; what the API calculates is what gets stored and shown afterwards.
 *
 *   MQL     = INQ × INQ→MQL rate      (half-up to a whole number)
 *   Deals   = MQL × MQL→deal rate     (half-up to a whole number)
 *   Revenue = Deals × average deal size
 */
import { formatCents, toCents, toHundredths } from './decimal';

/** The largest amount the API can hold: 999,999,999,999,999.99. */
const MAX_CENTS = 99_999_999_999_999_999n;

/** What the waterfall needs of a model (the API's Model has these fields). */
export interface WaterfallModel {
  /** A percentage with at most two decimals, "2.00". */
  inqToMqlRate: string;
  mqlToDealRate: string;
  /** In USD. */
  averageDealSize: { amount: string };
}

export interface WaterfallResult {
  mql: bigint;
  deals: bigint;
  /** In USD, as the API writes amounts ("2500000.00"). */
  revenue: string;
}

/**
 * The calculated stages for `inq` inquiries, or 'tooLarge' when the revenue would be beyond what
 * an amount can hold ("Too large to calculate — try fewer inquiries.").
 */
export function waterfall(inq: bigint, model: WaterfallModel): WaterfallResult | 'tooLarge' {
  const mql = ofWhole(inq, percent(model.inqToMqlRate));
  const deals = ofWhole(mql, percent(model.mqlToDealRate));
  const revenue = deals * toCents(model.averageDealSize.amount);
  if (revenue > MAX_CENTS || revenue < -MAX_CENTS) {
    return 'tooLarge';
  }
  return { mql, deals, revenue: formatCents(revenue) };
}

/** n × p %, rounded half-up (half away from zero below 0) to a whole number (Go's Percent.OfWhole). */
function ofWhole(n: bigint, hundredths: bigint): bigint {
  const product = n * hundredths;
  const magnitude = product < 0n ? -product : product;
  const rounded = (magnitude * 2n + 10_000n) / 20_000n;
  return product < 0n ? -rounded : rounded;
}

function percent(text: string): bigint {
  const hundredths = toHundredths(text);
  if (hundredths === null) {
    throw new Error(`Not a percentage with at most two decimals: "${text}"`);
  }
  return hundredths;
}

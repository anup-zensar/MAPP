import { InfoTip } from '@/ui/InfoTip/InfoTip';
import { cx } from '@/lib/cx';
import { useFormatters } from '@/lib/useFormatters';
import { moneyText } from './lineItemMath';
import type { Money } from './lineItemsApi';
import styles from './MoneyPair.module.css';
import { factsOf, RATES_UNKNOWN, rateText, useCurrentRates } from './rates';

export interface MoneyPairProps {
  /** The amount as typed: in the plan currency (activities) or its own currency. */
  primary: Money | null | undefined;
  /** The same amount in the reader's display currency, from the API; shown only when it differs. */
  secondary?: Money | null;
  /** Red, with the words given by the caller (a negative balance). */
  problem?: boolean;
}

/**
 * An amount in the plan currency with the reader's display currency beside it (016 FR-013, plan
 * "AmountPair"): "40,000.00 GBP ≈ 50,632.91 USD", the second one muted with an ⓘ that says which
 * rates were used (FR-009). When both currencies are the same only the first shows. Amounts use
 * the code form, the one money format of the activity and line-item pages (design system §4.4).
 */
export function MoneyPair({ primary, secondary, problem = false }: MoneyPairProps) {
  const both = Boolean(primary && secondary && secondary.currency !== primary.currency);
  return (
    <span className={styles.pair}>
      <span className={cx(styles.primary, problem && styles.problem)}>
        {primary ? moneyText(primary) : '—'}
      </span>
      {both && primary && secondary && (
        <span className={styles.secondary}>
          ≈ {moneyText(secondary)}
          <RateTip codes={[primary.currency, secondary.currency]} />
        </span>
      )}
    </span>
  );
}

/**
 * The ⓘ that names the rates behind a converted amount: "1 USD = 0.79 GBP · rate of 19 Oct 2026 ·
 * GSAP import" for each currency other than USD (016 FR-009).
 */
export function RateTip({ codes }: { codes: string[] }) {
  const { date } = useFormatters();
  const rates = useCurrentRates();
  const lines = codes
    .filter((code) => code !== 'USD')
    .map((code) => {
      const rate = rates.data?.items.find((item) => item.currency === code);
      return rate ? rateText(factsOf(rate), date) : null;
    });
  const text =
    lines.length > 0 && lines.every((line) => line !== null) ? lines.join('; ') : RATES_UNKNOWN;
  return <InfoTip label="the rates used" text={text} />;
}

import type { components } from '@/api/schema';
import { formatMoneyText } from '@/lib/decimal';
import { useFormatters } from '@/lib/useFormatters';
import styles from './AmountPair.module.css';
import { differentCurrency } from './planCurrency';

type Money = components['schemas']['Money'];

export interface AmountPairProps {
  /** The amount in the plan currency, as kept (016 FR-013); "—" when there is none. */
  primary: Money | null | undefined;
  /** The same amount in the reader's display currency; shown only when its currency differs. */
  secondary?: Money | null;
  /** Side by side ("600,000.00 GBP ≈ 759,493.67 USD") instead of one under the other. */
  inline?: boolean;
  /** Colours both lines red, for an amount that is a problem (over budget, negative). */
  problem?: boolean;
}

/**
 * An amount in the plan currency with the display-currency amount beside it (016 FR-013, FR-014;
 * plan "Shared parts", AmountPair). With one currency it is the usual "$12,500.00"; with two, both
 * use the code form ("600,000.00 GBP" and "≈ 759,493.67 USD", design system §4.4), so one screen
 * never mixes "$" with codes.
 *
 * Lane f1 keeps it in the plans feature until 016's shared `components/AmountPair` lands.
 */
export function AmountPair({
  primary,
  secondary,
  inline = false,
  problem = false,
}: AmountPairProps) {
  const { money } = useFormatters();
  const both = differentCurrency(primary, secondary);
  if (!primary || !both || !secondary) {
    return <span className={problem ? styles.problem : undefined}>{money(primary)}</span>;
  }
  return (
    <span className={`${inline ? styles.inline : styles.stack} ${problem ? styles.problem : ''}`}>
      <span>{formatMoneyText(primary)}</span>
      <span className={styles.secondary}>
        <span aria-hidden="true">≈ </span>
        <span className="visually-hidden">about </span>
        {formatMoneyText(secondary)}
      </span>
    </span>
  );
}

import styles from './LineItemForm.module.css';
import {
  EMPTY,
  FROM_FINANCE_FIELDS,
  FROM_FINANCE_NOTE,
  SECTION_FROM_FINANCE,
} from './lineItemText';

/**
 * "From finance (coming later)" (FR-025): the values the Finance module and GSAP will fill, each
 * "—", in a section closed by default.
 */
export function FromFinanceSection() {
  return (
    <details className={styles.fromFinance}>
      <summary>{SECTION_FROM_FINANCE}</summary>
      <p className={styles.note}>{FROM_FINANCE_NOTE}</p>
      <dl className={styles.facts}>
        {FROM_FINANCE_FIELDS.map((label) => (
          <div key={label}>
            <dt>{label}</dt>
            <dd>{EMPTY}</dd>
          </div>
        ))}
      </dl>
    </details>
  );
}

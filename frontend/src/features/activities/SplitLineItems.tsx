import { useId } from 'react';
import { moneyText } from '@/features/line-items';
import type { ActivitySplitOptions } from './split-api';
import styles from './SplitActivityDialog.module.css';

export interface SplitLineItemsProps {
  lineItems: ActivitySplitOptions['lineItems'];
  /** The plan currency the amounts are in. */
  code: string;
  /** The ticked line items' ids, in the order they were ticked. */
  value: string[];
  onChange: (ids: string[]) => void;
}

/** The line items to move, each with its plan-currency amount and its own one beside it. */
export function SplitLineItems({ lineItems, code, value, onChange }: SplitLineItemsProps) {
  const listId = useId();
  return (
    <fieldset className={styles.items} aria-describedby={`${listId}-hint`}>
      <legend className={styles.legend}>Line items to move</legend>
      <p id={`${listId}-hint`} className={styles.hint}>
        They keep their IDs, values and spreads. Amounts in {code}, the plan currency.
      </p>
      {lineItems.length === 0 ? (
        <p className={styles.hint}>This activity has no line items.</p>
      ) : (
        <ul className={styles.list}>
          {lineItems.map((item) => {
            const id = `${listId}-${item.id}`;
            return (
              <li key={item.id} className={styles.item}>
                <input
                  id={id}
                  type="checkbox"
                  checked={value.includes(item.id)}
                  onChange={(event) =>
                    onChange(
                      event.target.checked
                        ? [...value, item.id]
                        : value.filter((each) => each !== item.id),
                    )
                  }
                />
                <label htmlFor={id}>{item.name}</label>
                <span className={styles.amount}>
                  {moneyText(item.plannedCostPlan)}
                  {item.plannedCost.currency !== item.plannedCostPlan.currency && (
                    <span className={styles.local}> ({moneyText(item.plannedCost)})</span>
                  )}
                </span>
              </li>
            );
          })}
        </ul>
      )}
    </fieldset>
  );
}

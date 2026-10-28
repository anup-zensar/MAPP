import { useEffect, useState } from 'react';
import { Button } from '@/ui/Button/Button';
import { SearchField } from '@/ui/SearchField/SearchField';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { formatMoneyText } from '@/format/decimal';
import { useLineItemOptions } from './api';
import type { LineItemOptionQuery, PurchaseRequestLineItemOption } from './api';
import { chooseLabel, noMatchText, PICKER } from './form-text';
import styles from './LineItemPicker.module.css';

export const SEARCH_DELAY_MS = 300;

export interface LineItemOptionsListProps {
  /** The PR (purchaseRequestId) or the form's plan and company code (planId, companyCode). */
  query: Omit<LineItemOptionQuery, 'search'>;
  inForm?: string[];
  /** The visible words of each row's button: "Add line" in the dialog, "Choose" on the first step. */
  action: string;
  onChoose: (option: PurchaseRequestLineItemOption) => void;
}

/**
 * The line-item search of Add line and of New purchase request's first step (032 FR-001, FR-010):
 * name, ID, HX key and path with Available to request; unavailable ones listed with the reason.
 */
export function LineItemOptionsList({
  query,
  inForm = [],
  action,
  onChoose,
}: LineItemOptionsListProps) {
  const [typed, setTyped] = useState('');
  const search = useSettled(typed.trim(), SEARCH_DELAY_MS);
  const options = useLineItemOptions({ ...query, ...(search ? { search } : {}) });

  return (
    <div className={styles.picker}>
      <SearchField
        label={PICKER.search}
        placeholder={PICKER.placeholder}
        value={typed}
        onChange={setTyped}
      />
      <div aria-live="polite" className={styles.results}>
        {options.data ? (
          <OptionRows
            options={options.data.items}
            inForm={inForm}
            search={search}
            action={action}
            onChoose={onChoose}
          />
        ) : options.isError ? (
          <div role="alert" className={styles.failed}>
            <p>{PICKER.failed}</p>
            <Button size="sm" onClick={() => void options.refetch()}>
              {PICKER.tryAgain}
            </Button>
          </div>
        ) : (
          <>
            <p className="visually-hidden">{PICKER.loading}</p>
            <Skeleton lines={3} />
          </>
        )}
      </div>
    </div>
  );
}

interface OptionRowsProps {
  options: PurchaseRequestLineItemOption[];
  inForm: string[];
  search: string;
  action: string;
  onChoose: (option: PurchaseRequestLineItemOption) => void;
}

function OptionRows({ options, inForm, search, action, onChoose }: OptionRowsProps) {
  if (options.length === 0) {
    return <p className={styles.empty}>{noMatchText(search)}</p>;
  }
  return (
    <ul role="list" className={styles.list}>
      {options.map((option) => {
        const { lineItem } = option;
        const taken = inForm.includes(lineItem.id);
        const reason = taken ? PICKER.alreadyOnForm : option.available ? '' : option.reason;
        const activity = option.path.find((step) => step.type === 'activity');
        return (
          <li key={lineItem.id} className={reason ? styles.unavailable : styles.option}>
            <div className={styles.facts}>
              <p className={styles.name}>{`${lineItem.name} (${lineItem.id})`}</p>
              <p className={styles.meta}>
                {[option.hxKey, activity?.name ?? lineItem.activityName, lineItem.companyCode]
                  .filter(Boolean)
                  .join(' · ')}
              </p>
              <p className={styles.meta}>
                {`${PICKER.available}: ${formatMoneyText(option.availableToRequest)}`}
              </p>
              {reason && <p className={styles.reason}>{reason}</p>}
            </div>
            {!reason && (
              <Button
                size="sm"
                aria-label={chooseLabel(action, lineItem.name, lineItem.id)}
                onClick={() => onChoose(option)}
              >
                {action}
              </Button>
            )}
          </li>
        );
      })}
    </ul>
  );
}

function useSettled(value: string, delay: number): string {
  const [settled, setSettled] = useState(value);
  useEffect(() => {
    const timer = setTimeout(() => setSettled(value), delay);
    return () => clearTimeout(timer);
  }, [value, delay]);
  return settled;
}

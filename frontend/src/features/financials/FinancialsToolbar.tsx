import { useId } from 'react';
import { FilterChips } from '@/ui/FilterChips/FilterChips';
import type { FilterChip } from '@/ui/FilterChips/FilterChips';
import type { FinancialRecordRef, FinancialTransactionPage } from './api';
import { DelayedSearch } from './DelayedSearch';
import { clearedFilters } from './financials-links';
import type { FinancialFilters } from './financials-links';
import { countLine } from './financials-text';
import styles from './FinancialsPage.module.css';

const LEVEL_WORDS: Record<FinancialRecordRef['type'], string> = {
  plan: 'Plan',
  campaign: 'Campaign',
  initiative: 'Initiative',
  activity: 'Activity',
  lineItem: 'Line item',
  purchaseRequest: 'Purchase request',
  purchaseRequestLine: 'PR line',
};

interface FinancialsToolbarProps {
  filters: FinancialFilters;
  page: FinancialTransactionPage | undefined;
  onChange: (patch: Partial<FinancialFilters>, replace?: boolean) => void;
}

/** Search, Year and Plan, the count at the end; the record and plan filters as chips below. */
export function FinancialsToolbar({ filters, page, onChange }: FinancialsToolbarProps) {
  const ids = useId();
  const choices = page?.choices;
  const year = filters.year ?? page?.year ?? choices?.currentYear;
  const years = withChosen(choices?.years ?? [], year);
  const plans = choices?.plans ?? [];
  const planName = plans.find((plan) => plan.id === filters.planId)?.name ?? filters.planId;

  const chips: FilterChip[] = [];
  if (filters.recordId) {
    const record = page?.recordFilter;
    chips.push({
      key: 'recordId',
      label: record
        ? `${LEVEL_WORDS[record.type]}: ${record.name} (${record.id})`
        : `Record: ${filters.recordId}`,
    });
  }
  if (filters.planId) {
    chips.push({ key: 'planId', label: `Plan: ${planName}` });
  }

  return (
    <div className={styles.toolbar}>
      <div className={styles.row}>
        <div className={styles.search}>
          <DelayedSearch value={filters.q} onChange={(q) => onChange({ q }, true)} />
        </div>
        <div className={styles.field}>
          <label htmlFor={`${ids}-year`}>Year</label>
          <select
            id={`${ids}-year`}
            className={`${styles.control} ${styles.select} ${styles.year}`}
            value={year ?? ''}
            onChange={(event) => onChange({ year: Number(event.target.value), planId: '' })}
          >
            {years.map((choice) => (
              <option key={choice} value={choice}>
                {choice}
              </option>
            ))}
          </select>
        </div>
        <div className={styles.field}>
          <label htmlFor={`${ids}-plan`}>Plan</label>
          <select
            id={`${ids}-plan`}
            className={`${styles.control} ${styles.select}`}
            value={filters.planId}
            onChange={(event) => onChange({ planId: event.target.value })}
          >
            <option value="">All plans</option>
            {plans.map((plan) => (
              <option key={plan.id} value={plan.id}>
                {plan.name}
              </option>
            ))}
            {filters.planId && !plans.some((plan) => plan.id === filters.planId) && (
              <option value={filters.planId}>{filters.planId}</option>
            )}
          </select>
        </div>
        {page && (
          <p className={styles.count} aria-live="polite">
            {countLine(page.total, page.counts.all)}
          </p>
        )}
      </div>
      <FilterChips
        chips={chips}
        onRemove={(key) => onChange({ [key]: '' })}
        onClearAll={() => onChange(clearedFilters(filters))}
      />
    </div>
  );
}

/** The years to offer, with the chosen one even when no plan or transaction has it. */
function withChosen(years: number[], chosen: number | undefined): number[] {
  return chosen === undefined || years.includes(chosen)
    ? years
    : [...years, chosen].sort((a, b) => b - a);
}

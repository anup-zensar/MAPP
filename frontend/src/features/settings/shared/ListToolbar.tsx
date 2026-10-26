import type { ReactNode } from 'react';
import { SearchField } from '@/ui/SearchField/SearchField';
import { SegmentedFilter } from '@/ui/SegmentedFilter/SegmentedFilter';
import { formatWhole } from '@/format/decimal';
import styles from './SettingsTable.module.css';
import type { StatusFilter } from './useListFilters';

/** How a list names what it counts: "model" and "models". */
export interface ListNoun {
  one: string;
  many: string;
}

/**
 * The count line of every list, written one way (design system §5.2, §11 rule 14): "4 of 5
 * models", "6 of 6 people", "1 of 1 team".
 */
function countText(shown: number, total: number, noun: ListNoun): string {
  return `${formatWhole(shown)} of ${formatWhole(total)} ${total === 1 ? noun.one : noun.many}`;
}

export interface ListToolbarProps {
  /** The search box: its hidden label says what it searches ("Search models"). */
  search: {
    label: string;
    placeholder: string;
    value: string;
    onChange: (value: string) => void;
  };
  /** The record-state filter: Active (n) · Inactive (n) · All (n), Active first (§5.2). */
  status?: {
    value: StatusFilter;
    onChange: (value: StatusFilter) => void;
    counts: Record<StatusFilter, number>;
  };
  /** Other filters of the list, after the status filter. */
  filters?: ReactNode;
  /** "4 of 5 models", pushed to the end of the row. */
  count: { shown: number; total: number; noun: ListNoun };
}

/**
 * The one list toolbar of Settings, People and Teams (design system §5.2 "List toolbar", UI audit
 * X-16, PE-01): search · status filter · other filters · spacer · result count. It wraps on narrow
 * screens, the search taking the first row below 600 px.
 */
export function ListToolbar({ search, status, filters, count }: ListToolbarProps) {
  return (
    <div className={styles.toolbar}>
      <div className={styles.search}>
        <SearchField
          label={search.label}
          placeholder={search.placeholder}
          value={search.value}
          onChange={search.onChange}
        />
      </div>
      {status && (
        <SegmentedFilter
          label="Status"
          options={[
            { value: 'active', label: 'Active', count: status.counts.active },
            { value: 'inactive', label: 'Inactive', count: status.counts.inactive },
            { value: 'all', label: 'All', count: status.counts.all },
          ]}
          value={status.value}
          onChange={status.onChange}
        />
      )}
      {filters}
      <p className={styles.resultCount} aria-live="polite">
        {countText(count.shown, count.total, count.noun)}
      </p>
    </div>
  );
}

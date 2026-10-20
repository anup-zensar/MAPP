import { Check, ChevronDown, CircleAlert } from 'lucide-react';
import { DropdownMenu } from 'radix-ui';
import { useEffect, useEffectEvent, useId, useState } from 'react';
import type { ReactNode } from 'react';
import type { components } from '@/api/schema';
import { FilterChips, SearchField, SegmentedFilter } from '@/components';
import { useFormatters } from '@/lib/useFormatters';
import {
  ACTION_FILTERS,
  DATE_RANGES,
  NO_FILTERS,
  customRangeError,
  filterChips,
  removeFilter,
  typeLabel,
} from './logFilters';
import type { ActionFilter, DateRangeKey, LogFilterState } from './logFilters';
import styles from './LogFilterBar.module.css';

type ActivityLogSummary = components['schemas']['ActivityLogSummary'];
type RecordRef = components['schemas']['RecordRef'];

/** How long the search waits after the last key before it filters (plan "Filters"). */
export const SEARCH_DELAY_MS = 300;

export interface LogFilterBarProps {
  filters: LogFilterState;
  /** Applies new filters; `typing` is true for the search box, which should not add to history. */
  onChange: (filters: LogFilterState, typing?: boolean) => void;
  /** The summary's choices for Person, Record type and Plan; undefined while they load. */
  summary: ActivityLogSummary | undefined;
  /** The record of the Record filter, for its chip. */
  record: RecordRef | null;
  /** The result count ("12 of 1,204 entries"), at the end of the date row (design system §5.2). */
  count?: ReactNode;
}

/**
 * The activity log's filters (FR-003): search, Person, Record type, Action (several), Plan and the
 * date range (with Custom dates), then every filter in use as a removable chip and Clear filters.
 */
export function LogFilterBar({ filters, onChange, summary, record, count }: LogFilterBarProps) {
  const { date } = useFormatters();
  const ids = useId();
  const people = summary?.people ?? [];
  const recordTypes = summary?.recordTypes ?? [];
  const plans = summary?.plans ?? [];
  const rangeError = customRangeError(filters);
  const chips = filterChips(filters, { people, recordTypes, plans, record, formatDate: date });

  const set = (patch: Partial<LogFilterState>) => onChange({ ...filters, ...patch });

  return (
    <div className={styles.filters}>
      <div className={styles.row}>
        <div className={styles.search}>
          <DelayedSearch value={filters.q} onChange={(q) => onChange({ ...filters, q }, true)} />
        </div>
        <Select
          id={`${ids}-person`}
          label="Person"
          value={filters.person}
          onChange={(person) => set({ person })}
          options={[
            { value: '', label: 'Anyone' },
            ...withChosen(
              people.map((person) => ({ value: person.id, label: person.name })),
              filters.person,
            ),
          ]}
        />
        <Select
          id={`${ids}-type`}
          label="Record type"
          value={filters.type}
          onChange={(type) => set({ type })}
          options={[
            { value: '', label: 'Any type' },
            ...withChosen(
              recordTypes.map((choice) => ({
                value: choice.type,
                label: typeLabel(choice.type, recordTypes),
              })),
              filters.type,
            ),
          ]}
        />
        <ActionMenu chosen={filters.actions} onChange={(actions) => set({ actions })} />
        <Select
          id={`${ids}-plan`}
          label="Plan"
          value={filters.plan}
          onChange={(plan) => set({ plan })}
          options={[
            { value: '', label: 'Any plan' },
            ...withChosen(
              plans.map((plan) => ({ value: plan.id, label: plan.name })),
              filters.plan,
            ),
          ]}
        />
      </div>

      <div className={styles.row}>
        <div className={styles.date}>
          <span className={styles.dateLabel} aria-hidden="true">
            Date
          </span>
          <SegmentedFilter<DateRangeKey | ''>
            label="Date"
            options={DATE_RANGES}
            value={filters.range}
            onChange={(range) => set(range === 'custom' ? { range } : { range, from: '', to: '' })}
          />
        </div>
        {filters.range === 'custom' && (
          <div className={styles.dates}>
            <div className={styles.field}>
              <label htmlFor={`${ids}-from`}>From</label>
              <input
                id={`${ids}-from`}
                type="date"
                className={styles.control}
                value={filters.from}
                onChange={(event) => set({ from: event.target.value })}
              />
            </div>
            <div className={styles.field}>
              <label htmlFor={`${ids}-to`}>To</label>
              <input
                id={`${ids}-to`}
                type="date"
                className={styles.control}
                value={filters.to}
                min={filters.from || undefined}
                aria-invalid={rangeError ? true : undefined}
                aria-describedby={rangeError ? `${ids}-to-error` : undefined}
                onChange={(event) => set({ to: event.target.value })}
              />
            </div>
            {rangeError && (
              <p id={`${ids}-to-error`} className={styles.error} role="alert">
                <CircleAlert aria-hidden="true" />
                {rangeError}
              </p>
            )}
          </div>
        )}
        {count && <div className={styles.count}>{count}</div>}
      </div>

      <FilterChips
        chips={chips}
        onRemove={(key) => onChange(removeFilter(filters, key))}
        onClearAll={() => onChange(NO_FILTERS)}
      />
    </div>
  );
}

interface Option {
  value: string;
  label: string;
}

/** The choices, plus the chosen value when the summary does not list it (an old link). */
function withChosen(options: Option[], chosen: string): Option[] {
  return chosen && !options.some((option) => option.value === chosen)
    ? [...options, { value: chosen, label: chosen }]
    : options;
}

interface SelectProps {
  id: string;
  label: string;
  value: string;
  onChange: (value: string) => void;
  options: Option[];
}

/** A small labelled select of the toolbar. */
function Select({ id, label, value, onChange, options }: SelectProps) {
  return (
    <div className={styles.field}>
      <label htmlFor={id}>{label}</label>
      <select
        id={id}
        className={`${styles.control} ${styles.select}`}
        value={value}
        onChange={(event) => onChange(event.target.value)}
      >
        {options.map((option) => (
          <option key={option.value} value={option.value}>
            {option.label}
          </option>
        ))}
      </select>
    </div>
  );
}

/**
 * The Action filter: a menu of eight check boxes that stays open while people tick several
 * (`modal={false}`, so the page behind keeps scrolling and screen readers keep the page).
 */
function ActionMenu({
  chosen,
  onChange,
}: {
  chosen: ActionFilter[];
  onChange: (actions: ActionFilter[]) => void;
}) {
  const labelId = useId();
  const words = ACTION_FILTERS.filter((choice) => chosen.includes(choice.value)).map(
    (choice) => choice.word,
  );
  const summary = words.length > 0 ? words.join(', ') : 'Any action';

  const toggle = (value: ActionFilter, checked: boolean) => {
    const next = checked ? [...chosen, value] : chosen.filter((action) => action !== value);
    // Kept in the menu's order, so the address and the chips do not depend on the clicks' order.
    onChange(
      ACTION_FILTERS.map((choice) => choice.value).filter((action) => next.includes(action)),
    );
  };

  return (
    <div className={styles.field}>
      <span id={labelId}>Action</span>
      <DropdownMenu.Root modal={false}>
        <DropdownMenu.Trigger asChild>
          <button
            type="button"
            className={`${styles.control} ${styles.menuButton}`}
            aria-label={`Action: ${summary}`}
          >
            <span className={styles.menuText}>{summary}</span>
            <ChevronDown aria-hidden="true" />
          </button>
        </DropdownMenu.Trigger>
        <DropdownMenu.Portal>
          <DropdownMenu.Content
            className={styles.menu}
            align="start"
            sideOffset={4}
            aria-labelledby={labelId}
          >
            {ACTION_FILTERS.map((choice) => (
              <DropdownMenu.CheckboxItem
                key={choice.value}
                className={styles.menuItem}
                checked={chosen.includes(choice.value)}
                onCheckedChange={(checked) => toggle(choice.value, checked)}
                // Keeps the menu open, so people can tick several actions.
                onSelect={(event) => event.preventDefault()}
              >
                <span className={styles.box} aria-hidden="true">
                  <DropdownMenu.ItemIndicator>
                    <Check />
                  </DropdownMenu.ItemIndicator>
                </span>
                {choice.word}
              </DropdownMenu.CheckboxItem>
            ))}
          </DropdownMenu.Content>
        </DropdownMenu.Portal>
      </DropdownMenu.Root>
    </div>
  );
}

/**
 * The search box: it filters 300 ms after the last key (plan "Filters"), and shows the address's
 * search again when that changes elsewhere (a chip removed, Clear filters, Back).
 */
function DelayedSearch({ value, onChange }: { value: string; onChange: (q: string) => void }) {
  // What the person typed, and the applied search it was typed over.
  const [draft, setDraft] = useState<{ text: string; over: string } | null>(null);
  const text = draft && draft.over === value ? draft.text : value;
  const apply = useEffectEvent((q: string) => onChange(q));

  useEffect(() => {
    if (!draft || draft.over !== value || draft.text === value) {
      return;
    }
    const timer = setTimeout(() => apply(draft.text), SEARCH_DELAY_MS);
    return () => clearTimeout(timer);
  }, [draft, value]);

  return (
    <SearchField
      label="Search name, ID or person"
      placeholder="Search name, ID or person"
      value={text}
      onChange={(next) => setDraft({ text: next, over: value })}
    />
  );
}

import { SlidersHorizontal } from 'lucide-react';
import { useEffect, useId, useRef, useState } from 'react';
import { Button } from '@/ui/Button/Button';
import { FilterChips } from '@/ui/FilterChips/FilterChips';
import { SearchField } from '@/ui/SearchField/SearchField';
import { cx } from '@/ui/cx';
import { formatWhole } from '@/format/decimal';
import styles from './FilterBar.module.css';
import {
  ACTIVITY_STATUS_LABELS,
  CONTAINER_STATUS_LABELS,
  countText,
  FILTER_LABELS,
  FINANCE_LABELS,
  isLabelled,
  PLAN_STATUS_LABELS,
  PLAN_TYPE_LABELS,
  SHOW_LEVELS_LABELS,
  STRATEGIC_LABELS,
} from './all-work-text';
import type {
  AllWorkOptions,
  AllWorkResult,
  FilterValue,
  Filters,
  ViewState,
} from './all-work-types';
import { AttributeFilterMenu } from './AttributeFilterMenu';
import { FilterMenu } from './FilterMenu';
import type { FilterChoice } from './FilterMenu';
import { chipsOf, clearAll, hasFilters, removeChip } from './view-chips';
import { choicesOf, withYears, yearRange } from './view-state';
import type { IdFilter, ValueFilter } from './view-state';
import { YearRangeMenu } from './YearRangeMenu';

const SEARCH_DELAY_MS = 300;

const APPROVAL_PROGRESS = ['noActivities', 'inProgress', 'allApproved'];

export interface FilterBarProps {
  state: ViewState;
  /** The choices of the year; undefined while they load. */
  options: AllWorkOptions | undefined;
  /** "Showing N of M activities"; undefined while the first answer loads. */
  count: AllWorkResult['count'] | undefined;
  updating: boolean;
  onChange: (change: (current: ViewState) => ViewState) => void;
}

// The one filter bar of every view: filters apply as soon as they change, each value is a removable
// chip. Below 600 px the filter buttons fold into a Filters (n) drawer, so the data starts higher.
export function FilterBar({ state, options, count, updating, onChange }: FilterBarProps) {
  const menusId = useId();
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [typed, setTyped] = useState(state.search);
  const [searchShown, setSearchShown] = useState(state.search);
  const timer = useRef<number | undefined>(undefined);
  useEffect(() => () => window.clearTimeout(timer.current), []);

  // The search changed elsewhere (Clear all, a saved view): the box follows.
  if (state.search !== searchShown) {
    setSearchShown(state.search);
    setTyped(state.search);
  }

  function typeSearch(text: string) {
    setTyped(text);
    window.clearTimeout(timer.current);
    timer.current = window.setTimeout(
      () => onChange((current) => ({ ...current, search: text })),
      SEARCH_DELAY_MS,
    );
  }

  const { filters } = state;
  const setFilters = (change: Partial<Filters>) =>
    onChange((current) => ({ ...current, filters: { ...current.filters, ...change } }));

  /** The menu of a filter that holds IDs; a chosen value keeps the name it had. */
  function idMenu(filter: IdFilter) {
    const choices = choicesOf(filter, options);
    const chosen = filters[filter];
    return {
      label: FILTER_LABELS[filter],
      multiple: true,
      choices: choices.map((choice) => ({
        value: choice.id,
        label: choice.label,
        inactive: choice.inactive,
      })),
      selected: chosen.map((value) => value.id),
      summary: chosen.length > 0 ? formatWhole(chosen.length) : undefined,
      onChange: (ids: string[]) =>
        setFilters({
          [filter]: ids.map((id): FilterValue => ({
            id,
            label:
              choices.find((choice) => choice.id === id)?.label ??
              chosen.find((value) => value.id === id)?.label ??
              id,
          })),
        }),
    };
  }

  function valueMenu(
    filter: ValueFilter,
    labels: Record<string, string>,
    set: (values: string[]) => void,
    groupOf?: (value: string) => string,
  ) {
    const chosen: string[] = filters[filter] ?? [];
    return {
      label: FILTER_LABELS[filter],
      multiple: true,
      choices: Object.entries<string>(labels).map(([value, label]): FilterChoice => ({
        value,
        label,
        group: groupOf?.(value),
      })),
      selected: chosen,
      summary: chosen.length > 0 ? formatWhole(chosen.length) : undefined,
      onChange: set,
    };
  }

  const years = options ? options.years : [];
  const range = yearRange(state);
  const currencies = options?.planCurrencies ?? [];
  const menus = [
    idMenu('plans'),
    valueMenu('planStatuses', PLAN_STATUS_LABELS, (values) =>
      setFilters({ planStatuses: values.filter(isLabelled(PLAN_STATUS_LABELS)) }),
    ),
    valueMenu('planTypes', PLAN_TYPE_LABELS, (values) =>
      setFilters({ planTypes: values.filter(isLabelled(PLAN_TYPE_LABELS)) }),
    ),
    ...(currencies.length > 0 || (filters.planCurrencies?.length ?? 0) > 0
      ? [
          valueMenu(
            'planCurrencies',
            Object.fromEntries(currencies.map((code) => [code, code])),
            (planCurrencies) => setFilters({ planCurrencies }),
          ),
        ]
      : []),
    valueMenu(
      'containerStatuses',
      CONTAINER_STATUS_LABELS,
      (values) =>
        setFilters({ containerStatuses: values.filter(isLabelled(CONTAINER_STATUS_LABELS)) }),
      (value) => (APPROVAL_PROGRESS.includes(value) ? 'Approval progress' : 'Timing'),
    ),
    valueMenu('activityStatuses', ACTIVITY_STATUS_LABELS, (values) =>
      setFilters({ activityStatuses: values.filter(isLabelled(ACTIVITY_STATUS_LABELS)) }),
    ),
    idMenu('owners'),
    {
      ...idMenu('teams'),
      // My teams first: it ticks the person's teams.
      quickChoice: {
        label: 'My teams',
        values: (options?.teams ?? []).filter((team) => team.mine).map((team) => team.id),
      },
    },
    idMenu('businessUnits'),
    idMenu('activityTypes'),
    valueMenu('finance', FINANCE_LABELS, (values) =>
      setFilters({ finance: values.filter(isLabelled(FINANCE_LABELS)) }),
    ),
    {
      label: FILTER_LABELS.strategic,
      multiple: false,
      choices: Object.entries(STRATEGIC_LABELS).map(([value, label]) => ({ value, label })),
      selected: [filters.strategic],
      summary: filters.strategic === 'include' ? undefined : STRATEGIC_LABELS[filters.strategic],
      onChange: ([value = '']: string[]) => {
        if (isLabelled(STRATEGIC_LABELS)(value)) {
          setFilters({ strategic: value });
        }
      },
    },
    {
      label: FILTER_LABELS.showLevels,
      multiple: false,
      choices: Object.entries(SHOW_LEVELS_LABELS).map(([value, label]) => ({ value, label })),
      selected: [filters.showLevels],
      summary:
        filters.showLevels === 'everything' ? undefined : SHOW_LEVELS_LABELS[filters.showLevels],
      onChange: ([value = '']: string[]) => {
        if (isLabelled(SHOW_LEVELS_LABELS)(value)) {
          setFilters({ showLevels: value });
        }
      },
    },
  ];

  function clear() {
    window.clearTimeout(timer.current);
    setTyped('');
    onChange(clearAll);
  }

  const chips = chipsOf(state, options);
  const hasAttributes = Boolean(options?.attributes?.length) || Boolean(filters.attributes?.length);

  return (
    <div className={styles.bar}>
      <div className={styles.top}>
        <div className={styles.search}>
          <SearchField
            label="Search"
            placeholder="Search name, ID or description"
            value={typed}
            onChange={typeSearch}
          />
        </div>
        <Button
          size="md"
          icon={<SlidersHorizontal />}
          className={styles.drawerButton}
          aria-expanded={drawerOpen}
          aria-controls={menusId}
          onClick={() => setDrawerOpen(!drawerOpen)}
        >
          {chips.length > 0 ? `Filters (${formatWhole(chips.length)})` : 'Filters'}
        </Button>
        <p className={styles.count} aria-live="polite">
          {count && countText(count.shown, count.total, count.noun)}
          {updating && <span className={styles.updating}> Updating…</span>}
        </p>
      </div>
      <div
        id={menusId}
        className={cx(styles.menus, drawerOpen && styles.open)}
        role="group"
        aria-label="Filters"
      >
        {/* Another year range: the plans of other years cannot be chosen any more. */}
        <YearRangeMenu
          years={years}
          from={range.from}
          to={range.to}
          onChange={(from, to) => onChange((current) => withYears(current, from, to))}
        />
        {menus.map((menu) => (
          <FilterMenu key={menu.label} {...menu} />
        ))}
        {hasAttributes && (
          <AttributeFilterMenu
            attributes={options?.attributes ?? []}
            selected={filters.attributes ?? []}
            onChange={(attributes) => setFilters({ attributes })}
          />
        )}
      </div>
      {hasFilters(state) && (
        <div className={styles.results}>
          <FilterChips
            chips={chips}
            onRemove={(key) => onChange((current) => removeChip(current, key))}
          />
          <button type="button" className={styles.clearAll} onClick={clear}>
            Clear all
          </button>
        </div>
      )}
    </div>
  );
}

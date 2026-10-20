import { SlidersHorizontal } from 'lucide-react';
import { useEffect, useId, useRef, useState } from 'react';
import { Button, FilterChips, SearchField } from '@/components';
import { cx } from '@/lib/cx';
import { formatWhole } from '@/lib/decimal';
import styles from './FilterBar.module.css';
import {
  ACTIVITY_STATUS_LABELS,
  CONTAINER_STATUS_LABELS,
  countText,
  FILTER_LABELS,
  PLAN_STATUS_LABELS,
  SHOW_LEVELS_LABELS,
  STRATEGIC_LABELS,
} from './allWorkText';
import type {
  AllWorkOptions,
  AllWorkResult,
  FilterValue,
  Filters,
  ViewState,
} from './allWorkTypes';
import { FilterMenu } from './FilterMenu';
import type { FilterChoice } from './FilterMenu';
import { chipsOf, choicesOf, clearAll, hasFilters, removeChip } from './viewState';
import type { IdFilter } from './viewState';

/** How long the search waits after the last key before the view follows (plan "Page layout"). */
const SEARCH_DELAY_MS = 300;

/** Approval progress and Timing, the two groups of the campaign / initiative status (FR-004). */
const APPROVAL_PROGRESS = ['noActivities', 'inProgress', 'allApproved'];

export interface FilterBarProps {
  state: ViewState;
  /** The choices of the year; undefined while they load. */
  options: AllWorkOptions | undefined;
  /** "Showing N of M activities"; undefined while the first answer loads. */
  count: AllWorkResult['count'] | undefined;
  /** A new answer is on its way ("Updating…"). */
  updating: boolean;
  /** Changes the state; the change is a function of the state as it is then. */
  onChange: (change: (current: ViewState) => ViewState) => void;
}

/**
 * The one filter bar of the three views (US2, FR-004 – FR-010): the search box with the count at
 * the row's end, exactly the filters of FR-004 — no Shell-attribute filters (FR-010, D-207) —
 * which apply as soon as they change, and a removable chip per value with Clear all. Below 600 px
 * the filter buttons fold into a Filters (n) drawer, so the data starts higher (UI audit AW-03,
 * PH-04).
 */
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

  /** The menu of a filter that holds statuses. */
  function valueMenu<K extends 'planStatuses' | 'containerStatuses' | 'activityStatuses'>(
    filter: K,
    labels: Record<Filters[K][number], string>,
    groupOf?: (value: string) => string,
  ) {
    const chosen = filters[filter] as string[];
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
      onChange: (values: string[]) => setFilters({ [filter]: values } as Partial<Filters>),
    };
  }

  const years = options ? options.years : [];
  const menus = [
    {
      label: FILTER_LABELS.year,
      multiple: false,
      choices: (years.includes(state.year) ? years : [state.year, ...years]).map((year) => ({
        value: String(year),
        label: String(year),
      })),
      selected: [String(state.year)],
      summary: String(state.year),
      // The plans of another year cannot be chosen any more.
      onChange: ([year]: string[]) =>
        onChange((current) => ({
          ...current,
          year: Number(year),
          filters: { ...current.filters, plans: [] },
        })),
    },
    idMenu('plans'),
    valueMenu('planStatuses', PLAN_STATUS_LABELS),
    valueMenu('containerStatuses', CONTAINER_STATUS_LABELS, (value) =>
      APPROVAL_PROGRESS.includes(value) ? 'Approval progress' : 'Timing',
    ),
    valueMenu('activityStatuses', ACTIVITY_STATUS_LABELS),
    idMenu('owners'),
    {
      ...idMenu('teams'),
      // My teams first (FR-004): it ticks the person's teams.
      quickChoice: {
        label: 'My teams',
        values: (options?.teams ?? []).filter((team) => team.mine).map((team) => team.id),
      },
    },
    idMenu('businessUnits'),
    idMenu('activityTypes'),
    {
      label: FILTER_LABELS.strategic,
      multiple: false,
      choices: Object.entries(STRATEGIC_LABELS).map(([value, label]) => ({ value, label })),
      selected: [filters.strategic],
      summary: filters.strategic === 'include' ? undefined : STRATEGIC_LABELS[filters.strategic],
      onChange: ([value]: string[]) => setFilters({ strategic: value as Filters['strategic'] }),
    },
    {
      label: FILTER_LABELS.showLevels,
      multiple: false,
      choices: Object.entries(SHOW_LEVELS_LABELS).map(([value, label]) => ({ value, label })),
      selected: [filters.showLevels],
      summary:
        filters.showLevels === 'everything' ? undefined : SHOW_LEVELS_LABELS[filters.showLevels],
      onChange: ([value]: string[]) => setFilters({ showLevels: value as Filters['showLevels'] }),
    },
  ];

  function clear() {
    window.clearTimeout(timer.current);
    setTyped('');
    onChange(clearAll);
  }

  const chips = chipsOf(state, options);

  return (
    <div className={styles.bar}>
      <div className={styles.top}>
        <div className={styles.search}>
          <SearchField
            label="Search"
            placeholder="Search name, ID, external name or description"
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
        {menus.map((menu) => (
          <FilterMenu key={menu.label} {...menu} />
        ))}
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

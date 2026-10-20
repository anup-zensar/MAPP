import { ClipboardList, Plus, SearchX } from 'lucide-react';
import { useEffect, useRef, useState } from 'react';
import { useSearchParams } from 'react-router';
import { useCurrentUser } from '@/app/useSession';
import {
  Button,
  EmptyState,
  ErrorState,
  FilterChips,
  LinkButton,
  PageHeader,
  SearchField,
  SegmentedFilter,
  Skeleton,
  SummaryCard,
} from '@/components';
import type { FilterChip } from '@/components';
import { useAccess } from '@/lib/access';
import { useFormatters } from '@/lib/useFormatters';
import { usePageTitle } from '@/lib/usePageTitle';
import {
  DEFAULT_FILTERS,
  readFilters,
  remember,
  toSearchParams,
  yearRangeLabel,
} from './listPreferences';
import type { PlanListFilters } from './listPreferences';
import { PlanCard } from './PlanCard';
import { PlansFilters } from './PlansFilters';
import styles from './PlansPage.module.css';
import { usePlans } from './plansApi';
import { PLAN_TYPE_LABELS } from './planCurrency';
import { PlanTable } from './PlanTable';
import { MY_TEAMS_HINT, STATUS } from './planText';
import { usePlanActions } from './usePlanActions';

/** How long the search waits after the last key before it asks the API (plan "Plans list"). */
const SEARCH_DELAY_MS = 250;

/**
 * Plans (US2, FR-001 – FR-004; 016 FR-033): every plan as cards or a table, with summary cards,
 * status, years, plan type and team filters, search on name or ID, removable filter chips and the result count. Filters
 * live in the address and are remembered per person.
 */
export function PlansPage() {
  usePageTitle('Plans');
  const user = useCurrentUser();
  const canCreate = useAccess().has('plans.create');
  const { money, moneyCompact } = useFormatters();
  const [search, setSearch] = useSearchParams();
  // 016's API sends every plan's type; until then the list takes one year and no plan types, so
  // a plan type in the address is neither sent nor shown as a chip.
  const [served016, setServed016] = useState(false);
  const chosen = readFilters(search, user.id);
  const filters = served016 ? chosen : { ...chosen, types: [] };
  const [typed, setTyped] = useState(filters.q);
  const plans = usePlans({
    status: filters.status,
    yearFrom: filters.yearFrom,
    yearTo: filters.yearTo,
    types: filters.types,
    scope: filters.scope,
    q: filters.q,
  });
  const actions = usePlanActions();
  const answersTypes = plans.data?.items.some((plan) => plan.planType !== undefined) ?? false;
  if (answersTypes && !served016) {
    setServed016(true);
  }

  /** Changes some filters: in the address (replacing it) and in what is remembered. */
  function change(next: Partial<PlanListFilters>) {
    setSearch(
      (current) => {
        const all = { ...readFilters(current, user.id), ...next };
        remember(user.id, all);
        return toSearchParams(all);
      },
      { replace: true },
    );
  }

  // The search is sent 250 ms after typing stops.
  const searchTimer = useRef<number | undefined>(undefined);
  useEffect(() => () => window.clearTimeout(searchTimer.current), []);

  function typeSearch(text: string) {
    setTyped(text);
    window.clearTimeout(searchTimer.current);
    searchTimer.current = window.setTimeout(() => change({ q: text }), SEARCH_DELAY_MS);
  }

  function clearSearch() {
    window.clearTimeout(searchTimer.current);
    setTyped('');
  }

  function clearFilters() {
    clearSearch();
    change({ ...DEFAULT_FILTERS, layout: filters.layout });
  }

  const years = yearRangeLabel(filters);
  const chips: FilterChip[] = [
    ...(filters.status !== 'all' ? [{ key: 'status', label: STATUS[filters.status].word }] : []),
    ...(years !== null
      ? [
          {
            key: 'years',
            label: `${filters.yearFrom === filters.yearTo ? 'Year' : 'Years'}: ${years}`,
          },
        ]
      : []),
    ...(filters.types.length > 0
      ? [
          {
            key: 'types',
            label: `Plan type: ${filters.types.map((type) => PLAN_TYPE_LABELS[type]).join(', ')}`,
          },
        ]
      : []),
    ...(filters.scope === 'myTeams' ? [{ key: 'scope', label: 'My teams’ plans' }] : []),
    ...(filters.q.trim() ? [{ key: 'q', label: `Search: ${filters.q.trim()}` }] : []),
  ];

  function removeChip(key: string) {
    if (key === 'q') {
      clearSearch();
    }
    if (key === 'years') {
      change({ yearFrom: null, yearTo: null });
      return;
    }
    change({ [key]: DEFAULT_FILTERS[key as keyof PlanListFilters] });
  }

  const data = plans.data;
  const summary = data?.summary;
  const planYears = data?.years ?? [];
  const showsTable = filters.layout === 'table';

  /** The list's state when it has no plans to show: loading, failure, empty, filtered out. */
  function listState() {
    if (plans.isError && !data) {
      return <ErrorState error={plans.error} onRetry={() => void plans.refetch()} />;
    }
    if (!data) {
      return <ListSkeleton />;
    }
    if (data.total === 0 && chips.length === 0) {
      return (
        <EmptyState
          icon={<ClipboardList />}
          title="No plans yet — create your first plan"
          text="A plan holds one year of marketing for one business."
          // Secondary: the header already holds the primary New plan (design system §11, rule 1).
          action={
            canCreate && (
              <LinkButton to="/plans/new" icon={<Plus />}>
                Create your first plan
              </LinkButton>
            )
          }
        />
      );
    }
    if (data.items.length === 0) {
      return (
        <EmptyState
          icon={<SearchX />}
          title="Nothing matches these filters"
          action={<Button onClick={clearFilters}>Clear filters</Button>}
        />
      );
    }
    return null;
  }

  const state = listState();

  return (
    <>
      <PageHeader
        title="Plans"
        description="Every yearly marketing plan, its status, its budget and what's still missing."
        actions={
          // 014 FR-011: New plan only for a person who may create plans.
          canCreate && (
            <LinkButton to="/plans/new" variant="primary" icon={<Plus />}>
              New plan
            </LinkButton>
          )
        }
      />
      <div className={styles.page}>
        <section className={styles.summary} aria-label="Summary">
          {summary ? (
            <>
              <SummaryCard
                label="Plans"
                value={summary.plans}
                tip="Plans in the chosen years, plan types and teams."
                detail={
                  <span>
                    {summary.draft} draft · {summary.published} published
                  </span>
                }
              />
              <SummaryCard
                label="Total plan budget"
                value={moneyCompact(summary.budget)}
                fullValue={money(summary.budget)}
                tip="The plan budgets of these plans added up. Plans without a budget add nothing."
              />
              <SummaryCard
                label="Total planned cost"
                value={moneyCompact(summary.plannedCost)}
                fullValue={money(summary.plannedCost)}
                tip="The planned cost of these plans added up: the sum of their activities' planned cost."
              />
            </>
          ) : (
            [0, 1, 2].map((card) => (
              <div key={card} className={styles.skeletonCard}>
                <Skeleton width="60%" />
                <Skeleton height="24px" width="40%" />
              </div>
            ))
          )}
        </section>

        {/* The list toolbar inside the list's card, across its full width (design system §2.4 A). */}
        <section className={styles.listCard} aria-label="Plan list">
          <div className={styles.toolbar}>
            <div className={styles.search}>
              <SearchField
                label="Search plans"
                placeholder="Search name or ID"
                value={typed}
                onChange={typeSearch}
              />
            </div>
            <div className={styles.status}>
              <SegmentedFilter
                label="Status"
                options={[
                  { value: 'all', label: summary ? `All (${summary.plans})` : 'All' },
                  { value: 'draft', label: summary ? `Draft (${summary.draft})` : 'Draft' },
                  {
                    value: 'published',
                    label: summary ? `Published (${summary.published})` : 'Published',
                  },
                ]}
                value={filters.status}
                onChange={(status) => change({ status })}
              />
            </div>
            <PlansFilters
              filters={filters}
              years={planYears}
              served016={served016}
              onChange={change}
            />
            <div className={styles.end}>
              {data && (
                <p className={styles.count} aria-live="polite">
                  {data.items.length} of {data.total} {data.total === 1 ? 'plan' : 'plans'}
                </p>
              )}
              <div className={styles.layout}>
                <SegmentedFilter
                  label="Layout"
                  options={[
                    { value: 'cards', label: 'Cards' },
                    { value: 'table', label: 'Table' },
                  ]}
                  value={filters.layout}
                  onChange={(layout) => change({ layout })}
                />
              </div>
            </div>
          </div>
          {(chips.length > 0 || filters.scope === 'myTeams') && (
            <div className={styles.chips}>
              <FilterChips chips={chips} onRemove={removeChip} onClearAll={clearFilters} />
              {filters.scope === 'myTeams' && <p className={styles.hint}>{MY_TEAMS_HINT}</p>}
            </div>
          )}
          <div aria-busy={plans.isFetching}>
            {state ? (
              <div className={styles.state}>{state}</div>
            ) : (
              showsTable && data && <PlanTable plans={data.items} actionsFor={actions.menuItems} />
            )}
          </div>
        </section>

        {!state && !showsTable && data && (
          <div className={styles.cards}>
            {data.items.map((plan) => (
              <PlanCard key={plan.id} plan={plan} actions={actions.menuItems(plan)} />
            ))}
          </div>
        )}
      </div>
      {actions.dialogs}
    </>
  );
}

/** Grey lines while the plans load. */
function ListSkeleton() {
  return (
    <div className={styles.skeletonList} aria-label="Loading plans">
      {[0, 1, 2].map((row) => (
        <Skeleton key={row} height="40px" />
      ))}
    </div>
  );
}

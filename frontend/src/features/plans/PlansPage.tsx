import { ClipboardList, Plus, SearchX } from 'lucide-react';
import { useEffect, useRef, useState } from 'react';
import { useSearchParams } from 'react-router';
import { useCurrentUser } from '@/app/useSession';
import {
  Button,
  Card,
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
import { useFormatters } from '@/lib/useFormatters';
import { usePageTitle } from '@/lib/usePageTitle';
import { DEFAULT_FILTERS, readFilters, remember, toSearchParams } from './listPreferences';
import type { PlanListFilters } from './listPreferences';
import { PlanCard } from './PlanCard';
import styles from './PlansPage.module.css';
import { usePlans } from './plansApi';
import { PlanTable } from './PlanTable';
import { MY_TEAMS_HINT, STATUS } from './planText';
import { usePlanActions } from './usePlanActions';

/** How long the search waits after the last key before it asks the API (plan "Plans list"). */
const SEARCH_DELAY_MS = 250;

/**
 * Plans (US2, FR-001 – FR-004): every plan as cards or a table, with summary cards, status, year
 * and team filters, search on name or ID, removable filter chips and the result count. Filters
 * live in the address and are remembered per person.
 */
export function PlansPage() {
  usePageTitle('Plans');
  const user = useCurrentUser();
  const { money, moneyCompact } = useFormatters();
  const [search, setSearch] = useSearchParams();
  const filters = readFilters(search, user.id);
  const [typed, setTyped] = useState(filters.q);
  const plans = usePlans({
    status: filters.status,
    year: filters.year,
    scope: filters.scope,
    q: filters.q,
  });
  const actions = usePlanActions();

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

  const chips: FilterChip[] = [
    ...(filters.status !== 'all' ? [{ key: 'status', label: STATUS[filters.status].word }] : []),
    ...(filters.year !== null ? [{ key: 'year', label: String(filters.year) }] : []),
    ...(filters.scope === 'myTeams' ? [{ key: 'scope', label: 'My teams' }] : []),
    ...(filters.q.trim() ? [{ key: 'q', label: `Search: ${filters.q.trim()}` }] : []),
  ];

  function removeChip(key: string) {
    if (key === 'q') {
      clearSearch();
    }
    change({ [key]: DEFAULT_FILTERS[key as keyof PlanListFilters] });
  }

  const data = plans.data;
  const summary = data?.summary;
  const years = data?.years ?? [];
  const yearOptions =
    filters.year !== null && !years.includes(filters.year) ? [...years, filters.year] : years;

  return (
    <>
      <PageHeader
        eyebrow="Plans"
        title="Plans"
        description="Every yearly marketing plan, its status, its budget and what's still missing."
        actions={
          <LinkButton to="/plans/new" variant="primary" icon={<Plus />}>
            New plan
          </LinkButton>
        }
      />
      <div className={styles.page}>
        <section className={styles.summary} aria-label="Summary">
          {summary ? (
            <>
              <SummaryCard
                label="Plans"
                value={summary.plans}
                tip="Plans in the chosen year and teams."
              />
              <SummaryCard label="Draft" value={summary.draft} tip="Plans still being prepared." />
              <SummaryCard
                label="Published"
                value={summary.published}
                tip="Plans agreed as the plan of record."
              />
              <SummaryCard
                label="Total planned cost"
                value={moneyCompact(summary.plannedCost)}
                fullValue={money(summary.plannedCost)}
                tip="The planned cost of these plans added up: the sum of their activities' planned cost."
              />
              <SummaryCard
                label="Total plan budget"
                value={moneyCompact(summary.budget)}
                fullValue={money(summary.budget)}
                tip="The plan budgets of these plans added up. Plans without a budget add nothing."
              />
            </>
          ) : (
            [0, 1, 2, 3, 4].map((card) => (
              <div key={card} className={styles.skeletonCard}>
                <Skeleton width="60%" />
                <Skeleton height="24px" width="40%" />
              </div>
            ))
          )}
        </section>

        <div className={styles.toolbar}>
          <SegmentedFilter
            label="Status"
            options={[
              { value: 'all', label: 'All' },
              { value: 'draft', label: 'Draft' },
              { value: 'published', label: 'Published' },
            ]}
            value={filters.status}
            onChange={(status) => change({ status })}
          />
          <label className={styles.year}>
            <span className="visually-hidden">Year</span>
            <select
              value={filters.year === null ? '' : String(filters.year)}
              onChange={(event) =>
                change({ year: event.target.value ? Number(event.target.value) : null })
              }
            >
              <option value="">All years</option>
              {yearOptions.map((year) => (
                <option key={year} value={year}>
                  {year}
                </option>
              ))}
            </select>
          </label>
          <SegmentedFilter
            label="Show"
            options={[
              { value: 'all', label: 'All plans' },
              { value: 'myTeams', label: 'My teams' },
            ]}
            value={filters.scope}
            onChange={(scope) => change({ scope })}
          />
          <div className={styles.search}>
            <SearchField
              label="Search name or ID"
              placeholder="Search name or ID"
              value={typed}
              onChange={typeSearch}
            />
          </div>
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
        {filters.scope === 'myTeams' && <p className={styles.hint}>{MY_TEAMS_HINT}</p>}
        {(chips.length > 0 || data) && (
          <div className={styles.results}>
            <FilterChips chips={chips} onRemove={removeChip} onClearAll={clearFilters} />
            {data && (
              <p className={styles.count} aria-live="polite">
                {data.items.length} of {data.total}
              </p>
            )}
          </div>
        )}

        <div aria-busy={plans.isFetching}>
          {plans.isError && !data ? (
            <Card>
              <ErrorState error={plans.error} onRetry={() => void plans.refetch()} />
            </Card>
          ) : !data ? (
            <ListSkeleton />
          ) : data.total === 0 && chips.length === 0 ? (
            <Card>
              <EmptyState
                icon={<ClipboardList />}
                title="No plans yet — create your first plan"
                text="A plan holds one year of marketing for one business."
                action={
                  <LinkButton to="/plans/new" variant="primary" icon={<Plus />}>
                    Create your first plan
                  </LinkButton>
                }
              />
            </Card>
          ) : data.items.length === 0 ? (
            <Card>
              <EmptyState
                icon={<SearchX />}
                title="Nothing matches these filters"
                action={<Button onClick={clearFilters}>Clear filters</Button>}
              />
            </Card>
          ) : filters.layout === 'table' ? (
            <PlanTable plans={data.items} actionsFor={actions.menuItems} />
          ) : (
            <div className={styles.cards}>
              {data.items.map((plan) => (
                <PlanCard key={plan.id} plan={plan} actions={actions.menuItems(plan)} />
              ))}
            </div>
          )}
        </div>
      </div>
      {actions.dialogs}
    </>
  );
}

/** Five grey cards while the plans load. */
function ListSkeleton() {
  return (
    <div className={styles.cards} aria-label="Loading plans">
      {[0, 1, 2, 3, 4].map((card) => (
        <div key={card} className={styles.skeletonPlan}>
          <Skeleton width="70%" height="18px" />
          <Skeleton lines={3} />
          <Skeleton height="8px" />
        </div>
      ))}
    </div>
  );
}

import { ClipboardList, Plus, SearchX } from 'lucide-react';
import { useEffect, useRef, useState } from 'react';
import { useSearchParams } from 'react-router';
import { useCurrentUser, useSession } from '@/session/useSession';
import { Button, LinkButton } from '@/ui/Button/Button';
import { EmptyState } from '@/ui/EmptyState/EmptyState';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { FilterChips } from '@/ui/FilterChips/FilterChips';
import { PageHeader } from '@/ui/PageHeader/PageHeader';
import { SearchField } from '@/ui/SearchField/SearchField';
import { SegmentedFilter } from '@/ui/SegmentedFilter/SegmentedFilter';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { useToast } from '@/ui/Toast/useToast';
import { useAccess } from '@/session/access';
import { usePageTitle } from '@/ui/usePageTitle';
import { DEFAULT_FILTERS, readFilters, remember, toSearchParams } from './list-preferences';
import type { PlanListFilters } from './list-preferences';
import { NewPlanButton } from './NewPlanButton';
import { PlanCard } from './PlanCard';
import { PlansFilters } from './PlansFilters';
import styles from './PlansPage.module.css';
import { usePlans } from './api';
import type { PlanSummary } from './api';
import { chipReset, filterChips } from './plan-list-chips';
import { PlanListSummary } from './PlanListSummary';
import { PlanTable } from './PlanTable';
import { MY_TEAMS_HINT } from './plan-text';
import { usePlanActions } from './usePlanActions';

const SEARCH_DELAY_MS = 250;

// The API refuses more (D-317).
const MAX_PINNED_PLANS = 100;

// Each group keeps the list's own order; a pinned plan the list doesn't show is simply not there.
function pinnedFirst(plans: PlanSummary[], pinnedIds: string[]): PlanSummary[] {
  const pinned = new Set(pinnedIds);
  return [
    ...plans.filter((plan) => pinned.has(plan.id)),
    ...plans.filter((plan) => !pinned.has(plan.id)),
  ];
}

// Every plan as cards or a table; the filters live in the address and are remembered per person.
export function PlansPage() {
  usePageTitle('Plans');
  const user = useCurrentUser();
  const { updatePreferences } = useSession();
  const { show } = useToast();
  const canCreate = useAccess().has('plans.create');
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

  // Saved in the background like the theme; the stored order only says what is pinned (D-317).
  function togglePin(plan: PlanSummary) {
    const pinnedIds = user.preferences.pinnedPlanIds;
    if (pinnedIds.includes(plan.id)) {
      updatePreferences({ pinnedPlanIds: pinnedIds.filter((id) => id !== plan.id) });
      return;
    }
    if (pinnedIds.length >= MAX_PINNED_PLANS) {
      show({ message: `You can pin up to ${MAX_PINNED_PLANS} plans. Unpin one first.` });
      return;
    }
    updatePreferences({ pinnedPlanIds: [...pinnedIds, plan.id] });
  }

  const isPinned = (plan: PlanSummary) => user.preferences.pinnedPlanIds.includes(plan.id);

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

  const chips = filterChips(filters);

  function removeChip(key: string) {
    if (key === 'q') {
      clearSearch();
    }
    change(chipReset(key));
  }

  const data = plans.data;
  const summary = data?.summary;
  const planYears = data?.years ?? [];
  const showsTable = filters.layout === 'table';
  const items = data ? pinnedFirst(data.items, user.preferences.pinnedPlanIds) : [];

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
        actions={<NewPlanButton />}
      />
      <div className={styles.page}>
        <PlanListSummary summary={summary} />

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
              showsTable &&
              data && (
                <PlanTable
                  plans={items}
                  actionsFor={actions.menuItems}
                  isPinned={isPinned}
                  onPinToggle={togglePin}
                />
              )
            )}
          </div>
        </section>

        {!state && !showsTable && data && (
          <div className={styles.cards}>
            {items.map((plan) => (
              <PlanCard
                key={plan.id}
                plan={plan}
                actions={actions.menuItems(plan)}
                pinned={isPinned(plan)}
                onPinToggle={() => togglePin(plan)}
              />
            ))}
          </div>
        )}
      </div>
      {actions.dialogs}
    </>
  );
}

function ListSkeleton() {
  return (
    <div className={styles.skeletonList} aria-label="Loading plans">
      {[0, 1, 2].map((row) => (
        <Skeleton key={row} height="40px" />
      ))}
    </div>
  );
}

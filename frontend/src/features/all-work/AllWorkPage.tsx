import { ClipboardList, SearchX } from 'lucide-react';
import { useEffect, useMemo, useRef, useState } from 'react';
import type { ReactNode } from 'react';
import { ApiError } from '@/api/errors';
import {
  Button,
  Card,
  EmptyState,
  ErrorState,
  LinkButton,
  PageHeader,
  Tabs,
  Skeleton,
  useToast,
} from '@/components';
import type { TabItem } from '@/components';
import { usePageTitle } from '@/lib/usePageTitle';
import {
  useAllWork,
  useAllWorkOptions,
  useAllWorkPreferences,
  usePreferencesSaver,
} from './allWorkApi';
import styles from './AllWorkPage.module.css';
import { ALL_WORK_DESCRIPTION, VIEW_LABELS } from './allWorkText';
import type { AllWorkResult, Sort, ViewState, ViewType } from './allWorkTypes';
import { AllWorkGrid } from './AllWorkGrid';
import { BudgetView } from './BudgetView';
import { CalendarView } from './CalendarView';
import { ColumnChooser } from './ColumnChooser';
import { ExportMenu } from './ExportMenu';
import { FilterBar } from './FilterBar';
import { MonthCalendar } from './MonthCalendar';
import { Pager } from './Pager';
import { SummaryCards } from './SummaryCards';
import { useAllWorkSession } from './useAllWorkSession';
import { useOpenRecord } from './useOpenRecord';
import { ViewsControl } from './ViewsControl';
import { ViewToolbar } from './ViewToolbar';
import {
  clearAll,
  continuedPath,
  nextSort,
  pageOf,
  standardState,
  toggleRow,
  toQueryParams,
  visibleRows,
  withoutRefused,
  yearRange,
} from './viewState';

/** The views as pill tabs (FR-001). */
const TABS: TabItem<ViewType>[] = [
  { value: 'grid', label: VIEW_LABELS.grid },
  { value: 'calendar', label: VIEW_LABELS.calendar },
  { value: 'budget', label: VIEW_LABELS.budget },
];

/**
 * All work (feature 010, FR-001): every plan of a year as a read-only Grid, Calendar or Budget &
 * cost view, with the summary cards and one filter bar, under one state that survives moving around
 * NorthStar (FR-009). Nothing is created, changed or deleted here (FR-002); a row opens its
 * record (FR-016).
 */
export function AllWorkPage() {
  usePageTitle('All work');
  const session = useAllWorkSession();
  const { state, entry, setState } = session;
  const query = useAllWork(state ? toQueryParams(state) : null);
  const options = useAllWorkOptions(
    state ? state.year : null,
    state !== null,
    state ? yearRange(state).to : undefined,
  );
  const preferences = useAllWorkPreferences();
  const open = useOpenRecord(session.rememberScroll);
  const savePreferences = usePreferencesSaver();
  const [choosingColumns, setChoosingColumns] = useState(false);
  const { show } = useToast();
  // The previous answer stays while a new one loads, but only for the same view.
  const result = state && query.data?.view === state.view ? query.data : undefined;

  const visible = useMemo(
    () => (result && state ? visibleRows(result.rows, state) : []),
    [result, state],
  );
  const page = pageOf(visible, entry.page, entry.pageSize);
  // The Calendar's Month layout shows one month of activities; it has no pages.
  const monthLayout = state?.view === 'calendar' && state.calendarLayout === 'month';

  // Back from a record: the page scrolls to where it was (FR-016), once the rows are there.
  const scrollRestored = useRef(false);
  useEffect(() => {
    if (result && !scrollRestored.current) {
      scrollRestored.current = true;
      if (entry.scrollY > 0) {
        window.scrollTo(0, entry.scrollY);
      }
    }
  }, [result, entry.scrollY]);

  // A refused filter is named in a message and set back (plan "States").
  const refused = query.error instanceof ApiError && query.error.kind === 'validation';
  useEffect(() => {
    if (!refused || !state || !preferences.data || !options.data) {
      return;
    }
    const error = query.error as ApiError;
    show({ message: error.message });
    const standard = standardState(preferences.data, options.data.currentYear);
    const fields = error.fields.map((field) => field.field);
    setState((current) => withoutRefused(current, fields, standard));
  }, [refused, query.error, state, preferences.data, options.data, setState, show]);

  function changeExpansion(next: ViewState) {
    // Opening and closing rows keeps the page (it may become shorter).
    setState(next, { keepPage: true });
  }

  /**
   * Shows other columns or another sort (FR-015, FR-018). They are remembered for the person
   * while no saved view is applied; with one, they only make it "Modified" (FR-021, FR-030).
   */
  function changeColumns(columns: string[], sort: Sort | null) {
    setState((current) => ({ ...current, columns, sort }), { keepPage: true });
    if (!entry.appliedViewId) {
      savePreferences({ columns, sort });
    }
  }

  const view = state && (
    <>
      <FilterBar
        state={state}
        options={options.data}
        count={query.data?.count}
        updating={query.isFetching && query.data !== undefined}
        onChange={(change) => setState(change)}
      />
      {/* The cards and the count are the same in every view: the last answer serves. */}
      <SummaryCards summary={query.data?.summary ?? null} />
      <div aria-busy={query.isFetching} className={styles.card}>
        <ViewToolbar
          state={state}
          generatedAt={result?.generatedAt}
          onChange={changeExpansion}
          onColumns={() => setChoosingColumns(true)}
          onRefresh={() => void query.refetch()}
        />
        <ViewContent
          state={state}
          result={result}
          error={query.isError ? query.error : null}
          hasPlans={options.data ? options.data.plans.length > 0 : true}
          onRetry={() => void query.refetch()}
          onClearFilters={() => setState(clearAll)}
        >
          {result && monthLayout && (
            <MonthCalendar key={result.year} result={result} onOpen={(row) => void open(row)} />
          )}
          {result && state.view === 'calendar' && !monthLayout && (
            <CalendarView
              result={result}
              rows={page.rows}
              continued={continuedPath(page.rows, result.rows)}
              search={state.search}
              onToggle={(key) => changeExpansion(toggleRow(state, key))}
              onOpen={(row) => void open(row)}
            />
          )}
          {result && state.view === 'budget' && (
            <BudgetView
              result={result}
              rows={page.rows}
              continued={continuedPath(page.rows, result.rows)}
              search={state.search}
              onToggle={(key) => changeExpansion(toggleRow(state, key))}
              onOpen={(row) => void open(row)}
            />
          )}
          {result && state.view === 'grid' && (
            <AllWorkGrid
              result={result}
              rows={page.rows}
              continued={continuedPath(page.rows, result.rows)}
              search={state.search}
              sort={state.sort}
              onSort={(column) => changeColumns(state.columns, nextSort(state.sort, column))}
              onToggle={(key) => changeExpansion(toggleRow(state, key))}
              onOpen={(row) => void open(row)}
            />
          )}
        </ViewContent>
      </div>
      {result && result.rows.length > 0 && !monthLayout && (
        <Pager
          page={page}
          pageSize={entry.pageSize}
          onPage={session.setPage}
          onPageSize={session.setPageSize}
        />
      )}
    </>
  );

  return (
    <>
      <PageHeader
        title="All work"
        description={ALL_WORK_DESCRIPTION}
        actions={
          <>
            <ViewsControl
              state={state}
              appliedViewId={entry.appliedViewId}
              onApply={(saved) => void session.applyView(saved)}
              onSaved={(saved) => session.setAppliedView(saved.id)}
              onStandard={session.applyStandard}
            />
            {state && <ExportMenu state={state} result={result} />}
          </>
        }
      />
      <div className={styles.page}>
        {session.startError ? (
          <Card>
            <ErrorState error={session.startError} onRetry={session.retryStart} />
          </Card>
        ) : state ? (
          // The view switch sits right under the header, the filters after it (UI audit AW-03).
          <Tabs
            label="All work views"
            tabs={TABS}
            value={state.view}
            onChange={(next) => setState({ ...state, view: next }, { keepPage: true })}
          >
            <div className={styles.view}>{view}</div>
          </Tabs>
        ) : (
          <>
            <SummaryCards summary={null} />
            <div className={styles.card}>
              <GridSkeleton />
            </div>
          </>
        )}
      </div>
      {state && options.data && (
        <ColumnChooser
          open={choosingColumns}
          onOpenChange={setChoosingColumns}
          groups={options.data.columnGroups}
          columns={state.columns}
          defaultColumns={options.data.defaultColumns}
          onChange={(columns) => changeColumns(columns, state.sort)}
        />
      )}
    </>
  );
}

interface ViewContentProps {
  state: ViewState;
  result: AllWorkResult | undefined;
  error: unknown;
  /** Whether the year has plans at all (the filter bar's choices). */
  hasPlans: boolean;
  onRetry: () => void;
  onClearFilters: () => void;
  /** The view itself, shown when there are rows. */
  children: ReactNode;
}

/** The states of a view (plan "States"): loading, error, no plans, nothing matches, the rows. */
function ViewContent({
  state,
  result,
  error,
  hasPlans,
  onRetry,
  onClearFilters,
  children,
}: ViewContentProps) {
  if (!result) {
    return error && !(error instanceof ApiError && error.kind === 'validation') ? (
      <div className={styles.state}>
        <ErrorState error={error} onRetry={onRetry} />
      </div>
    ) : (
      <GridSkeleton />
    );
  }
  if (result.rows.length === 0) {
    return (
      <div className={styles.state}>
        {!hasPlans ? (
          <EmptyState
            icon={<ClipboardList />}
            title={`No plans in ${state.year} yet`}
            text={`Plans of ${state.year} appear here once they're created.`}
            action={<LinkButton to="/plans">Go to Plans</LinkButton>}
          />
        ) : (
          <EmptyState
            icon={<SearchX />}
            title="Nothing matches these filters"
            text="Try removing a filter or changing the search."
            action={<Button onClick={onClearFilters}>Clear filters</Button>}
          />
        )}
      </div>
    );
  }
  return <>{children}</>;
}

/** Eight grey lines while the first answer loads. */
function GridSkeleton() {
  return (
    <div className={styles.state}>
      <Skeleton lines={8} height="18px" />
    </div>
  );
}

import { useEffect, useMemo, useRef, useState } from 'react';
import { ApiError } from '@/api/errors';
import { Card } from '@/ui/Card/Card';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { PageHeader } from '@/ui/PageHeader/PageHeader';
import { Tabs } from '@/ui/Tabs/Tabs';
import { useToast } from '@/ui/Toast/useToast';
import type { TabItem } from '@/ui/Tabs/Tabs';
import { usePageTitle } from '@/ui/usePageTitle';
import { BulkBar, useSelection } from '@/features/bulk';
import { useAccess } from '@/session/access';
import { useAllWork, useAllWorkOptions, useAllWorkPreferences, usePreferencesSaver } from './api';
import styles from './AllWorkPage.module.css';
import { ALL_WORK_DESCRIPTION, VIEW_LABELS } from './all-work-text';
import type { Sort, ViewState, ViewType } from './all-work-types';
import { AllWorkGrid } from './AllWorkGrid';
import { BudgetView } from './BudgetView';
import { CalendarView } from './CalendarView';
import { ColumnChooser } from './ColumnChooser';
import { ExportMenu } from './ExportMenu';
import { FilterBar } from './FilterBar';
import { GridSkeleton } from './GridSkeleton';
import { MonthCalendar } from './MonthCalendar';
import { Pager } from './Pager';
import { SummaryCards } from './SummaryCards';
import { useAllWorkSession } from './useAllWorkSession';
import { useOpenRecord } from './useOpenRecord';
import { ViewsControl } from './ViewsControl';
import { ViewContent } from './ViewContent';
import { ViewToolbar } from './ViewToolbar';
import { clearAll } from './view-chips';
import { continuedPath, pageOf, toggleRow, visibleRows } from './view-rows';
import { nextSort } from './view-columns';
import { standardState, toQueryParams, withoutRefused, yearRange } from './view-state';
import { gridSelection, matchingRecords } from './grid-selection';

const TABS: TabItem<ViewType>[] = [
  { value: 'grid', label: VIEW_LABELS.grid },
  { value: 'calendar', label: VIEW_LABELS.calendar },
  { value: 'budget', label: VIEW_LABELS.budget },
];

/**
 * All work (feature 010): every plan of a year as a Grid, Calendar or Budget & cost view under one state
 * that survives moving around (FR-009); the Grid's rows are ticked for bulk actions (feature 012).
 */
export function AllWorkPage() {
  const { has } = useAccess();
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
  const selection = useSelection();
  // The previous answer stays while a new one loads, but only for the same view.
  const result = state && query.data?.view === state.view ? query.data : undefined;

  const visible = useMemo(
    () => (result && state ? visibleRows(result.rows, state) : []),
    [result, state],
  );
  const page = pageOf(visible, entry.page, entry.pageSize);
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

  const refusal =
    query.error instanceof ApiError && query.error.kind === 'validation' ? query.error : null;
  useEffect(() => {
    if (!refusal || !state || !preferences.data || !options.data) {
      return;
    }
    show({ message: refusal.message });
    const standard = standardState(preferences.data, options.data.currentYear);
    const fields = refusal.fields.map((field) => field.field);
    setState((current) => withoutRefused(current, fields, standard));
  }, [refusal, state, preferences.data, options.data, setState, show]);

  function clearSelection() {
    if (selection.records.length > 0) {
      selection.clear();
      show({ message: 'Selection cleared because the filters changed' });
    }
  }

  function changeExpansion(next: ViewState) {
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
        onChange={(change) => {
          clearSelection();
          setState(change);
        }}
      />
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
          onClearFilters={() => {
            clearSelection();
            setState(clearAll);
          }}
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
              selection={gridSelection(
                selection,
                page.rows.map((visible) => visible.row),
              )}
            />
          )}
        </ViewContent>
      </div>
      {result && state.view === 'grid' && (
        <BulkBar selection={selection} matching={matchingRecords(result.rows)} />
      )}
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
            {/* Export needs data.export (feature 014, FR-012); the server checks it too. */}
            {state && has('data.export') && <ExportMenu state={state} result={result} />}
          </>
        }
      />
      <div className={styles.page}>
        {session.startError ? (
          <Card>
            <ErrorState error={session.startError} onRetry={session.retryStart} />
          </Card>
        ) : state ? (
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

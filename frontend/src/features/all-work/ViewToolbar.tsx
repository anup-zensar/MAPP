import { ChevronsDownUp, ChevronsUpDown, Columns3, RotateCw } from 'lucide-react';
import { Button, SegmentedFilter } from '@/components';
import { formatTime } from '@/lib/format';
import type { ViewState } from './allWorkTypes';
import { Switch } from './Switch';
import styles from './ViewToolbar.module.css';

export interface ViewToolbarProps {
  state: ViewState;
  /** When the data on screen was read ("Updated at 14:05"); undefined while loading. */
  generatedAt: string | undefined;
  /** Changes how the rows are shown (expansion, unassigned rows, period, layout). */
  onChange: (next: ViewState) => void;
  onColumns: () => void;
  onRefresh: () => void;
}

/**
 * The toolbar at the top of a view's card (plan "Page layout"), in one order in every view: the
 * view's layout (Calendar: Timeline · Month; Budget & cost: Monthly · Quarterly), Expand all and
 * Collapse all, the Unassigned planned cost switch (FR-012); then, at the end, Columns (Grid),
 * "Updated at …" and Refresh. Below 600 px the buttons show their icons only (their names stay),
 * so the toolbar takes two rows and the data starts higher (UI audit PH-04).
 */
export function ViewToolbar({
  state,
  generatedAt,
  onChange,
  onColumns,
  onRefresh,
}: ViewToolbarProps) {
  const tree = state.view !== 'calendar';

  return (
    <div className={styles.toolbar}>
      <div className={styles.group}>
        {state.view === 'calendar' && (
          <SegmentedFilter
            label="Calendar layout"
            options={[
              { value: 'timeline', label: 'Timeline' },
              { value: 'month', label: 'Month' },
            ]}
            value={state.calendarLayout}
            onChange={(calendarLayout) => onChange({ ...state, calendarLayout })}
          />
        )}
        {state.view === 'budget' && (
          <SegmentedFilter
            label="Period"
            options={[
              { value: 'month', label: 'Monthly' },
              { value: 'quarter', label: 'Quarterly' },
            ]}
            value={state.period}
            onChange={(period) => onChange({ ...state, period })}
          />
        )}
        {tree && (
          <>
            <Button
              size="sm"
              icon={<ChevronsUpDown />}
              onClick={() =>
                onChange({ ...state, expansion: { mode: 'allExpanded', toggled: [] } })
              }
            >
              <span className={styles.label}>Expand all</span>
            </Button>
            <Button
              size="sm"
              icon={<ChevronsDownUp />}
              onClick={() =>
                onChange({ ...state, expansion: { mode: 'allCollapsed', toggled: [] } })
              }
            >
              <span className={styles.label}>Collapse all</span>
            </Button>
            <Switch
              label="Unassigned planned cost"
              checked={state.showUnassigned}
              onChange={(showUnassigned) => onChange({ ...state, showUnassigned })}
            />
          </>
        )}
      </div>
      <div className={styles.end}>
        {state.view === 'grid' && (
          <Button size="sm" icon={<Columns3 />} onClick={onColumns}>
            <span className={styles.label}>Columns</span>
          </Button>
        )}
        {generatedAt && (
          <span className={styles.updated}>Updated at {formatTime(generatedAt)}</span>
        )}
        <Button size="sm" icon={<RotateCw />} onClick={onRefresh}>
          <span className={styles.label}>Refresh</span>
        </Button>
      </div>
    </div>
  );
}

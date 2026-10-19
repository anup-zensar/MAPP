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
 * The toolbar above a view (plan "Page layout"): Expand all and Collapse all, the Unassigned
 * planned cost switch (FR-012), Columns (Grid), Monthly · Quarterly (Budget & cost, FR-022) or
 * Timeline · Month (Calendar, FR-028), and Refresh with "Updated at …".
 */
export function ViewToolbar({
  state,
  generatedAt,
  onChange,
  onColumns,
  onRefresh,
}: ViewToolbarProps) {
  return (
    <div className={styles.toolbar}>
      {state.view === 'calendar' ? (
        <SegmentedFilter
          label="Calendar layout"
          options={[
            { value: 'timeline', label: 'Timeline' },
            { value: 'month', label: 'Month' },
          ]}
          value={state.calendarLayout}
          onChange={(calendarLayout) => onChange({ ...state, calendarLayout })}
        />
      ) : (
        <>
          <Button
            size="sm"
            icon={<ChevronsUpDown />}
            onClick={() => onChange({ ...state, expansion: { mode: 'allExpanded', toggled: [] } })}
          >
            Expand all
          </Button>
          <Button
            size="sm"
            icon={<ChevronsDownUp />}
            onClick={() => onChange({ ...state, expansion: { mode: 'allCollapsed', toggled: [] } })}
          >
            Collapse all
          </Button>
          <Switch
            label="Unassigned planned cost"
            checked={state.showUnassigned}
            onChange={(showUnassigned) => onChange({ ...state, showUnassigned })}
          />
        </>
      )}
      {state.view === 'grid' && (
        <Button size="sm" icon={<Columns3 />} onClick={onColumns}>
          Columns
        </Button>
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
      <span className={styles.refresh}>
        {generatedAt && (
          <span className={styles.updated}>Updated at {formatTime(generatedAt)}</span>
        )}
        <Button size="sm" icon={<RotateCw />} onClick={onRefresh}>
          Refresh
        </Button>
      </span>
    </div>
  );
}

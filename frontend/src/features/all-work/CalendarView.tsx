import { useId } from 'react';
import { LevelTag } from '@/ui/LevelTag/LevelTag';
import { StatusChip } from '@/ui/StatusChip/StatusChip';
import { cx } from '@/ui/cx';
import gridStyles from './AllWorkGrid.module.css';
import type { AllWorkResult, AllWorkRow } from './all-work-types';
import { hasNoDates, todayOffset } from './calendar';
import { CalendarRow } from './CalendarRow';
import styles from './CalendarView.module.css';
import { RecordLink } from './RecordLink';
import { ScrollBox } from './ScrollBox';
import { statusWord } from './status-words';
import { legendOf, timelineOf } from './timeline';
import type { VisibleRow } from './view-rows';

/**
 * The narrowest one year of the timeline gets (CalendarView.module.css); a bar too short for its
 * words at this width writes them beside it instead of cutting them (UI audit AW-06).
 */
const YEAR_MIN_PX = 720;
/** In a range, each year gets at least this much: four quarters of 80 px. */
const RANGE_YEAR_MIN_PX = 320;

export interface CalendarViewProps {
  /** The answer for view=calendar (no line-item or unassigned rows). */
  result: AllWorkResult;
  /** The rows of the page on screen (D-115). */
  rows: VisibleRow[];
  continued: string;
  search: string;
  onToggle: (key: string) => void;
  onOpen: (row: AllWorkRow) => void;
}

/**
 * The Calendar's timeline (US5, FR-025 – FR-027): the year with its months across the top — or,
 * for a range of years, the years with their quarters (016 FR-036) — a bar per campaign,
 * initiative and activity from its start to its end date in its status colour with the status
 * word, the Today line, cut and open ends said in words, and "No dates yet (N)". Bars open their
 * record and cannot be dragged: dates change on the record's page (FR-027). The legend shows each
 * status word once, in its one colour (UI audit X-14).
 */
export function CalendarView({
  result,
  rows,
  continued,
  search,
  onToggle,
  onOpen,
}: CalendarViewProps) {
  const headingId = useId();
  const { year, today } = result;
  const yearTo = Math.max(result.yearTo ?? year, year);
  const timeline = timelineOf(year, yearTo);
  const todayAt = todayOffset(today, year, yearTo);
  const undated = result.rows.filter(hasNoDates);
  const statuses = legendOf(result.rows);
  const caption = yearTo > year ? `Calendar ${year}–${yearTo}` : `Calendar ${year}`;
  const trackPx =
    yearTo > year ? Math.max(YEAR_MIN_PX, RANGE_YEAR_MIN_PX * (yearTo - year + 1)) : YEAR_MIN_PX;

  return (
    <div className={gridStyles.grid}>
      {statuses.length > 0 && (
        <div className={styles.legend}>
          <span className={styles.legendTitle}>Status</span>
          <ul className={styles.legendList}>
            {statuses.map((status) => (
              <li key={status.label}>
                <StatusChip tone={status.tone}>{status.label}</StatusChip>
              </li>
            ))}
          </ul>
        </div>
      )}
      {continued && <p className={gridStyles.continued}>Continued: {continued}</p>}
      <ScrollBox label={caption}>
        <table
          className={cx(gridStyles.table, styles.calendar)}
          style={{ '--track-min': `${trackPx}px` }}
        >
          <caption className="visually-hidden">{caption}</caption>
          <thead>
            <tr>
              <th scope="col" className={gridStyles.nameHead}>
                Name
              </th>
              <th scope="col" className={styles.trackHead}>
                <div className={styles.track}>
                  {timeline.years && (
                    <div className={styles.years} style={{ gridTemplateColumns: timeline.years }}>
                      {Array.from({ length: yearTo - year + 1 }, (_, index) => (
                        <span key={index}>{year + index}</span>
                      ))}
                    </div>
                  )}
                  <div className={styles.months} style={{ gridTemplateColumns: timeline.columns }}>
                    {timeline.labels.map((label, index) => (
                      <span key={index}>{label}</span>
                    ))}
                  </div>
                  {todayAt !== null && (
                    <span className={styles.todayLabel} style={{ left: `${todayAt}%` }}>
                      Today
                    </span>
                  )}
                </div>
              </th>
            </tr>
          </thead>
          <tbody>
            {rows.map((visible) => (
              <CalendarRow
                key={visible.row.key}
                visible={visible}
                year={year}
                yearTo={yearTo}
                columns={timeline.columns}
                count={timeline.labels.length}
                todayAt={todayAt}
                trackPx={trackPx}
                search={search}
                onToggle={onToggle}
                onOpen={onOpen}
              />
            ))}
          </tbody>
        </table>
      </ScrollBox>
      {undated.length > 0 && (
        <section className={styles.undated} aria-labelledby={headingId}>
          <h3 id={headingId} className={styles.undatedTitle}>
            No dates yet ({undated.length})
          </h3>
          <ul className={styles.undatedList}>
            {undated.map((row) => (
              <li key={row.key} className={styles.undatedItem}>
                {row.level !== 'unassigned' && <LevelTag level={row.level} />}
                <RecordLink row={row} onOpen={onOpen} className={styles.undatedLink}>
                  {row.name}
                </RecordLink>
                {row.status && (
                  <StatusChip tone={row.status.tone}>{statusWord(row.status.label)}</StatusChip>
                )}
              </li>
            ))}
          </ul>
        </section>
      )}
    </div>
  );
}

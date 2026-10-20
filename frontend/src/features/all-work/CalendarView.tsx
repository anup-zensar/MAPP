import { useId } from 'react';
import type { ComponentProps, MouseEvent } from 'react';
import { Link } from 'react-router';
import { LevelTag, StatusChip, Tooltip } from '@/components';
import { cx } from '@/lib/cx';
import { useFormatters } from '@/lib/useFormatters';
import gridStyles from './AllWorkGrid.module.css';
import type { AllWorkResult, AllWorkRow, AllWorkStatus } from './allWorkTypes';
import { barOf, hasNoDates, monthDays, quarterDays, todayOffset, yearDays } from './calendar';
import type { Bar } from './calendar';
import styles from './CalendarView.module.css';
import { isControl } from './cellText';
import { RowName } from './RowName';
import { rowHref } from './rowHref';
import { ScrollBox } from './ScrollBox';
import { statusWord } from './statusWords';
import type { VisibleRow } from './viewState';

const MONTHS = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];
const QUARTERS = ['Q1', 'Q2', 'Q3', 'Q4'];

/**
 * The narrowest one year of the timeline gets (CalendarView.module.css); a bar too short for its
 * words at this width writes them beside it instead of cutting them (UI audit AW-06).
 */
const YEAR_MIN_PX = 720;
/** In a range, each year gets at least this much: four quarters of 80 px. */
const RANGE_YEAR_MIN_PX = 320;
/** About how wide one character of a bar's 12 px words is, and the bar's padding and gaps. */
const CHAR_PX = 7;
const BAR_PADDING_PX = 32;

/** The level of a bar, as a word in its tip. */
const LEVEL_WORDS: Partial<Record<AllWorkRow['level'], string>> = {
  campaign: 'Campaign',
  initiative: 'Initiative',
  activity: 'Activity',
};

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
          style={{ ['--track-min' as string]: `${trackPx}px` }}
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
                  <StatusChip tone={row.status.tone}>
                    {statusWord(row.status.label, row.status.tone)}
                  </StatusChip>
                )}
              </li>
            ))}
          </ul>
        </section>
      )}
    </div>
  );
}

interface CalendarRowProps {
  visible: VisibleRow;
  year: number;
  /** The last year of the timeline (the same as year for one year). */
  yearTo: number;
  /** The month or quarter columns, as wide as their days. */
  columns: string;
  /** How many month or quarter columns there are. */
  count: number;
  todayAt: number | null;
  /** The narrowest the whole timeline gets, in pixels. */
  trackPx: number;
  search: string;
  onToggle: (key: string) => void;
  onOpen: (row: AllWorkRow) => void;
}

function CalendarRow({
  visible,
  year,
  yearTo,
  columns,
  count,
  todayAt,
  trackPx,
  search,
  onToggle,
  onOpen,
}: CalendarRowProps) {
  const { row } = visible;
  const bar = barOf(row, year, yearTo);

  function openFromRow(event: MouseEvent<HTMLTableRowElement>) {
    if (!isControl(event.target)) {
      onOpen(row);
    }
  }

  return (
    <tr
      data-row-key={row.key}
      className={cx(gridStyles.row, row.role === 'context' && gridStyles.context)}
      onClick={openFromRow}
    >
      <th scope="row" className={cx(gridStyles.nameCell, gridStyles[row.level])}>
        {/* The bar says the status; only a row without a bar shows its chip (UI audit AW-06). */}
        <RowName
          visible={visible}
          search={search}
          showStatus={!bar}
          onToggle={onToggle}
          onOpen={onOpen}
        />
      </th>
      <td className={styles.trackCell}>
        <div className={styles.track}>
          <div className={styles.lines} style={{ gridTemplateColumns: columns }} aria-hidden="true">
            {Array.from({ length: count }, (_, index) => (
              <span key={index} />
            ))}
          </div>
          {todayAt !== null && (
            <span className={styles.today} style={{ left: `${todayAt}%` }} aria-hidden="true" />
          )}
          {bar && (
            <BarLink
              row={row}
              bar={bar}
              year={year}
              yearTo={yearTo}
              trackPx={trackPx}
              onOpen={onOpen}
            />
          )}
        </div>
      </td>
    </tr>
  );
}

interface BarLinkProps {
  row: AllWorkRow;
  bar: Bar;
  year: number;
  yearTo: number;
  /** The narrowest the whole timeline gets, in pixels. */
  trackPx: number;
  onOpen: (row: AllWorkRow) => void;
}

/**
 * A bar: a link to the record with its name, status and dates in a tip (FR-026, FR-027). Its
 * words sit on the bar when they fit, else beside it — after it, or before it near the end of the
 * timeline — so they are never cut while the row has room (UI audit AW-06).
 */
function BarLink({ row, bar, year, yearTo, trackPx, onOpen }: BarLinkProps) {
  const { date, dateRange, money } = useFormatters();
  const tone = row.status?.tone ?? 'neutral';
  const word = row.status ? statusWord(row.status.label, row.status.tone) : '';
  const from = bar.continuesFrom ? `Continues from ${year - 1}` : '';
  const into = bar.continuesInto ? `Continues into ${yearTo + 1}` : '';
  const ongoing = bar.ongoing ? 'Ongoing' : '';
  const characters = [from, row.name, word, ongoing, into].join('').length;
  const fits = characters * CHAR_PX + BAR_PADDING_PX <= (bar.width / 100) * trackPx;
  const beside = fits ? null : bar.left + bar.width > 65 ? 'before' : 'after';

  const tip = (
    <span className={styles.tip}>
      <span className={styles.tipTitle}>
        {row.name} · {row.recordId}
      </span>
      <span>
        {LEVEL_WORDS[row.level]}
        {word && ` · ${word}`}
      </span>
      <span>
        {bar.ongoing && row.startDate
          ? `From ${date(row.startDate)}, ongoing`
          : dateRange(row.startDate, row.endDate)}
      </span>
      {row.owner && (
        <span>
          Owner: {row.owner.name}
          {row.owner.inactive && ' (inactive)'}
        </span>
      )}
      {row.plannedCost && <span>Planned cost: {money(row.plannedCost)}</span>}
      <span>Change dates on the record&apos;s page</span>
    </span>
  );

  return (
    <Tooltip content={tip}>
      <RecordLink
        row={row}
        onOpen={onOpen}
        className={cx(
          styles.bar,
          styles[tone],
          bar.continuesFrom && styles.cutStart,
          (bar.continuesInto || bar.ongoing) && styles.openEnd,
          beside === 'after' && styles.wordsAfter,
          beside === 'before' && styles.wordsBefore,
        )}
        style={{ left: `${bar.left}%`, width: `${bar.width}%` }}
      >
        <span className={styles.words}>
          {from && <span className={styles.edge}>{from}</span>}
          <span className={styles.barName}>{row.name}</span>
          {word && <span className={styles.barStatus}>{word}</span>}
          {ongoing && <span className={styles.edge}>{ongoing}</span>}
          {into && <span className={styles.edge}>{into}</span>}
        </span>
      </RecordLink>
    </Tooltip>
  );
}

type RecordLinkProps = Omit<ComponentProps<typeof Link>, 'to' | 'onClick' | 'draggable'> & {
  row: AllWorkRow;
  onOpen: (row: AllWorkRow) => void;
};

/**
 * A link to a row's record that first checks it still exists (onOpen); never draggable, so a bar
 * cannot be dragged to other dates (FR-027).
 */
function RecordLink({ row, onOpen, ...linkProps }: RecordLinkProps) {
  function open(event: MouseEvent<HTMLAnchorElement>) {
    if (event.button !== 0 || event.ctrlKey || event.metaKey || event.shiftKey || event.altKey) {
      return;
    }
    event.preventDefault();
    onOpen(row);
  }
  return (
    // The other props include those of a Tooltip around the link (its ref and handlers).
    <Link {...linkProps} to={rowHref(row)} draggable={false} onClick={open} />
  );
}

/** The statuses the bars show, each word once, for the legend (FR-025, UI audit X-14). */
function legendOf(rows: AllWorkRow[]): AllWorkStatus[] {
  const seen = new Map<string, AllWorkStatus>();
  for (const row of rows) {
    if (row.status && LEVEL_WORDS[row.level]) {
      const label = statusWord(row.status.label, row.status.tone);
      if (!seen.has(label)) {
        seen.set(label, { ...row.status, label });
      }
    }
  }
  return [...seen.values()];
}

/** The columns of the timeline: months for one year; the years with their quarters for a range. */
interface Timeline {
  /** The CSS grid columns of the months or quarters, each as wide as its days. */
  columns: string;
  labels: string[];
  /** The CSS grid columns of the year band (a range only). */
  years?: string;
}

function timelineOf(year: number, yearTo: number): Timeline {
  const widths = (days: number[]) => days.map((count) => `minmax(0, ${count}fr)`).join(' ');
  if (yearTo === year) {
    return { columns: widths(monthDays(year)), labels: MONTHS };
  }
  const span = Array.from({ length: yearTo - year + 1 }, (_, index) => year + index);
  return {
    columns: widths(span.flatMap(quarterDays)),
    labels: span.flatMap(() => QUARTERS),
    years: widths(yearDays(year, yearTo)),
  };
}

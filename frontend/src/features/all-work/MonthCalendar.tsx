import { ChevronLeft, ChevronRight } from 'lucide-react';
import { Popover } from 'radix-ui';
import { useState } from 'react';
import type { MouseEvent } from 'react';
import { Link } from 'react-router';
import { Button } from '@/components';
import { cx } from '@/lib/cx';
import { useFormatters } from '@/lib/useFormatters';
import type { AllWorkResult, AllWorkRow } from './allWorkTypes';
import { monthWeeks, runsOn } from './calendar';
import styles from './MonthCalendar.module.css';
import { rowHref } from './rowHref';

const MONTH_NAMES = [
  'January',
  'February',
  'March',
  'April',
  'May',
  'June',
  'July',
  'August',
  'September',
  'October',
  'November',
  'December',
];
const WEEKDAYS = ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun'];

/** A day lists this many activities; the rest are behind "+ n more". */
const SHOWN_PER_DAY = 3;

export interface MonthCalendarProps {
  /** The answer for view=calendar; its activities are shown on the days they run. */
  result: AllWorkResult;
  onOpen: (row: AllWorkRow) => void;
}

/**
 * The Calendar's Month layout (FR-028, P3): one month of the year as a Monday-to-Sunday grid, with
 * Previous month, Next month and Today, and each activity on the days it runs; a busy day shows
 * three and "+ n more" with the whole list.
 */
export function MonthCalendar({ result, onOpen }: MonthCalendarProps) {
  const { year, today } = result;
  const todayInYear = today.startsWith(`${year}-`);
  const todayMonth = todayInYear ? Number(today.slice(5, 7)) : 1;
  const [month, setMonth] = useState(todayMonth);
  const title = `${MONTH_NAMES[month - 1]} ${year}`;
  const activities = result.rows.filter((row) => row.level === 'activity');

  return (
    <div className={styles.month}>
      <div className={styles.header}>
        <h3 className={styles.title} aria-live="polite">
          {title}
        </h3>
        <div className={styles.buttons}>
          <Button
            size="sm"
            icon={<ChevronLeft />}
            aria-label="Previous month"
            disabled={month === 1}
            onClick={() => setMonth(month - 1)}
          />
          {todayInYear && (
            <Button size="sm" onClick={() => setMonth(todayMonth)}>
              Today
            </Button>
          )}
          <Button
            size="sm"
            icon={<ChevronRight />}
            aria-label="Next month"
            disabled={month === 12}
            onClick={() => setMonth(month + 1)}
          />
        </div>
      </div>
      <div className={styles.scroll}>
        <table className={styles.grid}>
          <caption className="visually-hidden">{title}</caption>
          <thead>
            <tr>
              {WEEKDAYS.map((day) => (
                <th key={day} scope="col">
                  {day}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {monthWeeks(year, month).map((week) => (
              <tr key={week.find(Boolean)}>
                {week.map((date, index) => (
                  <td key={date ?? `empty-${index}`} className={cx(!date && styles.outside)}>
                    {date && (
                      <Day
                        date={date}
                        isToday={date === today}
                        activities={activities.filter((row) => runsOn(row, date, year))}
                        onOpen={onOpen}
                      />
                    )}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

interface DayProps {
  date: string;
  isToday: boolean;
  activities: AllWorkRow[];
  onOpen: (row: AllWorkRow) => void;
}

function Day({ date, isToday, activities, onOpen }: DayProps) {
  const formatters = useFormatters();
  const more = activities.length - SHOWN_PER_DAY;
  return (
    <div className={styles.day} aria-current={isToday ? 'date' : undefined}>
      <span className={cx(styles.number, isToday && styles.today)}>{Number(date.slice(8))}</span>
      <ul className={styles.items}>
        {activities.slice(0, SHOWN_PER_DAY).map((row) => (
          <li key={row.key}>
            <ActivityLink row={row} onOpen={onOpen} />
          </li>
        ))}
      </ul>
      {more > 0 && (
        <Popover.Root>
          <Popover.Trigger asChild>
            <button type="button" className={styles.more}>
              + {more} more <span className="visually-hidden">on {formatters.date(date)}</span>
            </button>
          </Popover.Trigger>
          <Popover.Portal>
            <Popover.Content
              className={styles.dayList}
              align="start"
              sideOffset={4}
              aria-label={formatters.date(date)}
            >
              <p className={styles.dayTitle}>{formatters.date(date)}</p>
              <ul className={styles.items}>
                {activities.map((row) => (
                  <li key={row.key}>
                    <ActivityLink row={row} onOpen={onOpen} />
                  </li>
                ))}
              </ul>
            </Popover.Content>
          </Popover.Portal>
        </Popover.Root>
      )}
    </div>
  );
}

/** An activity on a day: its name and status word on its status colour; it opens the activity. */
function ActivityLink({ row, onOpen }: { row: AllWorkRow; onOpen: (row: AllWorkRow) => void }) {
  function open(event: MouseEvent<HTMLAnchorElement>) {
    if (event.button !== 0 || event.ctrlKey || event.metaKey || event.shiftKey || event.altKey) {
      return;
    }
    event.preventDefault();
    onOpen(row);
  }
  return (
    <Link
      to={rowHref(row)}
      draggable={false}
      className={cx(styles.item, styles[row.status?.tone ?? 'neutral'])}
      onClick={open}
    >
      <span className={styles.itemName}>{row.name}</span>
      {row.status && <span className={styles.itemStatus}>{row.status.label}</span>}
    </Link>
  );
}

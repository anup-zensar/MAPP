import { Tooltip } from '@/ui/Tooltip/Tooltip';
import { cx } from '@/ui/cx';
import { useFormatters } from '@/format/useFormatters';
import type { AllWorkRow } from './all-work-types';
import type { Bar } from './calendar';
import styles from './CalendarView.module.css';
import { RecordLink } from './RecordLink';
import { statusWord } from './status-words';
import { LEVEL_WORDS } from './timeline';

/** About how wide one character of a bar's 12 px words is, and the bar's padding and gaps. */
const CHAR_PX = 7;
const BAR_PADDING_PX = 32;

interface BarLinkProps {
  row: AllWorkRow;
  bar: Bar;
  year: number;
  yearTo: number;
  /** The narrowest the whole timeline gets, in pixels. */
  trackPx: number;
  onOpen: (row: AllWorkRow) => void;
}

// A bar linking to its record. Its words sit on the bar when they fit, else beside it, so they are
// never cut while the row has room.
export function BarLink({ row, bar, year, yearTo, trackPx, onOpen }: BarLinkProps) {
  const { date, dateRange, money } = useFormatters();
  const tone = row.status?.tone ?? 'neutral';
  const word = row.status ? statusWord(row.status.label) : '';
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

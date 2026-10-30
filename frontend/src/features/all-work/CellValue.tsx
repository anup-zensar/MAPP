import { StatusChip } from '@/ui/StatusChip/StatusChip';
import { Tooltip } from '@/ui/Tooltip/Tooltip';
import { cx } from '@/ui/cx';
import { useFormatters } from '@/format/useFormatters';
import type { AllWorkCell } from './all-work-types';
import { cellText, isProblem, NOTHING } from './cell-text';
import styles from './CellValue.module.css';
import { Highlight } from './Highlight';
import { statusWord } from './status-words';

/** Text longer than this is cut to one line and shown in full in a tip. */
const LONG_TEXT = 40;

export interface CellValueProps {
  /** null: blank, the column belongs to another level (FR-020). */
  cell: AllWorkCell | null | undefined;
  search?: string;
}

// "—" for an empty value; money red with a minus sign when negative; per cent red above 100; long
// text cut with the full text on hover and focus; the Filtered mark of a roll-up filters make smaller.
export function CellValue({ cell, search = '' }: CellValueProps) {
  const formatters = useFormatters();
  if (!cell) {
    return null;
  }
  const value = <Value cell={cell} search={search} />;
  if (!cell.unfiltered) {
    return value;
  }
  return (
    <span className={styles.withMark}>
      {value}
      <Tooltip
        content={`Filters hide some records below. Without filters: ${cellText(cell.unfiltered, formatters)}.`}
      >
        <button type="button" className={styles.filtered}>
          Filtered
        </button>
      </Tooltip>
    </span>
  );
}

function Value({ cell, search }: { cell: AllWorkCell; search: string }) {
  const formatters = useFormatters();
  switch (cell.kind) {
    case 'empty':
      return <span className={styles.empty}>{NOTHING}</span>;
    case 'status':
      return (
        <StatusChip tone={cell.tone ?? 'neutral'}>
          {statusWord(cell.text ?? '')}
          {cell.inactive && ' (inactive)'}
        </StatusChip>
      );
    case 'money':
    case 'percent':
      return (
        <span className={cx(styles.number, isProblem(cell) && styles.problem)}>
          {cellText(cell, formatters)}
        </span>
      );
    case 'number':
      return <span className={styles.number}>{cellText(cell, formatters)}</span>;
    case 'text':
      return <LongText text={cellText(cell, formatters)} search={search} />;
    default:
      return <>{cellText(cell, formatters)}</>;
  }
}

/** Text on one line; when cut, the full text shows on hover and keyboard focus (edge case). */
function LongText({ text, search }: { text: string; search: string }) {
  if (text.length <= LONG_TEXT) {
    return (
      <span className={styles.text}>
        <Highlight text={text} search={search} />
      </span>
    );
  }
  return (
    <Tooltip content={text}>
      {/* eslint-disable-next-line jsx-a11y-x/no-noninteractive-tabindex -- opens the tooltip */}
      <span tabIndex={0} className={cx(styles.text, styles.cut)}>
        <Highlight text={text} search={search} />
      </span>
    </Tooltip>
  );
}

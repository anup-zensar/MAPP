import { formatWhole } from '@/format/decimal';
import type { Formatters } from '@/format/useFormatters';
import type { AllWorkCell, AllWorkColumn } from './allWorkTypes';
import { statusWord } from './statusWords';

/** Shown for an empty value on the column's own level (FR-020). */
export const NOTHING = '—';

/**
 * A cell as plain text, in the person's date format (plan "Grid"): for tips such as the
 * unfiltered value of a Filtered mark. Blank cells (other levels) are "".
 */
export function cellText(
  cell: AllWorkCell | null | undefined,
  formatters: Pick<Formatters, 'money' | 'date' | 'dateTime'>,
): string {
  if (!cell) {
    return '';
  }
  switch (cell.kind) {
    case 'money':
      return formatters.money(cell.money);
    case 'number':
      return cell.number === undefined ? NOTHING : formatWhole(cell.number);
    case 'percent':
      return cell.percent === undefined ? NOTHING : `${cell.percent}%`;
    case 'date':
      return cell.date ? formatters.date(cell.date) : NOTHING;
    case 'dateTime':
      return cell.dateTime ? formatters.dateTime(cell.dateTime) : NOTHING;
    case 'text':
      return `${cell.text ?? NOTHING}${cell.inactive ? ' (inactive)' : ''}`;
    case 'status':
      return `${cell.text ? statusWord(cell.text, cell.tone) : NOTHING}${cell.inactive ? ' (inactive)' : ''}`;
    default:
      return NOTHING;
  }
}

/** Whether a money or percent cell is shown in red (a negative amount, or above 100 %). */
export function isProblem(cell: AllWorkCell): boolean {
  if (cell.tone === 'problem') {
    return true;
  }
  if (cell.kind === 'money') {
    return cell.money?.amount.startsWith('-') ?? false;
  }
  if (cell.kind === 'percent') {
    return Number(cell.percent) > 100;
  }
  return false;
}

/** Whether a column holds numbers, which are right-aligned (design system §4). */
export function isNumeric(column: Pick<AllWorkColumn, 'kind'>): boolean {
  return column.kind === 'money' || column.kind === 'number' || column.kind === 'percent';
}

/** Whether a click landed on a control (link, button, field), which handles it itself. */
export function isControl(target: EventTarget): boolean {
  return (
    target instanceof Element &&
    target.closest('a, button, input, select, textarea, [tabindex]:not(tr)') !== null
  );
}

/** Whether search looks at a column's text: the ID, external names and descriptions (FR-006). */
export function isSearched(columnKey: string): boolean {
  return (
    columnKey === 'id' ||
    columnKey === 'activity.externalName' ||
    columnKey.endsWith('.description')
  );
}

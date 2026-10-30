import { parseMoneyInput } from '@/format/decimal';
import { plural } from '@/format/text';
import { formatMonth } from './line-item-math';

/** Pasted cells, each as an amount when it reads as one; null for a single value (038 FR-042). */
export function pastedValues(text: string): string[] | null {
  const cells = text
    .split(/[\t\r\n]+/)
    .map((cell) => cell.trim())
    .filter((cell) => cell !== '');
  if (cells.length <= 1) {
    return null;
  }
  return cells.map((cell) => parseMoneyInput(cell)?.amount ?? cell);
}

export function leftOutText(count: number, lastMonth: string): string {
  const values = plural(count, 'pasted value');
  const verb = count === 1 ? 'was' : 'were';
  return `${values} ${verb} left out: there is no month after ${formatMonth(lastMonth)}.`;
}

import { parseMoneyInput } from '@/format/decimal';
import { plural } from '@/format/text';
import { formatMonth } from './line-item-math';

/**
 * The cells of a paste from a spreadsheet ("1000⇥2000⇥3000"), each as an amount when it reads as
 * one ("$2,500.50" → "2500.50") and as typed otherwise; null for a single value, which the field
 * takes as usual (038 FR-042).
 */
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

/** "2 pasted values were left out: there is no month after May 2026." */
export function leftOutText(count: number, lastMonth: string): string {
  const values = plural(count, 'pasted value');
  const verb = count === 1 ? 'was' : 'were';
  return `${values} ${verb} left out: there is no month after ${formatMonth(lastMonth)}.`;
}

import type { components } from '@/api/generated/schema';
import { formatCents, toCents } from '@/format/decimal';
import { formatMonth } from './line-item-math';

type SpreadGrid = components['schemas']['SpreadGrid'];
type Money = components['schemas']['Money'];

export interface GridColumn {
  key: string;
  /** "Mar 2026", "Q1 2026" or "No dates yet". */
  label: string;
  outsideActivityDates: boolean;
  noDates: boolean;
}

export interface ShownGrid {
  columns: GridColumn[];
  rows: { lineItemId: string; name: string; cells: (Money | null)[]; total: Money }[];
  totals: (Money | null)[];
  total: Money;
}

export function monthColumns(grid: SpreadGrid): ShownGrid {
  return {
    columns: grid.columns.map((column) => ({
      key: column.key,
      label: column.month ? formatMonth(column.month) : 'No dates yet',
      outsideActivityDates: column.outsideActivityDates,
      noDates: column.month === null,
    })),
    rows: grid.rows,
    totals: grid.totals,
    total: grid.total,
  };
}

/** Display only (§3.8): a quarter adds up its months, outside the dates when all of them are. */
export function quarterColumns(grid: SpreadGrid): ShownGrid {
  const groups: { key: string; label: string; indexes: number[]; noDates: boolean }[] = [];
  grid.columns.forEach((column, index) => {
    const key = column.month ? quarterOf(column.month) : column.key;
    const group = groups.find((candidate) => candidate.key === key);
    if (group) {
      group.indexes.push(index);
    } else {
      groups.push({
        key,
        label: column.month ? key.replace(/^(\d{4})-(Q\d)$/, '$2 $1') : 'No dates yet',
        indexes: [index],
        noDates: column.month === null,
      });
    }
  });
  const currency = grid.total.currency;
  const sum = (cells: (Money | null)[], indexes: number[]): Money | null => {
    const amounts = indexes.flatMap((index) => cells[index] ?? []);
    if (amounts.length === 0) {
      return null;
    }
    const cents = amounts.reduce((total, cell) => total + toCents(cell.amount), 0n);
    return { amount: formatCents(cents), currency };
  };
  return {
    columns: groups.map((group) => ({
      key: group.key,
      label: group.label,
      outsideActivityDates: group.indexes.every(
        (index) => grid.columns[index]?.outsideActivityDates,
      ),
      noDates: group.noDates,
    })),
    rows: grid.rows.map((row) => ({
      ...row,
      cells: groups.map((group) => sum(row.cells, group.indexes)),
    })),
    totals: groups.map((group) => sum(grid.totals, group.indexes)),
    total: grid.total,
  };
}

function quarterOf(month: string): string {
  return `${month.slice(0, 4)}-Q${Math.floor((Number(month.slice(5, 7)) - 1) / 3) + 1}`;
}

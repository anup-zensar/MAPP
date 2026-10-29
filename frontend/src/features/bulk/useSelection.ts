import { useRef, useState } from 'react';
import type { SelectedRecord } from './bulk-types';

export interface Selection {
  records: SelectedRecord[];
  isSelected: (id: string) => boolean;
  toggle: (record: SelectedRecord, range?: SelectedRecord[]) => void;
  select: (records: SelectedRecord[]) => void;
  unselect: (records: SelectedRecord[]) => void;
  replace: (records: SelectedRecord[]) => void;
  clear: () => void;
}

/** The records ticked in one list; toggle with `range` (Shift) ticks every one since the last toggled. */
export function useSelection(): Selection {
  const [selected, setSelected] = useState<Map<string, SelectedRecord>>(() => new Map());
  const anchor = useRef<string | null>(null);

  function select(records: SelectedRecord[]) {
    setSelected((current) => new Map([...current, ...records.map((r) => [r.id, r] as const)]));
  }

  function toggle(record: SelectedRecord, range?: SelectedRecord[]) {
    const from = range && anchor.current ? range.findIndex((r) => r.id === anchor.current) : -1;
    const to = range ? range.findIndex((r) => r.id === record.id) : -1;
    anchor.current = record.id;
    if (range && from >= 0 && to >= 0) {
      select(range.slice(Math.min(from, to), Math.max(from, to) + 1));
      return;
    }
    setSelected((current) => {
      const next = new Map(current);
      if (next.has(record.id)) next.delete(record.id);
      else next.set(record.id, record);
      return next;
    });
  }

  return {
    records: [...selected.values()],
    isSelected: (id) => selected.has(id),
    toggle,
    select,
    unselect: (records) =>
      setSelected((current) => {
        const next = new Map(current);
        records.forEach((r) => next.delete(r.id));
        return next;
      }),
    replace: (records) => setSelected(new Map(records.map((r) => [r.id, r]))),
    clear: () => setSelected(new Map()),
  };
}

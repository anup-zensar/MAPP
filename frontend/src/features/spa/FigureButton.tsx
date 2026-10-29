import { useFormatters } from '@/format/useFormatters';
import { cx } from '@/ui/cx';
import type { Money, SpaContribution } from './api';
import styles from './Spa.module.css';

/** A figure whose contributing activities the trace panel lists (013 FR-007). */
export interface TraceRequest {
  figure: string;
  total: Money;
  items: SpaContribution[];
}

interface FigureButtonProps {
  /** What the figure is, for the panel's title: "Committed", "Q3 2026 planned". */
  figure: string;
  value: Money | null;
  trace: SpaContribution[];
  onTrace: (request: TraceRequest) => void;
  problem?: boolean;
}

/** An amount that opens the activities behind it; "—" when there is none. */
export function FigureButton({
  figure,
  value,
  trace,
  onTrace,
  problem = false,
}: FigureButtonProps) {
  const { money } = useFormatters();
  if (!value) {
    return <span className={styles.muted}>—</span>;
  }
  const shown = money(value);
  return (
    <button
      type="button"
      className={cx(styles.figure, problem && styles.problem)}
      aria-label={`${shown}, show what makes up ${figure}`}
      onClick={() => onTrace({ figure, total: value, items: trace })}
    >
      {shown}
    </button>
  );
}

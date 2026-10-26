import { formatWhole } from '@/format/decimal';
import { cx } from '../cx';
import styles from './SegmentedFilter.module.css';

/** One choice of the filter. */
export interface SegmentedFilterOption<T extends string> {
  value: T;
  label: string;
  /** How many records the choice shows, written after the label ("Active 5"). */
  count?: number;
}

export interface SegmentedFilterProps<T extends string> {
  /** Names the group for screen readers: "Status", "Level", "Layout". */
  label: string;
  options: SegmentedFilterOption<T>[];
  value: T;
  onChange: (value: T) => void;
}

/**
 * Pill toggles for a list's status, level or layout (design system §6.1): one is chosen at a time,
 * told by aria-pressed, never by colour alone.
 */
export function SegmentedFilter<T extends string>({
  label,
  options,
  value,
  onChange,
}: SegmentedFilterProps<T>) {
  return (
    <div role="group" aria-label={label} className={styles.group}>
      {options.map((option) => {
        const pressed = option.value === value;
        return (
          <button
            key={option.value}
            type="button"
            aria-pressed={pressed}
            className={cx(styles.pill, pressed && styles.pressed)}
            onClick={() => onChange(option.value)}
          >
            {option.label}
            {option.count !== undefined && (
              <>
                {' '}
                <span className={styles.count}>{formatWhole(option.count)}</span>
              </>
            )}
          </button>
        );
      })}
    </div>
  );
}

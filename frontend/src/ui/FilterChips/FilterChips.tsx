import { X } from 'lucide-react';
import styles from './FilterChips.module.css';

/** One filter in use: "Status: Draft", "Year: 2026". */
export interface FilterChip {
  key: string;
  label: string;
}

export interface FilterChipsProps {
  chips: FilterChip[];
  onRemove: (key: string) => void;
  /** Adds Clear filters after the chips. */
  onClearAll?: () => void;
}

/**
 * The filters in use as removable chips under a list toolbar (design system §6.1): each one a
 * button "Remove filter: <label>", and Clear filters. Nothing shows when no filter is in use.
 */
export function FilterChips({ chips, onRemove, onClearAll }: FilterChipsProps) {
  if (chips.length === 0) {
    return null;
  }
  return (
    <div className={styles.chips}>
      <ul className={styles.list}>
        {chips.map((chip) => (
          <li key={chip.key}>
            <button
              type="button"
              className={styles.chip}
              aria-label={`Remove filter: ${chip.label}`}
              onClick={() => onRemove(chip.key)}
            >
              {chip.label}
              <X aria-hidden="true" />
            </button>
          </li>
        ))}
      </ul>
      {onClearAll && (
        <button type="button" className={styles.clear} onClick={onClearAll}>
          Clear filters
        </button>
      )}
    </div>
  );
}

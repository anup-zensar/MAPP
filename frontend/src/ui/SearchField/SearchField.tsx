import { Search, X } from 'lucide-react';
import { useId, useRef } from 'react';
import { Tooltip } from '../Tooltip/Tooltip';
import styles from './SearchField.module.css';

export interface SearchFieldProps {
  /** Names the field for screen readers ("Search plans"); not shown on screen. */
  label: string;
  value: string;
  onChange: (value: string) => void;
  /** Says what it searches, on screen: "Search name or ID" (design system §6.1). */
  placeholder?: string;
}

/**
 * The search box of a list toolbar (design system §6.1): a search icon, what it searches, and a
 * Clear search button once something is typed. It only reports the text; the list filters.
 */
export function SearchField({ label, value, onChange, placeholder }: SearchFieldProps) {
  const id = useId();
  const input = useRef<HTMLInputElement>(null);

  function clear() {
    onChange('');
    input.current?.focus();
  }

  return (
    <div className={styles.search}>
      <label htmlFor={id} className="visually-hidden">
        {label}
      </label>
      <Search className={styles.icon} aria-hidden="true" />
      <input
        ref={input}
        id={id}
        type="search"
        value={value}
        placeholder={placeholder}
        onChange={(event) => onChange(event.target.value)}
        className={styles.input}
      />
      {value && (
        <Tooltip content="Clear search">
          <button type="button" className={styles.clear} aria-label="Clear search" onClick={clear}>
            <X />
          </button>
        </Tooltip>
      )}
    </div>
  );
}

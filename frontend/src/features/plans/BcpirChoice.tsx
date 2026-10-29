import type { PlanListFilters } from './list-preferences';
import styles from './PlansFilters.module.css';

const CHOICES = [
  ['include', 'All plans'],
  ['only', 'Only BCPIR plans'],
  ['exclude', 'No BCPIR plans'],
] as const;

interface BcpirChoiceProps {
  /** The radio group's name, unique on the page. */
  name: string;
  value: PlanListFilters['bcpir'];
  onChange: (next: Partial<PlanListFilters>) => void;
}

/** The "BCPIR plans" choice of the Plans list's Filters menu (013 FR-003). */
export function BcpirChoice({ name, value, onChange }: BcpirChoiceProps) {
  return (
    <fieldset className={styles.field}>
      <legend className={styles.label}>BCPIR plans</legend>
      {CHOICES.map(([choice, label]) => (
        <label key={choice} className={styles.radio}>
          <input
            type="radio"
            name={name}
            value={choice}
            checked={value === choice}
            onChange={() => onChange({ bcpir: choice })}
          />
          {label}
        </label>
      ))}
    </fieldset>
  );
}

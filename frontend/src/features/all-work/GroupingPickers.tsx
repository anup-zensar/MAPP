import { useId } from 'react';
import type { BreakdownChoice, Grouping } from './all-work-types';
import { BREAK_DOWN_BY, LEVEL_LABELS, NOTHING_ELSE, THEN_BY } from './breakdown-text';
import styles from './GroupingPickers.module.css';

export interface GroupingPickersProps {
  groupings: Grouping[];
  choice: BreakdownChoice;
  onChange: (choice: BreakdownChoice) => void;
}

const GENERAL = 'Plans and activities';

/** "Break down by" and "then by": the API's groupings, Shell attributes under their level. */
export function GroupingPickers({ groupings, choice, onChange }: GroupingPickersProps) {
  const id = useId();
  const others = groupings.filter((grouping) => grouping.key !== choice.by);

  return (
    <div className={styles.pickers}>
      <label htmlFor={`${id}-by`} className={styles.label}>
        {BREAK_DOWN_BY}
      </label>
      <select
        id={`${id}-by`}
        className={styles.select}
        value={choice.by}
        onChange={(event) => {
          const by = event.target.value;
          onChange({ by, thenBy: choice.thenBy === by ? null : choice.thenBy });
        }}
      >
        <Options groupings={groupings} />
      </select>
      <label htmlFor={`${id}-then`} className={styles.label}>
        {THEN_BY}
      </label>
      <select
        id={`${id}-then`}
        className={styles.select}
        value={choice.thenBy ?? ''}
        onChange={(event) => onChange({ by: choice.by, thenBy: event.target.value || null })}
      >
        <option value="">{NOTHING_ELSE}</option>
        <Options groupings={others} />
      </select>
    </div>
  );
}

function Options({ groupings }: { groupings: Grouping[] }) {
  const sections = new Map<string, Grouping[]>();
  for (const grouping of groupings) {
    const section = grouping.level ? (LEVEL_LABELS[grouping.level] ?? grouping.level) : GENERAL;
    sections.set(section, [...(sections.get(section) ?? []), grouping]);
  }
  return [...sections].map(([section, list]) => (
    <optgroup key={section} label={section}>
      {list.map((grouping) => (
        <option key={grouping.key} value={grouping.key}>
          {grouping.label}
        </option>
      ))}
    </optgroup>
  ));
}

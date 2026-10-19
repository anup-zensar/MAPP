import { ChevronDown } from 'lucide-react';
import { Popover } from 'radix-ui';
import { Fragment, useId, useState } from 'react';
import { cx } from '@/lib/cx';
import styles from './FilterMenu.module.css';

/** One choice of a filter. */
export interface FilterChoice {
  value: string;
  label: string;
  inactive?: boolean;
  /** The heading the choice is listed under ("Approval progress", "Timing"). */
  group?: string;
}

export interface FilterMenuProps {
  /** The filter's name: "Owner". It names the button and the menu. */
  label: string;
  choices: FilterChoice[];
  selected: string[];
  /** Several values (check boxes) or one (radio buttons). */
  multiple: boolean;
  onChange: (values: string[]) => void;
  /**
   * What the button shows after the name: the number of chosen values ("Owner · 2"), or the
   * chosen value of a single filter ("Year · 2026"); nothing when the filter is not in use.
   */
  summary?: string;
  /** A first choice that ticks several values at once: Team's "My teams". */
  quickChoice?: { label: string; values: string[] };
}

/** Lists with more choices than this get a search box (design system §5). */
const SEARCHABLE_FROM = 10;

/**
 * One filter of the filter bar (plan "Page layout"): a button that opens a list of check boxes,
 * or radio buttons for a single choice. A change applies at once (FR-009). Inactive values say
 * "(inactive)"; long lists can be searched.
 */
export function FilterMenu({
  label,
  choices,
  selected,
  multiple,
  onChange,
  summary,
  quickChoice,
}: FilterMenuProps) {
  const id = useId();
  const [search, setSearch] = useState('');
  const needle = search.trim().toLowerCase();
  // A value chosen earlier that is no longer offered stays in the list, so it can be removed.
  const all: FilterChoice[] = [
    ...choices,
    ...selected
      .filter((value) => !choices.some((choice) => choice.value === value))
      .map((value) => ({ value, label: value })),
  ];
  const shown = needle ? all.filter((choice) => choice.label.toLowerCase().includes(needle)) : all;
  const quickTicked =
    quickChoice !== undefined &&
    quickChoice.values.length > 0 &&
    quickChoice.values.every((value) => selected.includes(value));

  function toggle(value: string, ticked: boolean) {
    if (!multiple) {
      onChange([value]);
      return;
    }
    onChange(ticked ? [...selected, value] : selected.filter((chosen) => chosen !== value));
  }

  function toggleQuick(ticked: boolean) {
    if (!quickChoice) return;
    const others = selected.filter((value) => !quickChoice.values.includes(value));
    onChange(ticked ? [...quickChoice.values, ...others] : others);
  }

  return (
    <Popover.Root onOpenChange={() => setSearch('')}>
      <Popover.Trigger asChild>
        <button type="button" className={cx(styles.button, summary && styles.inUse)}>
          {label}
          {/* The space stays outside the span, so screen readers keep it between the words. */}
          {summary && ' '}
          {summary && <span className={styles.summary}>· {summary}</span>}
          <ChevronDown aria-hidden="true" />
        </button>
      </Popover.Trigger>
      <Popover.Portal>
        <Popover.Content className={styles.menu} align="start" sideOffset={6} aria-label={label}>
          {all.length > SEARCHABLE_FROM && (
            <div className={styles.search}>
              <label htmlFor={`${id}-search`} className="visually-hidden">
                Search filters
              </label>
              <input
                id={`${id}-search`}
                type="search"
                placeholder="Search filters"
                value={search}
                onChange={(event) => setSearch(event.target.value)}
              />
            </div>
          )}
          <fieldset className={styles.choices}>
            <legend className="visually-hidden">{label}</legend>
            {quickChoice && !needle && (
              <label className={cx(styles.choice, styles.quick)}>
                <input
                  type="checkbox"
                  checked={quickTicked}
                  onChange={(event) => toggleQuick(event.target.checked)}
                />
                {quickChoice.label}
              </label>
            )}
            {shown.length === 0 && <p className={styles.nothing}>Nothing to choose from.</p>}
            {shown.map((choice, index) => {
              const startsGroup =
                choice.group !== undefined && shown[index - 1]?.group !== choice.group;
              return (
                <Fragment key={choice.value}>
                  {startsGroup && <p className={styles.group}>{choice.group}</p>}
                  <label className={styles.choice}>
                    <input
                      type={multiple ? 'checkbox' : 'radio'}
                      name={multiple ? undefined : `${id}-choice`}
                      checked={selected.includes(choice.value)}
                      onChange={(event) => toggle(choice.value, event.target.checked)}
                    />
                    {choice.label}
                    {choice.inactive && ' (inactive)'}
                  </label>
                </Fragment>
              );
            })}
          </fieldset>
        </Popover.Content>
      </Popover.Portal>
    </Popover.Root>
  );
}

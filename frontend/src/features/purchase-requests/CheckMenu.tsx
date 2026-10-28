import { Check } from 'lucide-react';
import { DropdownMenu } from 'radix-ui';
import { useId } from 'react';
import type { Choice } from './list-filters';
import styles from './PurchaseRequestFilters.module.css';

export interface CheckMenuProps {
  label: string;
  anyText: string;
  choices: Choice[];
  chosen: string[];
  onChange: (chosen: string[]) => void;
  emptyText?: string;
}

/** A menu of check boxes that stays open while people tick several (`modal={false}`). */
export function CheckMenu({
  label,
  anyText,
  choices,
  chosen,
  onChange,
  emptyText,
}: CheckMenuProps) {
  const labelId = useId();
  const words = choices.filter((choice) => chosen.includes(choice.value)).map((c) => c.label);
  const summary = words.length > 0 ? words.join(', ') : anyText;

  function toggle(value: string, checked: boolean) {
    const next = checked ? [...chosen, value] : chosen.filter((each) => each !== value);
    const known = choices.map((choice) => choice.value).filter((each) => next.includes(each));
    onChange([...known, ...next.filter((each) => !known.includes(each))]);
  }

  return (
    <div className={styles.field}>
      <span id={labelId}>{label}</span>
      <DropdownMenu.Root modal={false}>
        <DropdownMenu.Trigger asChild>
          <button
            type="button"
            className={`${styles.control} ${styles.select} ${styles.menuButton}`}
            aria-label={`${label}: ${summary}`}
          >
            <span className={styles.menuText}>{summary}</span>
          </button>
        </DropdownMenu.Trigger>
        <DropdownMenu.Portal>
          <DropdownMenu.Content
            className={styles.menu}
            align="start"
            sideOffset={4}
            aria-labelledby={labelId}
          >
            {choices.length === 0 && emptyText && <p className={styles.empty}>{emptyText}</p>}
            {choices.map((choice) => (
              <DropdownMenu.CheckboxItem
                key={choice.value}
                className={styles.menuItem}
                checked={chosen.includes(choice.value)}
                onCheckedChange={(checked) => toggle(choice.value, checked)}
                onSelect={(event) => event.preventDefault()}
              >
                <span className={styles.box} aria-hidden="true">
                  <DropdownMenu.ItemIndicator>
                    <Check />
                  </DropdownMenu.ItemIndicator>
                </span>
                {choice.label}
              </DropdownMenu.CheckboxItem>
            ))}
          </DropdownMenu.Content>
        </DropdownMenu.Portal>
      </DropdownMenu.Root>
    </div>
  );
}

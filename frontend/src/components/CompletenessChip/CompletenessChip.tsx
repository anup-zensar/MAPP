import { ChevronDown } from 'lucide-react';
import { Popover } from 'radix-ui';
import { useRef, useState } from 'react';
import { cx } from '@/lib/cx';
import styles from './CompletenessChip.module.css';

/** A field that still has to be filled in. */
export interface MissingField {
  /** What onSelectField receives, usually the field's input id. */
  id: string;
  label: string;
}

export interface CompletenessChipProps {
  missing: MissingField[];
  /** Takes the person to the chosen field, for example by focusing its input. */
  onSelectField?: (id: string) => void;
}

/**
 * Says whether a record has everything it needs: "All filled in", or "3 fields needed" with a
 * list of those fields that leads to each of them (design system §5, constitution III).
 */
export function CompletenessChip({ missing, onSelectField }: CompletenessChipProps) {
  const [open, setOpen] = useState(false);
  // The field chosen in the list; the chip moves to it once the list has closed.
  const chosenField = useRef<string | null>(null);

  if (missing.length === 0) {
    return <span className={cx(styles.chip, styles.complete)}>All filled in</span>;
  }

  const summary = missing.length === 1 ? '1 field needed' : `${missing.length} fields needed`;

  function choose(id: string) {
    chosenField.current = id;
    setOpen(false);
  }

  return (
    <Popover.Root open={open} onOpenChange={setOpen}>
      <Popover.Trigger className={cx(styles.chip, styles.incomplete)}>
        {summary}
        <ChevronDown />
      </Popover.Trigger>
      <Popover.Portal>
        <Popover.Content
          className={styles.list}
          align="start"
          sideOffset={6}
          aria-label={summary}
          onCloseAutoFocus={(event) => {
            // After a choice the focus goes to that field, not back to the chip.
            const id = chosenField.current;
            chosenField.current = null;
            if (id && onSelectField) {
              event.preventDefault();
              onSelectField(id);
            }
          }}
        >
          <ul>
            {missing.map((field) => (
              <li key={field.id}>
                {onSelectField ? (
                  <button type="button" className={styles.field} onClick={() => choose(field.id)}>
                    {field.label}
                  </button>
                ) : (
                  <span className={styles.field}>{field.label}</span>
                )}
              </li>
            ))}
          </ul>
        </Popover.Content>
      </Popover.Portal>
    </Popover.Root>
  );
}

import { ChevronDown } from 'lucide-react';
import { Popover } from 'radix-ui';
import { useRef, useState } from 'react';
import { Link } from 'react-router';
import { cx } from '@/lib/cx';
import styles from './CompletenessChip.module.css';

/** A field that still has to be filled in. */
export interface MissingField {
  /** What onSelectField receives, usually the field's input id. */
  id: string;
  label: string;
}

/** A record below this one that needs attention, with the address of its page or panel. */
export interface ItemBelow {
  id: string;
  label: string;
  href: string;
}

export interface CompletenessChipProps {
  missing: MissingField[];
  /** Takes the person to the chosen field, for example by focusing its input. */
  onSelectField?: (id: string) => void;
  /**
   * Records below that need attention (feature 004, D-199): a list (a plan's, with links) or a
   * count (campaigns, initiatives, activities). Adds a second chip "1 item below needs attention" /
   * "<n> items below need attention"; nothing for 0.
   */
  itemsBelow?: number | ItemBelow[];
  /** Makes a counted second chip a button: opens the place that lists them. */
  onItemsBelowSelect?: () => void;
  /**
   * The second chip's text when the records below have a name of their own (feature 006: "1 line
   * item needs attention"); "<n> items below need attention" otherwise.
   */
  itemsBelowText?: (count: number) => string;
}

function itemsBelowSummary(count: number): string {
  return count === 1 ? '1 item below needs attention' : `${count} items below need attention`;
}

/**
 * Says whether a record has everything it needs: "All filled in", or "3 fields needed" with a
 * list of those fields that leads to each of them (design system §5, constitution III).
 */
export function CompletenessChip({
  missing,
  onSelectField,
  itemsBelow,
  onItemsBelowSelect,
  itemsBelowText = itemsBelowSummary,
}: CompletenessChipProps) {
  const below = typeof itemsBelow === 'number' ? itemsBelow : (itemsBelow?.length ?? 0);
  if (below === 0) {
    return <FieldsChip missing={missing} onSelectField={onSelectField} />;
  }
  return (
    <span className={styles.chips}>
      <FieldsChip missing={missing} onSelectField={onSelectField} />
      <ItemsBelowChip
        itemsBelow={itemsBelow ?? 0}
        onSelect={onItemsBelowSelect}
        summary={itemsBelowText(below)}
      />
    </span>
  );
}

/** The chip of the record's own fields. */
function FieldsChip({
  missing,
  onSelectField,
}: Pick<CompletenessChipProps, 'missing' | 'onSelectField'>) {
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

/** "<n> items below need attention": a list with links, a button, or a plain chip. */
function ItemsBelowChip({
  itemsBelow,
  onSelect,
  summary,
}: {
  itemsBelow: number | ItemBelow[];
  onSelect?: () => void;
  summary: string;
}) {
  if (typeof itemsBelow === 'number') {
    return onSelect ? (
      <button type="button" className={cx(styles.chip, styles.incomplete)} onClick={onSelect}>
        {summary}
      </button>
    ) : (
      <span className={cx(styles.chip, styles.attention)}>{summary}</span>
    );
  }

  return (
    <Popover.Root>
      <Popover.Trigger className={cx(styles.chip, styles.incomplete)}>
        {summary}
        <ChevronDown />
      </Popover.Trigger>
      <Popover.Portal>
        <Popover.Content className={styles.list} align="start" sideOffset={6} aria-label={summary}>
          <ul>
            {itemsBelow.map((item) => (
              <li key={item.id}>
                <Link to={item.href} className={styles.field}>
                  {item.label}
                </Link>
              </li>
            ))}
          </ul>
        </Popover.Content>
      </Popover.Portal>
    </Popover.Root>
  );
}

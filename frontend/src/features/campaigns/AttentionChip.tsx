import { ChevronDown } from 'lucide-react';
import { Popover } from 'radix-ui';
import { useRef, useState } from 'react';
import { Link } from 'react-router';
import type { ItemBelow, MissingField } from '@/components';
import styles from './AttentionChip.module.css';

export interface AttentionChipProps {
  /** The record's own empty needed fields. */
  missing: MissingField[];
  /** Takes the person to the chosen field (its edit form). */
  onSelectField?: (id: string) => void;
  /** Records below that need attention: a list with links (a plan's) or a count. */
  itemsBelow?: number | ItemBelow[];
  /** With a count: opens the place that lists them. */
  onItemsBelowSelect?: () => void;
  /**
   * Tree rows: "2 items below" on the chip (the popover and screen readers keep the whole
   * sentence), so the Status column stays narrow.
   */
  compact?: boolean;
}

/** "1 field needed" / "3 fields needed". */
function fieldsNeeded(count: number): string {
  return count === 1 ? '1 field needed' : `${count} fields needed`;
}

/** "1 item below needs attention" / "5 items below need attention". */
function itemsBelowNeedAttention(count: number): string {
  return count === 1 ? '1 item below needs attention' : `${count} items below need attention`;
}

/**
 * The one completeness chip of a tree row, a panel or a page header (design system §4.1: at most
 * two chips per row, the status and one problem chip). "All filled in" when nothing is missing;
 * otherwise one red chip — the record's own missing fields first, else the records below — whose
 * popover lists everything: each missing field (it opens the form on that field) and the records
 * below that need attention.
 */
export function AttentionChip({
  missing,
  onSelectField,
  itemsBelow = 0,
  onItemsBelowSelect,
  compact = false,
}: AttentionChipProps) {
  const [open, setOpen] = useState(false);
  // The field or action chosen in the list; it runs once the list has closed, so the focus can
  // move to the field instead of coming back to the chip.
  const chosen = useRef<(() => void) | null>(null);
  const below = typeof itemsBelow === 'number' ? itemsBelow : itemsBelow.length;

  if (missing.length === 0 && below === 0) {
    return <span className={`${styles.chip} ${styles.complete}`}>All filled in</span>;
  }

  const summary =
    missing.length > 0 ? fieldsNeeded(missing.length) : itemsBelowNeedAttention(below);

  function choose(action: () => void) {
    chosen.current = action;
    setOpen(false);
  }

  return (
    <Popover.Root open={open} onOpenChange={setOpen}>
      <Popover.Trigger className={`${styles.chip} ${styles.problem}`}>
        {compact && missing.length === 0 ? (
          <>
            <span aria-hidden="true">{below === 1 ? '1 item below' : `${below} items below`}</span>
            <span className="visually-hidden">{summary}</span>
          </>
        ) : (
          summary
        )}
        <ChevronDown aria-hidden="true" />
      </Popover.Trigger>
      <Popover.Portal>
        <Popover.Content
          className={styles.list}
          align="start"
          sideOffset={6}
          aria-label={summary}
          onCloseAutoFocus={(event) => {
            const action = chosen.current;
            chosen.current = null;
            if (action) {
              event.preventDefault();
              action();
            }
          }}
        >
          {missing.length > 0 && (
            <div className={styles.group}>
              <p className={styles.groupTitle}>{fieldsNeeded(missing.length)}</p>
              <ul>
                {missing.map((field) => (
                  <li key={field.id}>
                    {onSelectField ? (
                      <button
                        type="button"
                        className={styles.item}
                        onClick={() => choose(() => onSelectField(field.id))}
                      >
                        {field.label}
                      </button>
                    ) : (
                      <span className={styles.text}>{field.label}</span>
                    )}
                  </li>
                ))}
              </ul>
            </div>
          )}
          {below > 0 && (
            <div className={styles.group}>
              <p className={styles.groupTitle}>{itemsBelowNeedAttention(below)}</p>
              {typeof itemsBelow === 'number' ? (
                onItemsBelowSelect && (
                  <ul>
                    <li>
                      <button
                        type="button"
                        className={styles.item}
                        onClick={() => choose(onItemsBelowSelect)}
                      >
                        Show them
                      </button>
                    </li>
                  </ul>
                )
              ) : (
                <ul>
                  {itemsBelow.map((item) => (
                    <li key={item.id}>
                      <Link to={item.href} className={styles.item} onClick={() => setOpen(false)}>
                        {item.label}
                      </Link>
                    </li>
                  ))}
                </ul>
              )}
            </div>
          )}
        </Popover.Content>
      </Popover.Portal>
    </Popover.Root>
  );
}

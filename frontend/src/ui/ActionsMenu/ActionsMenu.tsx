import { ChevronDown, Ellipsis } from 'lucide-react';
import { DropdownMenu } from 'radix-ui';
import { Fragment, useId, useRef } from 'react';
import { cx } from '../cx';
import { Button } from '../Button/Button';
import { Tooltip } from '../Tooltip/Tooltip';
import styles from './ActionsMenu.module.css';

/** One action of the menu. */
export interface ActionsMenuItem {
  label: string;
  onSelect: () => void;
  /** danger: shown in red, for actions that remove something (Delete). */
  tone?: 'danger';
  /** Draws a line above the item; Delete comes last, after a line (design system §6.1). */
  separatorBefore?: boolean;
  /**
   * A second, smaller line under the label, read with it (feature 003): "People are never
   * deleted, so the history keeps their name."
   */
  hint?: string;
  /**
   * An action the person may not take on this record now (feature 014, FR-011): shown, not chosen,
   * with its reason as the hint — never greyed out without one (constitution III). See
   * availabilityProps in @/lib/access.
   */
  disabled?: boolean;
}

export interface ActionsMenuProps {
  /** The record the actions are for; the ⋯ button is named "Actions for <label>". */
  label: string;
  items: ActionsMenuItem[];
  /**
   * The button that opens the menu: "icon" (default) is the ⋯ of a table row or card; "more" is
   * the **More** button of a page header or panel footer (design system §6.3).
   */
  trigger?: 'icon' | 'more';
}

/**
 * The "⋯" menu of a row, or the **More** menu of a page or panel: Edit, record-specific actions
 * and Delete, last after a line (design system §6.1, §6.3).
 *
 * The chosen action runs only after the menu has closed and given the focus back to "⋯".
 * A dialog opened by the action therefore starts from "⋯" and returns the focus there when it
 * closes, instead of losing it.
 */
export function ActionsMenu({ label, items, trigger = 'icon' }: ActionsMenuProps) {
  const chosenAction = useRef<(() => void) | null>(null);
  const hintId = useId();

  return (
    // Not modal, like every MAPP menu: the rest of the page stays usable while it is open.
    <DropdownMenu.Root modal={false}>
      {trigger === 'more' ? (
        // A written label needs no tooltip: the button's name is the word on it.
        <DropdownMenu.Trigger asChild>
          <Button>
            More
            <ChevronDown aria-hidden="true" />
          </Button>
        </DropdownMenu.Trigger>
      ) : (
        <Tooltip content={`Actions for ${label}`}>
          <DropdownMenu.Trigger asChild>
            <Button
              variant="tertiary"
              size="sm"
              icon={<Ellipsis />}
              aria-label={`Actions for ${label}`}
            />
          </DropdownMenu.Trigger>
        </Tooltip>
      )}
      <DropdownMenu.Portal>
        <DropdownMenu.Content
          className={styles.menu}
          align="end"
          sideOffset={4}
          onCloseAutoFocus={() => {
            // Radix puts the focus back on "⋯" right after this; then the action may open a dialog.
            const action = chosenAction.current;
            chosenAction.current = null;
            action?.();
          }}
        >
          {items.map((item, index) => (
            <Fragment key={item.label}>
              {item.separatorBefore && <DropdownMenu.Separator className={styles.separator} />}
              <DropdownMenu.Item
                className={cx(styles.item, item.tone === 'danger' && styles.danger)}
                aria-describedby={item.hint ? `${hintId}-${index}` : undefined}
                disabled={item.disabled}
                onSelect={() => {
                  chosenAction.current = item.onSelect;
                }}
              >
                {item.label}
                {item.hint && (
                  // Out of the item's name; aria-describedby reads it as its description.
                  <span id={`${hintId}-${index}`} className={styles.hint} aria-hidden="true">
                    {item.hint}
                  </span>
                )}
              </DropdownMenu.Item>
            </Fragment>
          ))}
        </DropdownMenu.Content>
      </DropdownMenu.Portal>
    </DropdownMenu.Root>
  );
}

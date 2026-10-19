import { Ellipsis } from 'lucide-react';
import { DropdownMenu } from 'radix-ui';
import { Fragment, useRef } from 'react';
import { cx } from '@/lib/cx';
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
}

export interface ActionsMenuProps {
  /** The record the actions are for; the button is named "Actions for <label>". */
  label: string;
  items: ActionsMenuItem[];
}

/**
 * The "⋯" menu of a row: Edit, record-specific actions and Delete (design system §6.1).
 *
 * The chosen action runs only after the menu has closed and given the focus back to "⋯".
 * A dialog opened by the action therefore starts from "⋯" and returns the focus there when it
 * closes, instead of losing it.
 */
export function ActionsMenu({ label, items }: ActionsMenuProps) {
  const chosenAction = useRef<(() => void) | null>(null);

  return (
    // Not modal, like every NorthStar menu: the rest of the page stays usable while it is open.
    <DropdownMenu.Root modal={false}>
      <Tooltip content={`Actions for ${label}`}>
        <DropdownMenu.Trigger asChild>
          <Button
            variant="ghost"
            size="sm"
            icon={<Ellipsis />}
            aria-label={`Actions for ${label}`}
          />
        </DropdownMenu.Trigger>
      </Tooltip>
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
          {items.map((item) => (
            <Fragment key={item.label}>
              {item.separatorBefore && <DropdownMenu.Separator className={styles.separator} />}
              <DropdownMenu.Item
                className={cx(styles.item, item.tone === 'danger' && styles.danger)}
                onSelect={() => {
                  chosenAction.current = item.onSelect;
                }}
              >
                {item.label}
              </DropdownMenu.Item>
            </Fragment>
          ))}
        </DropdownMenu.Content>
      </DropdownMenu.Portal>
    </DropdownMenu.Root>
  );
}

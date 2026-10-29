import { ChevronDown, Ellipsis } from 'lucide-react';
import { DropdownMenu } from 'radix-ui';
import { Fragment, useId, useRef } from 'react';
import { cx } from '../cx';
import { Button } from '../Button/Button';
import { Tooltip } from '../Tooltip/Tooltip';
import { useHidesUnavailable } from '../ViewOnlyNote/view-only-context';
import styles from './ActionsMenu.module.css';

export interface ActionsMenuItem {
  label: string;
  onSelect: () => void;
  tone?: 'danger';
  separatorBefore?: boolean;
  /** A second, smaller line under the label, read with it: the reason of a disabled item. */
  hint?: string;
  /** Shown, not chosen, with its reason as the hint; left out inside a ViewOnlyArea. */
  disabled?: boolean;
}

export interface ActionsMenuProps {
  label: string;
  items: ActionsMenuItem[];
  /** "icon": the ⋯ of a row or card; "more": the **More** button of a page header or panel. */
  trigger?: 'icon' | 'more';
}

// The chosen action runs once the menu has given the focus back to its button, so a dialog it
// opens returns the focus there.
export function ActionsMenu({ label, items, trigger = 'icon' }: ActionsMenuProps) {
  const chosenAction = useRef<(() => void) | null>(null);
  const hintId = useId();
  const hidesUnavailable = useHidesUnavailable();
  const shown = hidesUnavailable ? items.filter((item) => !item.disabled) : items;

  return (
    <DropdownMenu.Root modal={false}>
      {trigger === 'more' ? (
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
            const action = chosenAction.current;
            chosenAction.current = null;
            action?.();
          }}
        >
          {shown.map((item, index) => (
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

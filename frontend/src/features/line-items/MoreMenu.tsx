import { ChevronDown } from 'lucide-react';
import { Fragment, useRef } from 'react';
import { DropdownMenu } from 'radix-ui';
import { Button } from '@/components';
import { cx } from '@/lib/cx';
import styles from './MoreMenu.module.css';

export interface MoreMenuItem {
  label: string;
  onSelect: () => void;
  tone?: 'danger';
  separatorBefore?: boolean;
}

/**
 * The details page's **More** button (FR-026): Duplicate, View history and, after a line, Delete.
 * Like every NorthStar menu it is not modal, and the chosen action runs once the menu has closed
 * and given the focus back, so a dialog it opens returns the focus here.
 */
export function MoreMenu({ items }: { items: MoreMenuItem[] }) {
  const chosen = useRef<(() => void) | null>(null);
  return (
    <DropdownMenu.Root modal={false}>
      <DropdownMenu.Trigger asChild>
        <Button>
          More
          <ChevronDown className={styles.chevron} aria-hidden="true" />
        </Button>
      </DropdownMenu.Trigger>
      <DropdownMenu.Portal>
        <DropdownMenu.Content
          className={styles.menu}
          align="end"
          sideOffset={4}
          onCloseAutoFocus={() => {
            const action = chosen.current;
            chosen.current = null;
            action?.();
          }}
        >
          {items.map((item) => (
            <Fragment key={item.label}>
              {item.separatorBefore && <DropdownMenu.Separator className={styles.separator} />}
              <DropdownMenu.Item
                className={cx(styles.item, item.tone === 'danger' && styles.danger)}
                onSelect={() => {
                  chosen.current = item.onSelect;
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

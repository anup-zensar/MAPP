import { Bookmark, Check } from 'lucide-react';
import { DropdownMenu } from 'radix-ui';
import { useRef } from 'react';
import { Button } from '@/ui/Button/Button';
import type { SavedView } from './allWorkTypes';
import styles from './ViewsMenu.module.css';

export interface ViewsMenuProps {
  /** The person's saved views, by name; undefined while they load. */
  views: SavedView[] | undefined;
  /** The view applied last, if it still exists. */
  active: SavedView | null;
  /** What is shown differs from the active view (FR-030). */
  modified: boolean;
  onApply: (view: SavedView) => void;
  onSave: () => void;
  onUpdate: () => void;
  onSaveAsNew: () => void;
  onManage: () => void;
  onStandard: () => void;
}

/**
 * **Views** (US6, FR-029 – FR-031): the person's saved views with the active one ticked and the
 * default one tagged, then Save current view…, Update view and Save as new view… when the view
 * is Modified, Manage views… and Standard view. The button names the active view.
 *
 * The chosen item runs only after the menu has closed and given the focus back to the button, so
 * a dialog opened from it returns the focus there too.
 */
export function ViewsMenu({
  views,
  active,
  modified,
  onApply,
  onSave,
  onUpdate,
  onSaveAsNew,
  onManage,
  onStandard,
}: ViewsMenuProps) {
  const chosen = useRef<(() => void) | null>(null);

  return (
    <DropdownMenu.Root modal={false}>
      <DropdownMenu.Trigger asChild>
        <Button icon={<Bookmark />}>
          {active ? `View: ${active.name}` : 'Views'}
          {modified && ' '}
          {modified && <span className={styles.modified}>Modified</span>}
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
          {views && views.length === 0 && (
            <p className={styles.empty}>
              No saved views yet — set up filters and columns, then choose Save current view.
            </p>
          )}
          {views?.map((view) => (
            <DropdownMenu.CheckboxItem
              key={view.id}
              className={styles.item}
              checked={view.id === active?.id}
              onSelect={() => {
                chosen.current = () => onApply(view);
              }}
            >
              <span className={styles.tick}>
                <DropdownMenu.ItemIndicator>
                  <Check aria-hidden="true" />
                </DropdownMenu.ItemIndicator>
              </span>
              <span className={styles.name}>{view.name}</span>
              {view.isDefault && <span className={styles.default}>Default</span>}
            </DropdownMenu.CheckboxItem>
          ))}
          {views && views.length > 0 && <DropdownMenu.Separator className={styles.separator} />}
          <DropdownMenu.Item
            className={styles.item}
            onSelect={() => {
              chosen.current = onSave;
            }}
          >
            Save current view…
          </DropdownMenu.Item>
          {active && modified && (
            <>
              <DropdownMenu.Item
                className={styles.item}
                onSelect={() => {
                  chosen.current = onUpdate;
                }}
              >
                Update view
              </DropdownMenu.Item>
              <DropdownMenu.Item
                className={styles.item}
                onSelect={() => {
                  chosen.current = onSaveAsNew;
                }}
              >
                Save as new view…
              </DropdownMenu.Item>
            </>
          )}
          {views && views.length > 0 && (
            <DropdownMenu.Item
              className={styles.item}
              onSelect={() => {
                chosen.current = onManage;
              }}
            >
              Manage views…
            </DropdownMenu.Item>
          )}
          <DropdownMenu.Item
            className={styles.item}
            onSelect={() => {
              chosen.current = onStandard;
            }}
          >
            Standard view
          </DropdownMenu.Item>
        </DropdownMenu.Content>
      </DropdownMenu.Portal>
    </DropdownMenu.Root>
  );
}

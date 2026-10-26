import { ArrowDown, ArrowUp, GripVertical } from 'lucide-react';
import { useId, useState } from 'react';
import type { DragEvent } from 'react';
import { Button } from '@/ui/Button/Button';
import { CheckboxField } from '@/ui/CheckboxField/CheckboxField';
import { SearchField } from '@/ui/SearchField/SearchField';
import { SidePanel } from '@/ui/SidePanel/SidePanel';
import type { AllWorkColumnGroup } from './all-work-types';
import styles from './ColumnChooser.module.css';
import { dropColumn, moveColumn } from './view-columns';

export interface ColumnChooserProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** The column catalogue: Standard, then one group per level (FR-018). */
  groups: AllWorkColumnGroup[];
  /** The chosen columns, in order (Name is always first and not among them). */
  columns: string[];
  defaultColumns: string[];
  /** Every change applies at once. */
  onChange: (columns: string[]) => void;
}

/** Why the Name tick is locked (FR-019). */
const NAME_LOCKED = 'The name column is always shown so you can tell rows apart';

/**
 * The column chooser (US3, FR-018 – FR-021): searchable groups of columns to tick, Name locked
 * with its reason, the chosen columns in order — moved by dragging or with Move up and Move down
 * — and Reset to default columns. Shell attributes added in Settings appear here by themselves.
 */
export function ColumnChooser({
  open,
  onOpenChange,
  groups,
  columns,
  defaultColumns,
  onChange,
}: ColumnChooserProps) {
  const id = useId();
  const [search, setSearch] = useState('');
  const [dragged, setDragged] = useState<string | null>(null);
  const needle = search.trim().toLowerCase();
  const labelOf = (key: string) =>
    groups.flatMap((group) => group.columns).find((column) => column.key === key)?.label ?? key;

  function tick(key: string, ticked: boolean) {
    onChange(ticked ? [...columns, key] : columns.filter((chosen) => chosen !== key));
  }

  function drop(event: DragEvent, target: string) {
    event.preventDefault();
    if (dragged) {
      onChange(dropColumn(columns, dragged, target));
    }
    setDragged(null);
  }

  return (
    <SidePanel
      open={open}
      onOpenChange={onOpenChange}
      title="Columns"
      description="Choose and order the columns of the grid."
      footer={
        <Button onClick={() => onChange([...defaultColumns])}>Reset to default columns</Button>
      }
    >
      <div className={styles.chooser}>
        <SearchField
          label="Search columns"
          placeholder="Search columns"
          value={search}
          onChange={setSearch}
        />
        {groups.map((group) => {
          const shown = group.columns.filter((column) =>
            column.label.toLowerCase().includes(needle),
          );
          const showName = group.group === 'standard' && 'name'.includes(needle);
          if (shown.length === 0 && !showName) {
            return null;
          }
          return (
            <fieldset key={group.group} className={styles.group}>
              <legend className={styles.legend}>{group.label}</legend>
              {showName && (
                <CheckboxField
                  label="Name"
                  name={`${id}-name`}
                  checked
                  onChange={() => {}}
                  disabledReason={NAME_LOCKED}
                />
              )}
              {shown.map((column) => (
                <CheckboxField
                  key={column.key}
                  label={column.inactive ? `${column.label} (inactive)` : column.label}
                  name={`${id}-${column.key}`}
                  checked={columns.includes(column.key)}
                  onChange={(ticked) => tick(column.key, ticked)}
                />
              ))}
            </fieldset>
          );
        })}
        <section className={styles.order} aria-labelledby={`${id}-order`}>
          <h3 id={`${id}-order`} className={styles.legend}>
            Shown columns, in order
          </h3>
          <ol className={styles.list}>
            {columns.map((key, index) => {
              const label = labelOf(key);
              return (
                <li
                  key={key}
                  className={styles.item}
                  draggable
                  data-dragging={dragged === key || undefined}
                  onDragStart={(event) => {
                    event.dataTransfer?.setData('text/plain', key);
                    setDragged(key);
                  }}
                  onDragOver={(event) => event.preventDefault()}
                  onDrop={(event) => drop(event, key)}
                  onDragEnd={() => setDragged(null)}
                >
                  <GripVertical className={styles.grip} aria-hidden="true" />
                  <span className={styles.label}>{label}</span>
                  {index > 0 && (
                    <Button
                      variant="tertiary"
                      size="sm"
                      icon={<ArrowUp />}
                      aria-label={`Move ${label} up`}
                      onClick={() => onChange(moveColumn(columns, key, -1))}
                    />
                  )}
                  {index < columns.length - 1 && (
                    <Button
                      variant="tertiary"
                      size="sm"
                      icon={<ArrowDown />}
                      aria-label={`Move ${label} down`}
                      onClick={() => onChange(moveColumn(columns, key, 1))}
                    />
                  )}
                </li>
              );
            })}
          </ol>
        </section>
      </div>
    </SidePanel>
  );
}

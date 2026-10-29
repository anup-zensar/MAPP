import { useEffect, useId, useRef, useState } from 'react';
import { ActionsMenu } from '@/ui/ActionsMenu/ActionsMenu';
import { Avatar } from '@/ui/Avatar/Avatar';
import { Button } from '@/ui/Button/Button';
import { LevelTag } from '@/ui/LevelTag/LevelTag';
import type { LevelTagProps } from '@/ui/LevelTag/LevelTag';
import { Tooltip } from '@/ui/Tooltip/Tooltip';
import type { ActionsMenuItem } from '@/ui/ActionsMenu/ActionsMenu';
import { isPlanningType } from '@/ui/record-links';
import { availabilityProps, isShown } from '@/session/access';
import { initialsOf } from '@/features/activity-log';
import { useFormatters } from '@/format/useFormatters';
import {
  capitalise,
  insideRecords,
  restoreStateLine,
  rowContentsText,
  wasInText,
} from './bin-text';
import type { BinItem, BinRecord } from './bin-text';
import styles from './BinRow.module.css';

export interface BinRowProps {
  item: BinItem;
  /** "Contains A-1042 'Fleet expo stand'" when the search found a record inside it. */
  containsText: string;
  /** The row a link from the activity log points at (`?record=`): scrolled to and focused. */
  highlighted: boolean;
  /** With `highlighted`: the record is inside the entry, so its contents start open. */
  highlightInside: boolean;
  onRestore: () => void;
  onPurge: () => void;
}

/**
 * One recycle-bin entry (FR-015, FR-017): its level, name and ID, where it was, what was inside,
 * who deleted it and when, why it cannot simply come back, **Restore** and ⋯ **Delete
 * permanently**, and its contents as a tree under **Show contents**.
 */
export function BinRow({
  item,
  containsText,
  highlighted,
  highlightInside,
  onRestore,
  onPurge,
}: BinRowProps) {
  const { dateTime, relativeTime } = useFormatters();
  // What the person's roles allow here (feature 014, FR-011): an action they may not take moves
  // into the menu, shown with the server's reason; one without a reason is left out.
  const restoreOk = !item.access || item.access.restore.available;
  const menu: ActionsMenuItem[] = [
    ...(!restoreOk && isShown(item.access?.restore)
      ? [{ label: 'Restore', onSelect: onRestore, ...availabilityProps(item.access?.restore) }]
      : []),
    ...(isShown(item.access?.deletePermanently)
      ? [
          {
            label: 'Delete permanently',
            tone: 'danger' as const,
            onSelect: onPurge,
            ...availabilityProps(item.access?.deletePermanently),
          },
        ]
      : []),
  ];
  const [open, setOpen] = useState(highlightInside);
  const heading = useRef<HTMLHeadingElement>(null);
  const contentsId = useId();
  const wasIn = wasInText(item.path);
  const stateLine = restoreStateLine(item);
  const inside = insideRecords(item);
  const record = `${item.recordTypeLabel} '${item.recordName}'`;
  const deletedAt = dateTime(item.deletedAt);
  const deletedAgo = relativeTime(item.deletedAt);

  useEffect(() => {
    if (highlighted) {
      heading.current?.scrollIntoView?.({ block: 'center' });
      heading.current?.focus();
    }
  }, [highlighted]);

  const level = levelOf(item.recordType);

  return (
    <li className={styles.row} data-highlighted={highlighted || undefined}>
      <div className={styles.main}>
        <div className={styles.title}>
          {/* One tag style for every record type (038 FR-049). */}
          {level ? (
            <LevelTag level={level} />
          ) : (
            <span className={styles.tag}>{item.recordTypeLabel.toUpperCase()}</span>
          )}
          <h2 ref={heading} tabIndex={-1} className={styles.name}>
            {item.recordName}
          </h2>
          <span className={styles.id}>{item.recordId}</span>
        </div>
        {wasIn && <p className={styles.meta}>{wasIn}</p>}
        <p className={styles.meta}>{rowContentsText(item.contents)}</p>
        <p className={styles.deleted}>
          <Avatar initials={initialsOf(item.deletedBy.name)} size="sm" />
          <span>Deleted by {item.deletedBy.name}</span>
          <span aria-hidden="true">·</span>
          <Tooltip content={deletedAt}>
            <button
              type="button"
              className={styles.time}
              aria-label={`${deletedAgo}, ${deletedAt}`}
            >
              <time dateTime={item.deletedAt}>{deletedAgo}</time>
            </button>
          </Tooltip>
        </p>
        {stateLine && <p className={styles.state}>{stateLine}</p>}
        {containsText && <p className={styles.contains}>{containsText}</p>}
        {inside.length > 0 && (
          <>
            <button
              type="button"
              className={styles.toggle}
              aria-expanded={open}
              aria-controls={contentsId}
              onClick={() => setOpen(!open)}
            >
              {open ? 'Hide contents' : 'Show contents'}
            </button>
            <div id={contentsId} hidden={!open}>
              <ContentsTree records={inside} parentId={item.recordId} />
            </div>
          </>
        )}
      </div>
      <div className={styles.actions}>
        {restoreOk && (
          <Button size="sm" aria-label={`Restore ${record}`} onClick={onRestore}>
            Restore
          </Button>
        )}
        {menu.length > 0 && <ActionsMenu label={record} items={menu} />}
      </div>
    </li>
  );
}

/** The records inside, as a nested list by parent: "Activity 'Fleet expo stand' (A-1042)". */
function ContentsTree({ records, parentId }: { records: BinRecord[]; parentId: string }) {
  const children = records.filter((record) => record.parentId === parentId);
  if (children.length === 0) {
    return null;
  }
  return (
    <ul className={styles.tree}>
      {children.map((record) => (
        <li key={record.recordId}>
          {`${capitalise(record.recordTypeLabel)} '${record.recordName}' (${record.recordId})`}
          <ContentsTree records={records} parentId={record.recordId} />
        </li>
      ))}
    </ul>
  );
}

/** The LevelTag of a planning or purchase-request record; null for other record types. */
function levelOf(recordType: string): LevelTagProps['level'] | null {
  if (isPlanningType(recordType)) {
    return recordType;
  }
  return recordType === 'purchaseRequest' || recordType === 'purchaseRequestLine'
    ? recordType
    : null;
}

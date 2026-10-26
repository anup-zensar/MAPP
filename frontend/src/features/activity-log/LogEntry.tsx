import { useId, useState } from 'react';
import { Link } from 'react-router';
import { Avatar } from '@/ui/Avatar/Avatar';
import { StatusChip } from '@/ui/StatusChip/StatusChip';
import { Tooltip } from '@/ui/Tooltip/Tooltip';
import { formatTime } from '@/format/intl';
import { useFormatters } from '@/format/useFormatters';
import {
  ACTION_CHIPS,
  detailLines,
  initialsOf,
  pathText,
  recordLink,
  sentenceLead,
} from './logText';
import type { ActivityLogEntry } from './logText';
import styles from './LogEntry.module.css';

export interface LogEntryProps {
  entry: ActivityLogEntry;
}

/**
 * One activity-log entry (FR-005 – FR-008, FR-010): who did what to which record, its ID, where
 * the record was, the time (the full date in a tooltip), the action chip, and **Show details**
 * when there is more to say. Read-only: an entry has no edit or delete control.
 */
export function LogEntry({ entry }: LogEntryProps) {
  const { dateTime } = useFormatters();
  const [open, setOpen] = useState(false);
  const detailsId = useId();
  const chip = ACTION_CHIPS[entry.action];
  const details = detailLines(entry);
  const place = pathText(entry.path);
  const fullTime = dateTime(entry.at);

  return (
    <li className={styles.entry}>
      <Avatar initials={initialsOf(entry.actor.name)} size="sm" />
      <div className={styles.body}>
        <p className={styles.sentence}>
          {sentenceLead(entry)}
          {entry.action !== 'emptiedRecycleBin' && (
            <>
              {' '}
              <RecordName entry={entry} />
              <span className={styles.id}>{entry.recordId}</span>
            </>
          )}
        </p>
        {place && <p className={styles.meta}>{place}</p>}
        {details.length > 0 && (
          <>
            <button
              type="button"
              className={styles.toggle}
              aria-expanded={open}
              aria-controls={detailsId}
              onClick={() => setOpen(!open)}
            >
              {open ? 'Hide details' : 'Show details'}
            </button>
            <ul id={detailsId} className={styles.details} hidden={!open}>
              {details.map((line, index) => (
                <li key={index}>{line}</li>
              ))}
            </ul>
          </>
        )}
      </div>
      <div className={styles.side}>
        {/* A button, so keyboard and screen-reader users reach the full date too. */}
        <Tooltip content={fullTime}>
          <button type="button" className={styles.time} aria-label={fullTime}>
            <time dateTime={entry.at}>{formatTime(entry.at)}</time>
          </button>
        </Tooltip>
        <StatusChip tone={chip.tone}>{chip.word}</StatusChip>
      </div>
    </li>
  );
}

/**
 * The record's name as it was then (FR-008): a link to its page (with " — now '<name>'" when
 * renamed), a link to its row in the recycle bin, or plain text.
 */
function RecordName({ entry }: { entry: ActivityLogEntry }) {
  const link = recordLink(entry);
  switch (link.kind) {
    case 'record':
      return (
        <>
          &apos;
          <Link className={styles.link} to={link.href}>
            {entry.recordName}
          </Link>
          &apos;{link.now}
        </>
      );
    case 'recycleBin':
      return (
        <>
          &apos;
          <Link className={styles.link} to={link.href}>
            {entry.recordName}
          </Link>
          &apos; (in the recycle bin)
        </>
      );
    case 'text':
      return (
        <>
          <span className={styles.name}>&apos;{entry.recordName}&apos;</span>
          {link.suffix}
        </>
      );
  }
}

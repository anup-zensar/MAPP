import type { RecycleBinEntry, SampleFolder } from './api';

/** Joins "2 sample notes" and "1 campaign" as "2 sample notes and 1 campaign". */
const AND_LIST = new Intl.ListFormat('en-GB', { type: 'conjunction' });

/** What a folder takes with it to the recycle bin (plan "User-visible messages"). */
export function whatGoesWith(folder: SampleFolder): string {
  const count = folder.notes.length;
  if (count === 0) {
    return 'It goes to the recycle bin. You can restore it.';
  }
  const notes = count === 1 ? 'its sample note' : `its ${count} sample notes`;
  return `It goes to the recycle bin together with ${notes}. You can restore it.`;
}

/**
 * "Sample folder 'Ideas' (SF-1001), with 2 sample notes — deleted by Priya Raman, 12 Mar 2026,
 * 14:32".
 */
export function deletedText(
  entry: RecycleBinEntry,
  dateTime: (timestamp: string) => string,
): string {
  const contents = entry.contents.map((content) => `${content.count} ${content.label}`);
  const withContents = contents.length > 0 ? `, with ${AND_LIST.format(contents)}` : '';
  return (
    `${recordTitle(entry)} (${entry.recordId})${withContents}` +
    ` — deleted by ${entry.deletedBy.name}, ${dateTime(entry.deletedAt)}`
  );
}

/**
 * "Sample folder 'Ideas'": the record type, with a capital letter as it starts the sentence
 * (data model §5), and the record's name.
 */
export function recordTitle(entry: RecycleBinEntry): string {
  const recordType = entry.recordTypeLabel;
  return `${recordType.charAt(0).toUpperCase()}${recordType.slice(1)} '${entry.recordName}'`;
}

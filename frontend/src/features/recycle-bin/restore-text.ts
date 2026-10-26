import type { components } from '@/api/generated/schema';
import { isPlanningType, recordHref } from '@/ui/record-links';
import { contentsText, titled } from './bin-text';
import type { BinItem, RecordRef } from './bin-text';

export type RestorePreview = components['schemas']['RestorePreview'];
export type RestorePreviewChange = components['schemas']['RestorePreviewChange'];
type LabelledRef = components['schemas']['LabelledRef'];

/** "Restore initiative 'Trade shows'?" */
export function restoreTitle(item: BinItem): string {
  return `Restore ${item.recordTypeLabel} '${item.recordName}'?`;
}

/** "Restore initiative", "Restore both" or "Restore all 3" (FR-023). */
export function restoreConfirmLabel(item: BinItem, parentEntries: number): string {
  if (parentEntries === 0) {
    return `Restore ${item.recordTypeLabel}`;
  }
  return parentEntries === 1 ? 'Restore both' : `Restore all ${parentEntries + 1}`;
}

/**
 * Where it returns (FR-018): "It returns to FY2026 Lubricants Europe › Helix launch with 2
 * activities and 5 line items."; at the top: "It returns to the Plans list" (plans) or "It
 * returns to its list".
 */
export function destinationSentence(preview: RestorePreview): string {
  const { entry, destination } = preview;
  const place =
    destination.length > 0
      ? destination.map((record) => record.name).join(' › ')
      : entry.recordType === 'plan'
        ? 'the Plans list'
        : 'its list';
  const contents = contentsText(entry.contents);
  return `It returns to ${place}${contents ? ` with ${contents}` : ''}.`;
}

/** One sentence per change the restore makes (plan "Restore sentences", FR-019, FR-020, FR-022). */
export function changeSentence(change: RestorePreviewChange, entry: BinItem): string {
  const subject =
    change.recordId === entry.recordId
      ? 'It'
      : `${titled(change.recordTypeLabel, change.recordName)} (${change.recordId})`;
  if (change.field === 'status') {
    if (change.restored && change.recordType === 'activity') {
      return `${subject} comes back as ${change.newText} and must be submitted for approval again.`;
    }
    if (change.restored && change.recordType === 'plan') {
      return `${subject} comes back as ${change.newText} and must be published again.`;
    }
    if (!change.restored && change.recordType === 'activity') {
      return `${subject} goes back to ${change.newText} because its line items change, and must be submitted for approval again.`;
    }
    return `${subject} changes from ${change.oldText} to ${change.newText}.`;
  }
  if (change.field === 'name') {
    return `${subject} comes back as '${change.newText}' because the name '${change.oldText}' is taken.`;
  }
  return `${subject}: ${change.label} changes from ${change.oldText} to ${change.newText}.`;
}

/** "Its initiative 'Trade shows' is also in the recycle bin. Restoring brings these back first:" */
export function parentsIntro(parent: LabelledRef): string {
  return `Its ${parent.typeLabel} '${parent.name}' is also in the recycle bin. Restoring brings these back first:`;
}

/** "Initiative 'Trade shows' (I-1009) with 1 activity and 1 line item" (FR-023). */
export function parentEntryLine(entry: BinItem): string {
  const contents = contentsText(entry.contents);
  return `${titled(entry.recordTypeLabel, entry.recordName)} (${entry.recordId})${contents ? ` with ${contents}` : ''}`;
}

/** "This activity can't be restored" (FR-024). */
export function blockedTitle(item: BinItem): string {
  return `This ${item.recordTypeLabel} can't be restored`;
}

/** Why, and what can be done instead (FR-024). */
export function blockedText(item: BinItem): string {
  const parent = item.missingParent;
  const because = parent
    ? ` because its ${parent.typeLabel} '${parent.name}' was permanently deleted`
    : '';
  return `This ${item.recordTypeLabel} can't be restored${because}. You can delete it permanently instead.`;
}

/** "Initiative 'Trade shows' restored" — the message with View (FR-026). */
export function restoredMessage(typeLabel: string, name: string): string {
  return `${titled(typeLabel, name)} restored`;
}

/**
 * The page of a restored record, for the message's View: a planning record's page, inside the
 * plan it returned to, or a global budget's page (014 FR-035); undefined for records without a page.
 */
export function restoredHref(
  recordType: string,
  recordId: string,
  destination: RecordRef[],
): string | undefined {
  if (recordType === 'globalBudget') {
    return `/budgets/${encodeURIComponent(recordId)}`;
  }
  if (!isPlanningType(recordType)) {
    return undefined;
  }
  const planId = destination.find((record) => record.type === 'plan')?.id;
  if ((recordType === 'campaign' || recordType === 'initiative') && !planId) {
    return undefined;
  }
  return recordHref(recordType, recordId, planId);
}

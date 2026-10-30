import { describe, expect, it } from 'vitest';
import { logEntry } from '@/testing/fixtures/activity-log';
import {
  countLine,
  dayLabel,
  detailLines,
  entrySentence,
  initialsOf,
  loadedLine,
  pathText,
  recordLink,
  sentenceLead,
} from './log-text';

describe('entry sentence (FR-005, FR-006)', () => {
  it('names the person, the verb, the record type and the record', () => {
    expect(entrySentence(logEntry({ verb: 'submitted' }))).toBe(
      "Priya Raman submitted activity 'Film shoot'",
    );
    expect(sentenceLead(logEntry())).toBe('Priya Raman changed activity');
  });

  it('counts the items of an emptied recycle bin, singular for one', () => {
    const item = {
      recordType: 'initiative',
      recordTypeLabel: 'initiative',
      recordId: 'I-1009',
      recordName: 'Trade shows',
      details: '',
    };
    const emptied = logEntry({ action: 'emptiedRecycleBin', verb: 'emptied', items: [item, item] });

    expect(entrySentence(emptied)).toBe('Priya Raman emptied the recycle bin (2 items)');
    expect(entrySentence({ ...emptied, items: [item] })).toBe(
      'Priya Raman emptied the recycle bin (1 item)',
    );
  });

  it('says where the record was at the time, nothing for a top-level record', () => {
    expect(pathText(logEntry().path)).toBe('in FY2026 Lubricants Europe › Helix launch');
    expect(pathText([])).toBe('');
  });

  it('takes the avatar initials from the person’s name', () => {
    expect(initialsOf('Priya Raman')).toBe('PR');
    expect(initialsOf('Ana Maria de Souza')).toBe('AS');
    expect(initialsOf('Cher')).toBe('C');
  });
});

describe('details (FR-007)', () => {
  it('lists each change as old → new, then the comment', () => {
    const entry = logEntry({
      changes: [
        {
          field: 'plannedCost',
          label: 'Planned cost',
          old: '40000',
          new: '45000',
          oldText: '40,000.00 USD',
          newText: '45,000.00 USD',
        },
      ],
      comment: 'Budget agreed with finance',
    });

    expect(detailLines(entry)).toEqual([
      'Planned cost: 40,000.00 USD → 45,000.00 USD',
      'Comment: Budget agreed with finance',
    ]);
  });

  it('says what went with a deleted record, and other details as stored', () => {
    expect(detailLines(logEntry({ action: 'deleted', details: 'with 2 line items' }))).toEqual([
      'Deleted with 2 line items',
    ]);
    expect(detailLines(logEntry({ action: 'copied', details: "Copied from 'Film'" }))).toEqual([
      "Copied from 'Film'",
    ]);
  });

  it('lists every item of an emptied recycle bin', () => {
    const entry = logEntry({
      action: 'emptiedRecycleBin',
      items: [
        {
          recordType: 'initiative',
          recordTypeLabel: 'initiative',
          recordId: 'I-1009',
          recordName: 'Trade shows',
          details: 'with 1 activity and 1 line item',
        },
        {
          recordType: 'lineItem',
          recordTypeLabel: 'line item',
          recordId: 'L-1004',
          recordName: 'Print',
          details: '',
        },
      ],
    });

    expect(detailLines(entry)).toEqual([
      "Initiative 'Trade shows' (I-1009) with 1 activity and 1 line item",
      "Line item 'Print' (L-1004)",
    ]);
  });

  it('has nothing to show for a plain entry', () => {
    expect(detailLines(logEntry())).toEqual([]);
  });
});

describe('record link (FR-008)', () => {
  it('links an existing record to its page, inside its plan', () => {
    expect(recordLink(logEntry())).toEqual({
      kind: 'record',
      href: '/plans/P-1001/activities/A-1042',
      now: '',
    });
    expect(
      recordLink(logEntry({ recordType: 'initiative', recordId: 'I-1003', recordName: 'Helix' })),
    ).toMatchObject({ href: '/plans/P-1001/initiatives/I-1003' });
  });

  it('adds the current name when the record was renamed', () => {
    const entry = logEntry({
      recordNow: { state: 'exists', name: 'Film shoot (final)', path: [], binEntryId: '' },
    });

    expect(recordLink(entry)).toEqual({
      kind: 'record',
      href: '/activities/A-1042',
      now: " — now 'Film shoot (final)'",
    });
  });

  it('opens the recycle bin with the record highlighted', () => {
    const entry = logEntry({
      recordNow: { state: 'inRecycleBin', name: 'Film shoot', path: [], binEntryId: 'BIN-1003' },
    });

    expect(recordLink(entry)).toEqual({ kind: 'recycleBin', href: '/recycle-bin?record=A-1042' });
  });

  it('shows a permanently deleted record as plain text', () => {
    const entry = logEntry({
      recordNow: { state: 'deletedPermanently', name: '', path: [], binEntryId: '' },
    });

    expect(recordLink(entry)).toEqual({ kind: 'text', suffix: ' (permanently deleted)' });
  });

  it('links global budgets (014) and GSAP transactions (015) to their pages', () => {
    const exists = { state: 'exists' as const, name: '', path: [], binEntryId: '' };
    expect(
      recordLink(
        logEntry({
          recordType: 'globalBudget',
          recordId: 'GB-1001',
          recordName: 'FY2026 Global marketing budget',
          recordNow: { ...exists, name: 'FY2026 Global marketing budget' },
        }),
      ),
    ).toEqual({ kind: 'record', href: '/budgets/GB-1001', now: '' });
    expect(
      recordLink(
        logEntry({
          recordType: 'financialTransaction',
          recordId: 'FT-1004',
          recordName: 'PO 4500001004 / 10',
          recordNow: { ...exists, name: 'PO 4500001004 / 10' },
        }),
      ),
    ).toEqual({ kind: 'record', href: '/financials/transactions/FT-1004', now: '' });
  });

  it('shows records without a page, or without a known state, as plain text', () => {
    expect(
      recordLink(
        logEntry({
          recordType: 'currency',
          recordNow: { state: 'exists', name: 'Euro', path: [], binEntryId: '' },
        }),
      ),
    ).toEqual({ kind: 'text', suffix: " — now 'Euro'" });
    expect(recordLink(logEntry({ recordNow: undefined }))).toEqual({ kind: 'text', suffix: '' });
    expect(
      recordLink(logEntry({ recordNow: { state: 'none', name: '', path: [], binEntryId: '' } })),
    ).toEqual({ kind: 'text', suffix: '' });
  });
});

describe('day headings and counts (FR-004, FR-003, FR-009)', () => {
  it('names today, yesterday, and other days with their weekday', () => {
    expect(dayLabel('2026-10-19', '2026-10-19')).toBe('Today');
    expect(dayLabel('2026-10-18', '2026-10-19')).toBe('Yesterday');
    expect(dayLabel('2026-10-12', '2026-10-19')).toBe('Monday 12 Oct 2026');
    expect(dayLabel('2026-02-28', '2026-03-01')).toBe('Yesterday');
  });

  it("038 FR-058: writes the day in the person's date format; Today and Yesterday stay words", () => {
    expect(dayLabel('2026-10-27', '2026-11-02', 'YYYY-MM-DD')).toBe('Tuesday 2026-10-27');
    expect(dayLabel('2026-10-27', '2026-11-02', 'DD/MM/YYYY')).toBe('Tuesday 27/10/2026');
    expect(dayLabel('2026-11-02', '2026-11-02', 'YYYY-MM-DD')).toBe('Today');
    expect(dayLabel('2026-11-01', '2026-11-02', 'MM/DD/YYYY')).toBe('Yesterday');
  });

  it('counts all entries, or the matching ones of all while a filter is in use', () => {
    expect(countLine(1204, 1204)).toBe('1,204 of 1,204 entries');
    expect(countLine(1, 1)).toBe('1 of 1 entry');
    expect(countLine(12, 1204)).toBe('12 of 1,204 entries');
  });

  it('says how many entries are loaded of all', () => {
    expect(loadedLine(50, 1204)).toBe('Showing the last 50 of 1,204');
  });
});

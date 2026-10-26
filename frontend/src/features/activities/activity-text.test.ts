import { describe, expect, it } from 'vitest';
import { formatDate, formatDateRange } from '@/format/intl';
import { activityBannerText } from '@/ui/ActivityStatusChip/activity-status';
import { activityEditHref, activityHref, datesText, newActivityHref } from './activity-text';

describe('activityText (feature 006)', () => {
  it('writes the dates as every date range reads, ongoing ones from their start, and "—" without both (D-216)', () => {
    const format = { date: formatDate, dateRange: formatDateRange };
    expect(datesText('2026-03-01', '2026-05-31', false, format)).toBe('01 Mar – 31 May 2026');
    expect(datesText('2026-12-01', '2027-01-31', false, format)).toBe('01 Dec 2026 – 31 Jan 2027');
    expect(datesText('2026-03-01', null, true, format)).toBe('Ongoing from 01 Mar 2026');
    expect(datesText('2026-03-01', null, false, format)).toBe('—');
    expect(datesText(null, null, false, format)).toBe('—');
    const daysFirst = {
      date: (date: string) => formatDate(date, 'DD/MM/YYYY'),
      dateRange: (start: string | null | undefined, end: string | null | undefined) =>
        formatDateRange(start, end, 'DD/MM/YYYY'),
    };
    expect(datesText('2026-03-01', '2026-05-31', false, daysFirst)).toBe('01/03/2026 – 31/05/2026');
  });

  it('gives the addresses of the activity pages', () => {
    expect(activityHref('P-1001', 'A-1042')).toBe('/plans/P-1001/activities/A-1042');
    expect(activityEditHref('P-1001', 'A-1042')).toBe('/plans/P-1001/activities/A-1042/edit');
    expect(activityEditHref('P-1001', 'A-1042', 'ownerId')).toBe(
      '/plans/P-1001/activities/A-1042/edit?field=ownerId',
    );
    expect(newActivityHref('P-1001', 'I-1001')).toBe(
      '/plans/P-1001/initiatives/I-1001/activities/new',
    );
  });
});

describe('activityBannerText (feature 006, FR-027)', () => {
  it.each([
    ['submitted', 'Submitted for approval — changing planned cost or dates will restart approval.'],
    ['approved', 'Approved — changing planned cost or dates will restart approval.'],
    ['inProgress', 'In progress — changing planned cost or dates will restart approval.'],
    ['complete', 'Complete — changing planned cost or dates will restart approval.'],
  ] as const)('says a %s activity restarts approval on a cost or date change', (status, text) => {
    expect(activityBannerText(status)).toBe(text);
  });

  it('has no banner for Created and Declined', () => {
    expect(activityBannerText('created')).toBeNull();
    expect(activityBannerText('declined')).toBeNull();
  });
});

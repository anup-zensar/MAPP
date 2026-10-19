import { describe, expect, it } from 'vitest';
import { formatDate } from '@/lib/format';
import { activityBannerText } from './activityStatus';
import {
  activityEditHref,
  activityHref,
  datesText,
  lineItemsText,
  newActivityHref,
  thingsText,
} from './activityText';

describe('activityText (feature 006)', () => {
  it('writes the dates in full, ongoing ones from their start, and "—" without both', () => {
    expect(datesText('2026-03-01', '2026-05-31', false, formatDate)).toBe(
      '01 Mar 2026 – 31 May 2026',
    );
    expect(datesText('2026-03-01', null, true, formatDate)).toBe('Ongoing from 01 Mar 2026');
    expect(datesText('2026-03-01', null, false, formatDate)).toBe('—');
    expect(datesText(null, null, false, formatDate)).toBe('—');
  });

  it('counts things and line items', () => {
    expect(thingsText(1)).toBe('1 thing');
    expect(thingsText(5)).toBe('5 things');
    expect(lineItemsText(1)).toBe('1 line item');
    expect(lineItemsText(3)).toBe('3 line items');
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

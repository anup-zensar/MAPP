import fc from 'fast-check';
import { describe, expect, it } from 'vitest';
import { safeNext } from './safe-next';

/** The page reads "next" the way the sign-in page does: decoded by URLSearchParams. */
function nextFrom(query: string): string | null {
  return new URLSearchParams(query).get('next');
}

describe('safeNext', () => {
  it('keeps an internal path with its query, so people return where they were (FR-005)', () => {
    expect(safeNext('/plans?status=draft&year=2026')).toBe('/plans?status=draft&year=2026');
    expect(safeNext('/settings#currencies')).toBe('/settings#currencies');
  });

  it('turns a full address of SP&A One itself into its path (FR-005)', () => {
    expect(safeNext(`${window.location.origin}/approvals`)).toBe('/approvals');
  });

  it('goes Home when there is nothing to go back to (FR-005)', () => {
    expect(safeNext(null)).toBe('/');
    expect(safeNext(undefined)).toBe('/');
    expect(safeNext('')).toBe('/');
  });

  it.each([
    ['//evil.com'],
    ['/\\evil.com'],
    ['https://evil.com'],
    ['javascript:alert(1)'],
    ['/.//evil.com'],
  ])('never sends anyone to another site: %s → / (FR-005)', (raw) => {
    expect(safeNext(raw)).toBe('/');
  });

  it('refuses the decoded forms of /%5Cevil.com and /%09/evil.com (FR-005)', () => {
    expect(safeNext(nextFrom('next=/%5Cevil.com'))).toBe('/');
    expect(safeNext(nextFrom('next=/%09/evil.com'))).toBe('/');
  });

  it.each([['/sign-in?next=/plans'], ['/sign-in/'], ['/SIGN-IN']])(
    'never goes back to the sign-in page itself: %s → / (FR-005)',
    (raw) => {
      expect(safeNext(raw)).toBe('/');
    },
  );

  it('sends any address only to a page of SP&A One other than sign-in (FR-005)', () => {
    fc.assert(
      fc.property(fc.oneof(fc.string(), fc.webUrl(), fc.webPath()), (raw) => {
        const next = new URL(safeNext(raw), window.location.origin);
        expect(next.origin).toBe(window.location.origin);
        expect(next.pathname.toLowerCase().replace(/\/+$/, '')).not.toBe('/sign-in');
      }),
    );
  });
});

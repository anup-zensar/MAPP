import { describe, expect, it } from 'vitest';
import { accruedLabel, accruedTip, monthText } from './financials-text';

describe('Home financials texts', () => {
  it("038 FR-058: writes the accrual month in the person's date format", () => {
    expect(monthText('2026-09-30')).toBe('Sep 2026');
    expect(monthText('2026-09', 'DD/MM/YYYY')).toBe('09/2026');
    expect(accruedLabel('2026-09', 'YYYY-MM-DD')).toBe('Accrued (2026-09)');
    expect(accruedTip('2026-09', 'MM/DD/YYYY')).toContain('for 09/2026.');
    expect(accruedLabel(null)).toBe('Accrued');
  });
});

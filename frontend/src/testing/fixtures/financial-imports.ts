import type { FinancialImport, FinancialImportCheck } from '@/features/financials';

/** The import of the demo data: GSAP's feed at 02:00, by MAPP itself. */
export const SYSTEM = { id: 'system', name: 'MAPP' };

export const DEMO_IMPORT: FinancialImport = {
  id: 'FIM-1001',
  source: 'demo',
  feedName: 'Demo data (simulated GSAP)',
  fileName: '',
  importedAt: '2026-10-19T02:00:00Z',
  importedBy: SYSTEM,
  counts: { rows: 11, new: 11, updated: 0, unchanged: 0, refused: 0, notLinked: 1, relinked: 0 },
};

/** What the check of a 4-row file says (US4 scenario 2). */
export function checkResult(overrides: Partial<FinancialImportCheck> = {}): FinancialImportCheck {
  return {
    fileName: 'gsap-2026-10-19.csv',
    counts: { rows: 4, new: 2, updated: 1, unchanged: 1, refused: 0, notLinked: 1, relinked: 0 },
    problems: [],
    problemsTotal: 0,
    notLinked: [
      {
        row: 5,
        name: 'PO 4500001009 / 10',
        reason: 'No line item for HX key HX1099 — check the tracking key in GSAP.',
      },
    ],
    notLinkedTotal: 1,
    unusedColumns: [],
    ...overrides,
  };
}

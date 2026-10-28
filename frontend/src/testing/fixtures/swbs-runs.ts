import type { components } from '@/api/generated/schema';

type SwbsRun = components['schemas']['SwbsRun'];
type SwbsRunPage = components['schemas']['SwbsRunPage'];

export const DEMO_RUN: SwbsRun = {
  id: 'SWR-1001',
  source: 'demoData',
  ranAt: '2026-10-19T09:00:00Z',
  ranBy: { id: 'system', name: 'MAPP' },
  sent: 1,
  codes: [
    {
      lineItemId: 'L-1003',
      lineItemName: 'Venue hire',
      companyCode: 'DE03',
      swbsCode: 'SIM-DE03-0001',
    },
  ],
};

export const ALEX_RUN: SwbsRun = {
  id: 'SWR-1002',
  source: 'simulatedGsap',
  ranAt: '2026-10-20T14:05:00Z',
  ranBy: { id: 'U-1001', name: 'Alex Morgan' },
  sent: 2,
  codes: [
    {
      lineItemId: 'L-1005',
      lineItemName: 'Media booking',
      companyCode: 'NL02',
      swbsCode: 'SIM-NL02-0001',
    },
    {
      lineItemId: 'L-1006',
      lineItemName: 'Stand build',
      companyCode: 'GB01',
      swbsCode: 'SIM-GB01-0001',
    },
  ],
};

export const EMPTY_RUN: SwbsRun = { ...ALEX_RUN, id: 'SWR-1003', sent: 0, codes: [] };

export function runsPage(overrides: Partial<SwbsRunPage> = {}): SwbsRunPage {
  return { items: [DEMO_RUN], total: 1, readyToSend: 1, ...overrides };
}

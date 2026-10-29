/**
 * The SP&A view of Shell's worked example (013 spec US1): Activity A 340,000.00 and Activity B 60,000.00
 * planned, 200,000.00 committed and 150,000.00 paid on A in Q3, in US dollars.
 */
import type { SpaView } from '@/features/spa';

const usd = (amount: string) => ({ amount, currency: 'USD' });
const A = { activityId: 'A-1001', name: 'Launch film' };
const B = { activityId: 'A-1002', name: 'Social always-on' };
const HELIX = { type: 'campaign' as const, id: 'C-1001', name: 'Helix launch' };

function quarter(
  q: number,
  planned: string,
  actual: string,
): SpaView['periods']['quarters'][number] {
  return {
    key: `2026-Q${q}`,
    label: `Q${q} 2026`,
    planned: usd(planned),
    actual: usd(actual),
    plannedBar: planned === '0.00' ? 0 : 100,
    actualBar: actual === '0.00' ? 0 : 75,
    plannedTrace: planned === '0.00' ? [] : [{ ...A, amount: usd(planned) }],
    actualTrace: actual === '0.00' ? [] : [{ ...A, amount: usd(actual) }],
  };
}

export function spaView(overrides: Partial<SpaView> = {}): SpaView {
  return {
    plan: {
      id: 'P-1001',
      name: 'FY2026 Lubricants Europe',
      year: 2026,
      status: 'published',
      bcpir: true,
    },
    scope: { type: 'plan', id: 'P-1001', name: 'FY2026 Lubricants Europe' },
    campaigns: [HELIX, { type: 'campaign', id: 'C-1002', name: 'Rimula fleet' }],
    currency: 'USD',
    financeAccess: true,
    financeNote: null,
    financeDataAsOf: '2026-10-21T14:05:00Z',
    headline: {
      plannedCost: usd('400000.00'),
      committed: usd('200000.00'),
      actualSpend: usd('150000.00'),
      invoiced: usd('120000.00'),
      remaining: usd('250000.00'),
      utilisationPercent: '37.5',
      variancePercent: '-62.5',
      overspend: false,
      noPlannedCostReason: null,
      plannedCostTrace: [
        { ...A, amount: usd('340000.00') },
        { ...B, amount: usd('60000.00') },
      ],
      committedTrace: [{ ...A, amount: usd('200000.00') }],
      actualSpendTrace: [{ ...A, amount: usd('150000.00') }],
    },
    periods: {
      quarters: [
        quarter(1, '0.00', '0.00'),
        quarter(2, '0.00', '0.00'),
        quarter(3, '200000.00', '150000.00'),
        quarter(4, '200000.00', '0.00'),
      ],
      months: Array.from({ length: 12 }, (_, i) => ({
        ...quarter(1, i === 6 ? '200000.00' : '0.00', i === 7 ? '150000.00' : '0.00'),
        key: `2026-${String(i + 1).padStart(2, '0')}`,
        label: `${['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'][i] ?? ''} 2026`,
      })),
      notScheduled: null,
      notScheduledTrace: [],
    },
    dimension: {
      key: 'attr:SA-1013',
      label: 'Country / OU',
      rows: [
        {
          key: 'value:China',
          label: 'China',
          amount: usd('264000.00'),
          share: '66.00',
          bar: 100,
          trace: [
            { ...A, amount: usd('204000.00') },
            { ...B, amount: usd('60000.00') },
          ],
        },
        {
          key: 'value:India',
          label: 'India',
          amount: usd('136000.00'),
          share: '34.00',
          bar: 52,
          trace: [{ ...A, amount: usd('136000.00') }],
        },
      ],
      total: usd('400000.00'),
      countsInEach: false,
    },
    dimensions: [
      { key: 'attr:SA-1012', label: 'Channel / audience' },
      { key: 'activityType', label: 'Activity type' },
      { key: 'attr:SA-1013', label: 'Country / OU' },
    ],
    activities: [
      {
        id: 'A-1001',
        name: 'Launch film',
        campaign: HELIX,
        initiative: { type: 'initiative', id: 'I-1001', name: 'Digital & social' },
        owner: 'Priya Raman',
        activityType: 'Content production',
        channel: 'External',
        startQuarter: 'Q3 2026',
        status: 'approved',
        plannedCost: usd('340000.00'),
        committed: usd('200000.00'),
        actualSpend: usd('150000.00'),
        variancePercent: '-55.9',
        overspend: false,
        lineItems: [
          {
            id: 'L-1001',
            name: 'Agency fee',
            plannedCost: usd('200000.00'),
            committed: usd('200000.00'),
            actualSpend: usd('150000.00'),
          },
          {
            id: 'L-1002',
            name: 'Film production',
            plannedCost: usd('140000.00'),
            committed: usd('0.00'),
            actualSpend: usd('0.00'),
          },
        ],
        raisePurchaseRequest: { available: true, reason: '' },
      },
      {
        id: 'A-1002',
        name: 'Social always-on',
        campaign: HELIX,
        initiative: { type: 'initiative', id: 'I-1001', name: 'Digital & social' },
        owner: null,
        activityType: 'Digital media',
        channel: null,
        startQuarter: 'Q4 2026',
        status: 'created',
        plannedCost: usd('60000.00'),
        committed: usd('0.00'),
        actualSpend: usd('0.00'),
        variancePercent: '-100.0',
        overspend: false,
        lineItems: [],
        raisePurchaseRequest: {
          available: false,
          reason: 'Add a line item to this activity first.',
        },
      },
    ],
    budget: {
      budget: usd('2200000.00'),
      plannedCost: usd('400000.00'),
      committed: usd('200000.00'),
      plannedVsBudgetPercent: '-81.8',
      overBudget: false,
      noBudgetReason: null,
      linked: usd('300000.00'),
      notLinked: usd('100000.00'),
    },
    procurement: {
      count: 2,
      approved: usd('200000.00'),
      pendingApproval: usd('0.00'),
      draft: usd('10000.00'),
      invoiced: usd('120000.00'),
      items: [
        {
          id: 'PR-1001',
          name: 'Agency retainer',
          vendor: 'Brightside Media BV',
          status: 'approved',
          amount: usd('200000.00'),
          counted: true,
        },
        {
          id: 'PR-1002',
          name: 'Old print run',
          vendor: '',
          status: 'declined',
          amount: usd('7.00'),
          counted: false,
        },
      ],
    },
    notes: [],
    emptyMessage: null,
    ...overrides,
  };
}

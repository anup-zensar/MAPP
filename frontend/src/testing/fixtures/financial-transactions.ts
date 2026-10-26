import type { components } from '@/api/generated/schema';
import type {
  FinancialTransaction,
  FinancialTransactionRow,
  LineItemPreview,
} from '@/features/financials';
import { DEMO_IMPORT, SYSTEM } from './financial-imports';
import { FT_1004 } from './financials';

type RecordRef = components['schemas']['RecordRef'];

const usd = (amount: string) => ({ amount, currency: 'USD' });
const ALEX = { id: 'U-1001', name: 'Alex Morgan' };
const MEI = { id: 'U-1004', name: 'Mei Lin', initials: 'ML', jobTitle: 'Marketer', active: true };
const GRAND_OPENING_PATH: RecordRef[] = [
  { type: 'plan', id: 'P-1002', name: 'FY2026 Mobility China' },
  { type: 'campaign', id: 'C-1003', name: 'Retail growth' },
  { type: 'initiative', id: 'I-1005', name: 'Station promotions' },
];

/** A transaction with the panel's fields, from one of the rows above. */
export function transactionOf(
  source: FinancialTransactionRow,
  overrides: Partial<FinancialTransaction> = {},
): FinancialTransaction {
  const notLinked = source.link.state === 'notLinked';
  return {
    ...source,
    amountUsd: source.amountDisplay,
    trackingKey:
      source.type === 'purchaseOrder'
        ? `HX${source.id === 'FT-1004' ? '1099' : (source.link.lineItemId ?? '').slice(2)}`
        : '',
    sentLineItemId: '',
    import: {
      id: DEMO_IMPORT.id,
      feedName: DEMO_IMPORT.feedName,
      fileName: '',
      importedAt: DEMO_IMPORT.importedAt,
      importedBy: SYSTEM,
    },
    canLink: notLinked,
    canUnlink: false,
    version: 1,
    createdAt: '2026-10-19T02:00:00Z',
    updatedAt: '2026-10-19T02:00:00Z',
    updatedBy: SYSTEM,
    ...overrides,
  };
}

/** FT-1004 after Alex Morgan linked it to L-1007 by hand (US7 scenario 1). */
export function manuallyLinked(): FinancialTransaction {
  return transactionOf(FT_1004, {
    link: {
      state: 'linked',
      matchedBy: 'manual',
      reason: null,
      lineItemId: 'L-1007',
      path: [
        { type: 'plan', id: 'P-1002', name: 'FY2026 Mobility China' },
        { type: 'campaign', id: 'C-1003', name: 'Retail growth' },
        { type: 'initiative', id: 'I-1005', name: 'Station promotions' },
        { type: 'activity', id: 'A-1008', name: 'Grand opening Shanghai' },
        { type: 'lineItem', id: 'L-1007', name: 'Station signage' },
      ],
      manualBy: { id: 'U-1001', name: 'Alex Morgan' },
      manualAt: '2026-10-20T09:00:00Z',
    },
    canLink: false,
    canUnlink: true,
    version: 2,
  });
}

/** GET /api/v1/line-items/L-1007 for the link dialog's preview: 12,000.00 USD of signage. */
export const STATION_SIGNAGE: LineItemPreview = {
  lineItem: {
    id: 'L-1007',
    activityId: 'A-1008',
    name: 'Station signage',
    owner: MEI,
    description: 'Signage for the 12 stations of the opening.',
    currency: { code: 'USD', name: 'US dollar', symbol: '$', ratePerUsd: '1', active: true },
    plannedCost: usd('12000.00'),
    plannedCostUsd: usd('12000.00'),
    plannedCostDisplay: usd('12000.00'),
    spread: {
      method: 'firstMonth',
      state: 'spread',
      months: [
        {
          month: '2026-05',
          amount: usd('12000.00'),
          amountUsd: usd('12000.00'),
          outsideActivityDates: false,
        },
      ],
      customTotal: null,
      addsUp: true,
    },
    attributes: {},
    contractedAmount: null,
    requestSwbs: false,
    swbsStatus: 'notRequested',
    financeBudgetYear: 2026,
    financeBudgetItem: null,
    missing: [],
    version: 1,
    createdAt: '2026-04-02T08:00:00Z',
    createdBy: ALEX,
    updatedAt: '2026-04-02T08:00:00Z',
    updatedBy: ALEX,
  },
  activity: {
    id: 'A-1008',
    name: 'Grand opening Shanghai',
    status: 'approved',
    restartsApproval: true,
    startDate: '2026-05-01',
    endDate: '2026-05-31',
    ongoing: false,
    plannedCost: usd('40000.00'),
    path: GRAND_OPENING_PATH,
    planYear: 2026,
    defaultSpreadMethod: 'evenly',
    defaultFinanceBudget: null,
    businessUnitOptionIds: [],
  },
  balance: {
    plannedCost: usd('40000.00'),
    assigned: usd('12000.00'),
    balance: usd('28000.00'),
    state: 'left',
    roundingHint: false,
  },
};

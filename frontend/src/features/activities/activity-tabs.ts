import type { TabItem } from '@/ui/Tabs/Tabs';
import type { ActivityTab } from './schemas';

/**
 * The tabs of an activity's page (design system §2.4 B, plan decision 5): **Overview** (the
 * working view: Ready to submit, Financials and the line items), **Details** (details, Shell
 * attributes, goals), **Approval** and **History**. The chosen tab is kept in the address
 * (`?tab=details`); Overview has no parameter.
 */
export const ACTIVITY_TABS: TabItem<ActivityTab>[] = [
  { value: 'overview', label: 'Overview' },
  { value: 'details', label: 'Details' },
  { value: 'approval', label: 'Approval' },
  { value: 'history', label: 'History' },
];

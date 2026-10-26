import type { ActivityDestination } from '@/features/activities';

/** "Print & outdoor" (1 Jan – 31 Dec 2026) in "Helix launch": Launch film may move or copy there. */
export function destination(overrides: Partial<ActivityDestination> = {}): ActivityDestination {
  return {
    campaign: { type: 'campaign', id: 'C-1001', name: 'Helix launch' },
    initiative: { type: 'initiative', id: 'I-1002', name: 'Print & outdoor' },
    startDate: '2026-01-01',
    endDate: '2026-12-31',
    current: false,
    moveProblem: null,
    copyProblem: null,
    ...overrides,
  };
}

export const OWN_INITIATIVE = destination({
  initiative: { type: 'initiative', id: 'I-1001', name: 'Digital & social' },
  endDate: '2026-06-30',
  current: true,
  moveProblem: "It's already in this initiative.",
});

/** "Retail events" (1 Jun – 31 Dec 2026): Launch film's dates don't fit. */
export const RETAIL_EVENTS = destination({
  campaign: { type: 'campaign', id: 'C-1002', name: 'Rimula fleet' },
  initiative: { type: 'initiative', id: 'I-1003', name: 'Retail events' },
  startDate: '2026-06-01',
  moveProblem:
    "'Launch film' runs 01 Mar 2026 – 31 May 2026, but 'Retail events' runs 01 Jun 2026 – 31 Dec 2026. Change the activity's dates first, or pick another initiative.",
  copyProblem:
    "'Launch film' runs 01 Mar 2026 – 31 May 2026, but 'Retail events' runs 01 Jun 2026 – 31 Dec 2026. Change the activity's dates first, or pick another initiative.",
});

export function destinations(
  items: ActivityDestination[] = [OWN_INITIATIVE, destination(), RETAIL_EVENTS],
) {
  return { items, total: items.length };
}

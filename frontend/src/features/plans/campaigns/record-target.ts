import type { Campaign, Initiative } from './api';
import type { RecordLevel } from './texts';

/** A campaign or initiative a row action is about. */
export interface RecordTarget {
  level: RecordLevel;
  id: string;
  name: string;
  planId: string;
  /** The plan's year: a campaign moves only between plans of that year (FR-023). */
  year: number;
  /** The version the row or panel showed; a move sends it (FR-025). */
  version: number;
  /** What the signed-in person may do with it (feature 014); missing means everything. */
  access?: Campaign['access'];
  /** What goes with it (the delete, move and copy dialogs). */
  counts: {
    initiatives?: number;
    activities: number;
    approvedActivities: number;
    lineItems: number;
  };
}

/** The record a row stands for, for its actions. */
export function targetOf(record: Campaign | Initiative): RecordTarget {
  const common = {
    id: record.id,
    name: record.name,
    planId: record.planId,
    year: record.plan.year,
    version: record.version,
    access: record.access,
  };
  return 'initiatives' in record
    ? { ...common, level: 'campaign', counts: record.counts }
    : { ...common, level: 'initiative', counts: record.counts };
}

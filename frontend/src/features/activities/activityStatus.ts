/**
 * An activity's status as people see it (006 data-model, "Statuses"; design system §4): the word
 * and the colour family of each display status. Built with the shared foundation because feature
 * 008 (approvals) shows it too; feature 006 adds the banner texts here.
 */
import type { components } from '@/api/schema';
import type { StatusTone } from '@/components';

export type ActivityDisplayStatus = components['schemas']['ActivityDisplayStatus'];

const STATUS: Record<ActivityDisplayStatus, { word: string; tone: StatusTone }> = {
  created: { word: 'Created', tone: 'neutral' },
  submitted: { word: 'Submitted', tone: 'review' },
  declined: { word: 'Declined', tone: 'problem' },
  approved: { word: 'Approved', tone: 'positive' },
  inProgress: { word: 'In progress', tone: 'active' },
  complete: { word: 'Complete', tone: 'done' },
};

/** "Created", "Submitted", "Declined", "Approved", "In progress", "Complete". */
export function activityStatusWord(status: ActivityDisplayStatus): string {
  return STATUS[status].word;
}

/** The status family of design system §4: neutral, review, problem, positive, active, done. */
export function activityStatusTone(status: ActivityDisplayStatus): StatusTone {
  return STATUS[status].tone;
}

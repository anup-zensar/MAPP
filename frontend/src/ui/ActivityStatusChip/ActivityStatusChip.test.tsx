import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { activityStatusTone, activityStatusWord } from './activity-status';
import { ActivityStatusChip } from './ActivityStatusChip';

describe('ActivityStatusChip and activityStatus (feature 006, FR-021)', () => {
  it.each([
    ['created', 'Created', 'neutral'],
    ['submitted', 'Submitted', 'review'],
    ['declined', 'Declined', 'problem'],
    ['approved', 'Approved', 'positive'],
    ['inProgress', 'In progress', 'active'],
    ['complete', 'Complete', 'done'],
  ] as const)('shows %s as "%s" in the %s family', (status, word, tone) => {
    render(<ActivityStatusChip status={status} />);

    expect(screen.getByText(word)).toBeInTheDocument();
    expect(activityStatusWord(status)).toBe(word);
    expect(activityStatusTone(status)).toBe(tone);
  });
});

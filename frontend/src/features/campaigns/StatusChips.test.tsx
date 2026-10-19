import { screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { renderWithProviders } from '@/test/render';
import { ApprovalProgressChip, OutsideYearNote, TimingChip } from './StatusChips';

describe('StatusChips (feature 005, FR-013, FR-015)', () => {
  it.each([
    [{ state: 'noActivities', approved: 0, total: 0 }, 'No activities', 'neutral'],
    [
      { state: 'inProgress', approved: 0, total: 5 },
      'In progress · 0 of 5 activities approved',
      'review',
    ],
    [
      { state: 'inProgress', approved: 3, total: 5 },
      'In progress · 3 of 5 activities approved',
      'review',
    ],
    [
      { state: 'inProgress', approved: 0, total: 1 },
      'In progress · 0 of 1 activity approved',
      'review',
    ],
    [{ state: 'allApproved', approved: 5, total: 5 }, 'All approved', 'positive'],
  ] as const)('shows approval progress %o as "%s"', (progress, text, tone) => {
    renderWithProviders(<ApprovalProgressChip progress={progress} />);

    expect(screen.getByText(text).className).toContain(tone);
  });

  it.each([
    ['notStarted', 'Not started', 'neutral'],
    ['running', 'Running', 'active'],
    ['finished', 'Finished', 'done'],
  ] as const)('shows timing %s as "%s"', (timing, text, tone) => {
    renderWithProviders(<TimingChip timing={timing} />);

    expect(screen.getByText(text).className).toContain(tone);
  });

  it('shows "—" with the reason while there are no dates', async () => {
    const { user } = renderWithProviders(<TimingChip timing={null} />);

    expect(screen.getByText('—')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'About Timing' }));
    expect(await screen.findByText('Timing appears once dates are set')).toBeInTheDocument();
  });

  it('warns that a campaign runs outside its plan’s year, as text or as a row icon (FR-008)', () => {
    renderWithProviders(
      <>
        <OutsideYearNote year={2026} />
        <OutsideYearNote year={2026} compact />
      </>,
    );

    expect(
      screen.getByText("This campaign runs outside 2026, the plan's year"),
    ).toBeInTheDocument();
    expect(
      screen.getByRole('img', { name: "This campaign runs outside 2026, the plan's year" }),
    ).toBeInTheDocument();
  });
});

import { screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { renderWithProviders } from '@/test/render';
import { ApprovalProgressChip, OutsideYearNote, TimingNote } from './StatusChips';

describe('StatusChips (feature 005, FR-013, FR-015)', () => {
  it.each([
    [{ state: 'noActivities', approved: 0, total: 0 }, 'No activities', 'neutral'],
    [{ state: 'allApproved', approved: 5, total: 5 }, 'All approved', 'positive'],
  ] as const)('shows approval progress %o as "%s"', (progress, text, tone) => {
    renderWithProviders(<ApprovalProgressChip progress={progress} />);

    expect(screen.getByText(text).className).toContain(tone);
  });

  it.each([
    [
      { state: 'inProgress', approved: 0, total: 5 },
      '0 of 5 approved',
      '0 of 5 activities approved',
    ],
    [
      { state: 'inProgress', approved: 3, total: 5 },
      '3 of 5 approved',
      '3 of 5 activities approved',
    ],
    [{ state: 'inProgress', approved: 0, total: 1 }, '0 of 1 approved', '0 of 1 activity approved'],
  ] as const)(
    'writes a partly approved roll-up %o as "%s", never "In progress" (functional review A-02)',
    (progress, shown, heard) => {
      renderWithProviders(<ApprovalProgressChip progress={progress} />);

      const chip = screen.getByText(shown).parentElement as HTMLElement;
      expect(chip.className).toContain('review');
      expect(chip).toHaveTextContent(heard);
      expect(screen.queryByText(/In progress/)).not.toBeInTheDocument();
    },
  );

  it.each([
    ['notStarted', 'Not started'],
    ['running', 'Running'],
    ['finished', 'Finished'],
  ] as const)('writes timing %s as "%s" under the dates, not as a chip', (timing, text) => {
    renderWithProviders(<TimingNote timing={timing} />);

    expect(screen.getByText(text)).toBeInTheDocument();
  });

  it('writes no timing while there are no dates', () => {
    renderWithProviders(<TimingNote timing={null} />);

    expect(screen.queryByText(/Not started|Running|Finished/)).not.toBeInTheDocument();
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

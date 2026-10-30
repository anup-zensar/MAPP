import { screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import { spaView } from '@/testing/fixtures/spa';
import { renderWithProviders } from '@/testing/render';
import { SpaPeriods } from './SpaPeriods';

const usd = (amount: string) => ({ amount, currency: 'USD' });

function renderPeriods(onTrace = vi.fn()) {
  const view = spaView();
  const periods = {
    ...view.periods,
    notScheduled: usd('20000.00'),
    notScheduledTrace: [
      { activityId: 'A-1002', name: 'Social always-on', amount: usd('20000.00') },
    ],
    linesAbovePlannedCost: usd('5000.00'),
    linesAbovePlannedCostTrace: [
      { activityId: 'A-1001', name: 'Launch film', amount: usd('5000.00') },
    ],
  };
  renderWithProviders(
    <SpaPeriods
      view={{ ...view, periods }}
      period="quarter"
      onPeriod={vi.fn()}
      onTrace={onTrace}
    />,
  );
  return onTrace;
}

describe('SpaPeriods (013 FR-017)', () => {
  it('shows the excess of line items over planned cost on a row of its own (038 FR-023, M-2)', async () => {
    const onTrace = renderPeriods();

    const row = screen.getByRole('row', { name: /Line items above planned cost/ });
    const excess = within(row).getByRole('button', {
      name: '$5,000.00, show what makes up Line items above planned cost',
    });
    await userEvent.click(excess);
    expect(onTrace).toHaveBeenCalledWith({
      figure: 'Line items above planned cost',
      total: usd('5000.00'),
      items: [{ activityId: 'A-1001', name: 'Launch film', amount: usd('5000.00') }],
    });
  });

  it('shows Not yet scheduled as a plain figure, never as a problem (038 FR-023, M-2)', () => {
    renderPeriods();

    const row = screen.getByRole('row', { name: /Not yet scheduled/ });
    const figure = within(row).getByRole('button', { name: /^\$20,000\.00/ });
    expect(figure.className).not.toMatch(/problem/);
  });
});

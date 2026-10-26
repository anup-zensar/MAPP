import { screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { mockApi } from '@/testing/mock-api';
import { renderWithProviders } from '@/testing/render';
import { plan } from '@/testing/fixtures/plans';
import { reply } from '@/testing/replies';
import type { Plan } from '../api';
import { DeletePlanDialog } from './DeletePlanDialog';

function renderDialog(target: Plan) {
  mockApi([reply('GET /api/v1/plans/{planId}', { body: target }, { planId: 'P-1001' })]);
  const onDelete = vi.fn();
  const result = renderWithProviders(
    <DeletePlanDialog
      open
      planId="P-1001"
      planName={target.name}
      busy={false}
      onDelete={onDelete}
      onClose={() => {}}
    />,
  );
  return { ...result, onDelete };
}

describe('DeletePlanDialog (US6#3, FR-029)', () => {
  it('says what goes with a published plan and that approved activities return as Created', async () => {
    const { user, onDelete } = renderDialog(
      plan({
        status: 'published',
        contents: {
          campaigns: 2,
          initiatives: 4,
          activities: 7,
          approvedActivities: 4,
          lineItems: 15,
        },
      }),
    );

    const dialog = screen.getByRole('alertdialog', {
      name: "Delete plan 'FY2026 Lubricants Europe'?",
    });
    expect(
      await screen.findByText(
        'It holds 2 campaigns, 4 initiatives, 7 activities and 15 line items.',
      ),
    ).toBeInTheDocument();
    expect(dialog).toHaveTextContent('This plan is published.');
    expect(dialog).toHaveTextContent(
      '4 of its activities are approved; if the plan is restored later from the recycle bin, they return as Created.',
    );
    expect(dialog).toHaveTextContent('It moves to the recycle bin; you can restore it.');
    expect(screen.getByRole('button', { name: 'Cancel' })).toHaveFocus();

    await user.click(screen.getByRole('button', { name: 'Delete plan' }));
    expect(onDelete).toHaveBeenCalledTimes(1);
  });

  it('says when the plan holds nothing yet', async () => {
    renderDialog(plan());

    expect(await screen.findByText('It holds nothing yet.')).toBeInTheDocument();
    expect(screen.queryByText('This plan is published.')).not.toBeInTheDocument();
  });

  it('names one approved activity in the singular', async () => {
    renderDialog(
      plan({
        contents: {
          campaigns: 1,
          initiatives: 1,
          activities: 1,
          approvedActivities: 1,
          lineItems: 0,
        },
      }),
    );

    expect(
      await screen.findByText(
        /^1 of its activities is approved; if the plan is restored later from the recycle bin/,
      ),
    ).toBeInTheDocument();
  });
});

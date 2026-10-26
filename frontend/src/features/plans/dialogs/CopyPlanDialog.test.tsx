import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { mockApi } from '@/testing/mock-api';
import type { LooseAnswer } from '@/testing/replies';
import { renderWithProviders } from '@/testing/render';
import { looseReply } from '@/testing/replies';
import type { PlanCopyOptions } from '../api';
import { CopyPlanDialog } from './CopyPlanDialog';
import type { CopyRefusal } from './CopyPlanDialog';

function copyOptions(overrides: Partial<PlanCopyOptions> = {}): PlanCopyOptions {
  return {
    name: 'FY2026 Lubricants Europe (copy)',
    sameYear: 2026,
    nextYear: { year: 2027, available: true, reason: '' },
    contents: { campaigns: 2, initiatives: 4, activities: 7, approvedActivities: 4, lineItems: 15 },
    ...overrides,
  };
}

function renderDialog(
  answer: LooseAnswer = { body: copyOptions() },
  onCopy: (request: unknown) => Promise<CopyRefusal | null> = async () => null,
) {
  const api = mockApi([
    looseReply('GET /api/v1/plans/{planId}/copy-options', answer, { planId: 'P-1001' }),
  ]);
  const copy = vi.fn(onCopy);
  const onClose = vi.fn();
  const result = renderWithProviders(
    <CopyPlanDialog
      open
      planId="P-1001"
      planName="FY2026 Lubricants Europe"
      onCopy={copy}
      onClose={onClose}
    />,
  );
  const dialog = screen.getByRole('dialog', { name: "Copy plan 'FY2026 Lubricants Europe'" });
  return { ...result, api, dialog, onCopy: copy, onClose };
}

describe('CopyPlanDialog (US6#1–2, FR-026 – FR-028)', () => {
  it('offers the suggested name, the years and the counts, and copies everything by default', async () => {
    const { user, dialog, onCopy } = renderDialog();

    const name = await within(dialog).findByRole('textbox', { name: /Name of the copy/ });
    expect(name).toHaveValue('FY2026 Lubricants Europe (copy)');
    expect(name).toHaveFocus();
    const year = within(dialog).getByRole('group', { name: 'Year' });
    expect(within(year).getByRole('radio', { name: 'Same year (2026)' })).toBeChecked();
    expect(within(year).getByRole('radio', { name: 'Next year (2027)' })).toBeEnabled();
    const include = within(dialog).getByRole('checkbox', {
      name: 'Include campaigns, initiatives, activities and line items',
    });
    expect(include).toBeChecked();
    expect(include).toHaveAccessibleDescription(
      '2 campaigns, 4 initiatives, 7 activities and 15 line items',
    );
    expect(dialog).not.toHaveTextContent('Every date moves one year later.');

    await user.click(within(dialog).getByRole('button', { name: 'Copy plan' }));

    expect(onCopy).toHaveBeenCalledWith({
      name: 'FY2026 Lubricants Europe (copy)',
      year: 2026,
      includeContents: true,
    });
  });

  it('copies into the next year under another name, with the plan only (US6#1, FR-028)', async () => {
    const { user, dialog, onCopy } = renderDialog();

    const name = await within(dialog).findByRole('textbox', { name: /Name of the copy/ });
    await user.clear(name);
    await user.type(name, 'FY2027 Lubricants Europe');
    await user.click(within(dialog).getByRole('radio', { name: 'Next year (2027)' }));
    expect(within(dialog).getByText('Every date moves one year later.')).toBeInTheDocument();
    await user.click(
      within(dialog).getByRole('checkbox', {
        name: 'Include campaigns, initiatives, activities and line items',
      }),
    );
    await user.keyboard('{Enter}');

    expect(onCopy).toHaveBeenCalledWith({
      name: 'FY2027 Lubricants Europe',
      year: 2027,
      includeContents: false,
    });
  });

  it('shows the next year as unavailable, with the reason (FR-028)', async () => {
    const { dialog } = renderDialog({
      body: copyOptions({
        name: 'FY2031 Lubricants (copy 2)',
        sameYear: 2031,
        nextYear: {
          year: 2032,
          available: false,
          reason: 'Plans can only be created up to 2031.',
        },
      }),
    });

    const next = await within(dialog).findByRole('radio', { name: 'Next year (2032)' });
    expect(next).toBeDisabled();
    expect(next).toHaveAccessibleDescription('Plans can only be created up to 2031.');
    expect(within(dialog).getByRole('textbox', { name: /Name of the copy/ })).toHaveValue(
      'FY2031 Lubricants (copy 2)',
    );
  });

  it('says when the plan has nothing inside yet', async () => {
    const { dialog } = renderDialog({
      body: copyOptions({
        contents: {
          campaigns: 0,
          initiatives: 0,
          activities: 0,
          approvedActivities: 0,
          lineItems: 0,
        },
      }),
    });

    expect(
      await within(dialog).findByRole('checkbox', {
        name: 'Include campaigns, initiatives, activities and line items',
      }),
    ).toHaveAccessibleDescription('This plan has nothing inside yet.');
  });

  it('shows the API’s problems under their fields and keeps what was typed', async () => {
    const { user, dialog } = renderDialog({ body: copyOptions() }, async () => ({
      problems: [
        {
          field: 'name',
          message:
            "There is already a plan called 'FY2026 Lubricants Europe (copy)'. Choose another name.",
        },
        { field: 'year', message: 'Copy into 2026 or 2027.' },
      ],
    }));

    await within(dialog).findByRole('textbox', { name: /Name of the copy/ });
    await user.click(within(dialog).getByRole('button', { name: 'Copy plan' }));

    const name = within(dialog).getByRole('textbox', { name: /Name of the copy/ });
    expect(name).toHaveAccessibleDescription(
      expect.stringContaining(
        "There is already a plan called 'FY2026 Lubricants Europe (copy)'. Choose another name.",
      ),
    );
    expect(name).toHaveValue('FY2026 Lubricants Europe (copy)');
    expect(within(dialog).getByRole('group', { name: 'Year' })).toHaveAccessibleDescription(
      'Copy into 2026 or 2027.',
    );
  });

  it('shows a refusal that is not about one field above the form', async () => {
    const { user, dialog } = renderDialog({ body: copyOptions() }, async () => ({
      problems: [],
      message: 'Something went wrong. Please try again.',
    }));

    await within(dialog).findByRole('textbox', { name: /Name of the copy/ });
    await user.click(within(dialog).getByRole('button', { name: 'Copy plan' }));

    expect(await within(dialog).findByRole('alert')).toHaveTextContent(
      'Something went wrong. Please try again.',
    );
  });

  it('shows grey lines while the options load, then an error with Try again', async () => {
    let calls = 0;
    const { user, dialog, onCopy } = renderDialog(() => {
      calls += 1;
      return calls === 1 ? { status: 500 } : { body: copyOptions() };
    });

    expect(within(dialog).getByRole('status')).toHaveTextContent('Loading…');
    const tryAgain = await within(dialog).findByRole('button', { name: 'Try again' });
    // Nothing to copy with yet: Copy plan does nothing while the error says why.
    await user.click(within(dialog).getByRole('button', { name: 'Copy plan' }));
    expect(onCopy).not.toHaveBeenCalled();
    await user.click(tryAgain);

    expect(await within(dialog).findByRole('textbox', { name: /Name of the copy/ })).toHaveValue(
      'FY2026 Lubricants Europe (copy)',
    );
  });

  it('says when the plan no longer exists, and Cancel closes', async () => {
    const { user, dialog, onClose } = renderDialog({
      status: 404,
      body: {
        error: {
          code: 'not_found',
          message: 'This plan no longer exists — someone may have deleted it.',
          fields: [],
        },
      },
    });

    expect(await within(dialog).findByRole('alert')).toHaveTextContent(
      'This plan no longer exists — someone may have deleted it.',
    );
    expect(within(dialog).queryByRole('button', { name: 'Try again' })).not.toBeInTheDocument();
    await user.click(within(dialog).getByRole('button', { name: 'Cancel' }));
    await waitFor(() => expect(onClose).toHaveBeenCalled());
    expect(screen.queryByRole('textbox')).not.toBeInTheDocument();
  });
});

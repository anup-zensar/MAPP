import { screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { ALEX, RULES } from '@/testing/fixtures/access';
import { renderAccess } from '@/testing/fixtures/access-page';
import { reply } from '@/testing/replies';

describe('AccessPage: Rules tab (feature 014, FR-019)', () => {
  it('shows the rules and saves a change with their version', async () => {
    const { api, user } = renderAccess(
      [
        reply('PATCH /api/v1/access-rules', {
          body: {
            ...RULES,
            noSelfApproval: false,
            version: 1,
            updatedAt: '2026-10-20T10:00:00Z',
            updatedBy: { id: ALEX.id, name: ALEX.name },
          },
        }),
      ],
      '/settings/access?tab=rules',
    );

    expect(await screen.findByText('On — someone else must approve.')).toBeInTheDocument();
    const card = screen.getByRole('region', { name: 'Access rules' });
    expect(card).toHaveTextContent('Nobody approves an activity they own or submitted');
    expect(card).toHaveTextContent('On — someone else must approve.');
    expect(card).toHaveTextContent('Viewer · My teams');
    await user.click(within(card).getByRole('button', { name: 'Edit rules' }));
    const panel = await screen.findByRole('dialog', { name: 'Edit access rules' });
    const role = within(panel).getByRole('combobox', { name: /Role for new people/ });
    expect(within(role).queryByRole('option', { name: 'Administrator' })).not.toBeInTheDocument();
    await user.click(
      within(panel).getByRole('checkbox', {
        name: /Nobody approves an activity they own or submitted/,
      }),
    );
    await user.click(within(panel).getByRole('button', { name: 'Save' }));

    expect(await screen.findByText('Access rules saved')).toBeInTheDocument();
    expect(api.bodiesOf('PATCH /api/v1/access-rules')).toEqual([
      { noSelfApproval: false, version: 0 },
    ]);
  });
});

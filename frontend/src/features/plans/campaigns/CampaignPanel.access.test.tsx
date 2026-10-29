import { screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { DETAILS, findDetails, renderPanel } from '@/testing/fixtures/campaign-panel';
import { campaign } from '@/testing/fixtures/campaigns';
import { VIEWER_CONTAINER_ACCESS } from '@/testing/fixtures/viewer';
import { reply } from '@/testing/replies';

const REASON =
  "Changing campaigns and initiatives isn't part of your role (Viewer). Ask an administrator if you need it.";

const READ_ONLY = reply('GET /api/v1/campaigns/{campaignId}', {
  body: campaign({ access: VIEWER_CONTAINER_ACCESS }),
});

describe('014 FR-011, FR-014: campaign panels follow the reader’s access', () => {
  it('says why at the edit address of a campaign the person may not change', async () => {
    renderPanel(`${DETAILS}/edit`, [READ_ONLY]);

    expect(await screen.findByText(REASON)).toBeVisible();
    const panel = screen.getByRole('dialog', { name: 'Edit campaign' });
    expect(within(panel).queryByLabelText('Campaign name')).not.toBeInTheDocument();
    expect(within(panel).getByRole('button', { name: 'Back to the plan' })).toBeVisible();
  });

  it('lists Edit in More, disabled with its reason, instead of the Edit button', async () => {
    const { user } = renderPanel(DETAILS, [READ_ONLY]);

    const panel = await findDetails();
    expect(within(panel).queryByRole('link', { name: 'Edit' })).not.toBeInTheDocument();
    await user.click(within(panel).getByRole('button', { name: 'Actions for Helix launch' }));
    expect(screen.getByRole('menuitem', { name: 'Edit' })).toHaveAccessibleDescription(REASON);
  });
});

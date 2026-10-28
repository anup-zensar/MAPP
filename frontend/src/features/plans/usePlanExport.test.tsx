import { screen, within } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { ActionsMenu } from '@/ui/ActionsMenu/ActionsMenu';
import { testUser } from '@/testing/fixtures/users';
import { mockApi } from '@/testing/mock-api';
import { findMessages, renderWithProviders } from '@/testing/render';
import { apiError, reply } from '@/testing/replies';
import type { Answer } from '@/testing/replies';
import { usePlanExport } from './usePlanExport';

const NAME = 'SP&A One – FY2026 Lubricants Europe – 28 Oct 2026.pptx';
const VIEWER = testUser({
  access: {
    roles: [],
    administrators: [],
    noSelfApproval: true,
    permissions: [{ key: 'plans.view', all: true, teams: [] }],
  },
});

let saved: string[];

beforeEach(() => {
  saved = [];
  URL.createObjectURL = () => 'blob:export';
  URL.revokeObjectURL = () => {};
  vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(function (
    this: HTMLAnchorElement,
  ) {
    saved.push(this.download);
  });
});

afterEach(() => vi.restoreAllMocks());

function PlanMenu() {
  return <ActionsMenu label="More actions" trigger="more" items={usePlanExport('P-1001')} />;
}

function renderMenu(answer: Answer<'GET /api/v1/plans/{planId}/export'>, user = testUser()) {
  const api = mockApi([
    reply('GET /api/v1/plans/{planId}/export', answer),
    reply('GET /api/v1/me', { body: user }),
  ]);
  return { ...renderWithProviders(<PlanMenu />, { user }), api };
}

describe('usePlanExport (036 US2, FR-023)', () => {
  it('saves the plan as a PowerPoint file under the name the API gives', async () => {
    const file = new Response('PK', {
      headers: {
        'Content-Disposition': `attachment; filename*=UTF-8''${encodeURIComponent(NAME)}`,
      },
    });
    const { user, api } = renderMenu(() => file);

    await user.click(screen.getByRole('button', { name: 'More' }));
    await user.click(await screen.findByRole('menuitem', { name: 'Export to PowerPoint' }));

    expect(within(await findMessages()).getByText(`Exported '${NAME}'`)).toBeInTheDocument();
    expect(saved).toEqual([NAME]);
    expect(api.requests.at(-1)?.search).toBe('?format=pptx');
  });

  it("shows the API's refusal (US2 scenario 6)", async () => {
    const { user } = renderMenu({
      status: 403,
      body: apiError('forbidden', "Your role can't export this plan."),
    });

    await user.click(screen.getByRole('button', { name: 'More' }));
    await user.click(await screen.findByRole('menuitem', { name: 'Export to PDF' }));

    expect(
      within(await findMessages()).getByText("Your role can't export this plan."),
    ).toBeInTheDocument();
  });

  it('offers nothing to a person without the export permission (US2 scenario 6)', async () => {
    const { user } = renderMenu({ status: 403 }, VIEWER);

    await user.click(screen.getByRole('button', { name: 'More' }));

    expect(screen.queryByRole('menuitem', { name: /Export/ })).not.toBeInTheDocument();
  });
});

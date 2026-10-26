import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { ActionsMenu } from '@/ui/ActionsMenu/ActionsMenu';
import { mockApi } from '@/testing/mockApi';
import type { MockHandler } from '@/testing/mockApi';
import { findMessages, renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/users';
import { plan } from './testData';
import { usePlanActions } from './usePlanActions';
import type { ActionPlan } from './usePlanActions';

const CHINA: ActionPlan = { id: 'P-1002', name: 'FY2026 Mobility China', status: 'draft' };
const PUBLISHED: ActionPlan = {
  id: 'P-1003',
  name: 'FY2025 Lubricants Europe',
  status: 'published',
};

function Harness({ target }: { target: ActionPlan }) {
  const actions = usePlanActions();
  return (
    <>
      <ActionsMenu label={target.name} items={actions.menuItems(target)} />
      {actions.dialogs}
    </>
  );
}

function renderActions(target: ActionPlan, answers: Record<string, MockHandler>) {
  const api = mockApi(answers);
  const result = renderWithProviders(<Harness target={target} />, {
    user: testUser(),
    route: '/plans',
    path: '/plans',
  });
  return { ...result, api };
}

async function choose(
  user: ReturnType<typeof renderActions>['user'],
  name: string,
  action: string,
) {
  await user.click(screen.getByRole('button', { name: `Actions for ${name}` }));
  await user.click(await screen.findByRole('menuitem', { name: action }));
}

const NOT_READY = {
  status: 409,
  body: {
    error: {
      code: 'not_ready',
      message: 'Publish needs 3 fields',
      fields: [],
      missing: [
        {
          record: { type: 'plan', id: 'P-1002', name: 'FY2026 Mobility China' },
          fields: [{ field: 'attributes.SA-BU', label: 'Business unit' }],
        },
        {
          record: { type: 'campaign', id: 'C-1001', name: 'Retail growth' },
          fields: [
            { field: 'attributes.SA-CLUSTER', label: 'Cluster' },
            { field: 'attributes.SA-REGION', label: 'Region' },
          ],
        },
      ],
    },
  },
};

function reasonProblem(message: string) {
  return {
    status: 400,
    body: {
      error: {
        code: 'validation_failed',
        message: 'Some fields need your attention.',
        fields: [{ field: 'reason', message }],
      },
    },
  };
}

describe('usePlanActions: publish and unpublish (US5, FR-024, FR-025)', () => {
  it('offers Publish on a Draft plan and Unpublish on a Published one', async () => {
    const { user } = renderActions(PUBLISHED, {});

    await user.click(screen.getByRole('button', { name: `Actions for ${PUBLISHED.name}` }));
    expect((await screen.findAllByRole('menuitem')).map((item) => item.textContent)).toEqual([
      'View',
      'Edit',
      'Copy',
      'Unpublish',
      'Delete',
    ]);
  });

  it('publishes a complete Draft plan (US5#1)', async () => {
    const { user, api } = renderActions(CHINA, {
      'POST /api/v1/plans/P-1002/publish': {
        body: plan({ id: 'P-1002', name: 'FY2026 Mobility China', status: 'published' }),
      },
    });

    await choose(user, CHINA.name, 'Publish');

    expect(await screen.findByText("Plan 'FY2026 Mobility China' published")).toBeInTheDocument();
    expect(api.requests.filter((r) => r.path === '/api/v1/plans/P-1002/publish')).toHaveLength(1);
  });

  it('lists every missing field, grouped by record, with links (US5#2, FR-024)', async () => {
    const { user, router } = renderActions(CHINA, {
      'POST /api/v1/plans/P-1002/publish': NOT_READY,
    });

    await choose(user, CHINA.name, 'Publish');

    const dialog = await screen.findByRole('dialog', { name: 'Publish needs 3 fields' });
    expect(dialog).toHaveTextContent('Fill these in, then publish again:');
    expect(
      within(dialog)
        .getAllByRole('heading', { level: 3 })
        .map((h) => h.textContent),
    ).toEqual(['Plan', "Campaign 'Retail growth'"]);
    expect(within(dialog).getByRole('link', { name: 'Cluster' })).toHaveAttribute(
      'href',
      '/plans/P-1002/campaigns/C-1001/edit?focus=attributes.SA-CLUSTER',
    );
    await user.click(within(dialog).getByRole('link', { name: 'Business unit' }));

    await waitFor(() => expect(router.state.location.pathname).toBe('/plans/P-1002/edit'));
    expect(router.state.location.search).toBe('?focus=attributes.SA-BU');
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
  });

  it('closes the blocked dialog with Close', async () => {
    const { user } = renderActions(CHINA, { 'POST /api/v1/plans/P-1002/publish': NOT_READY });

    await choose(user, CHINA.name, 'Publish');
    await user.click(await screen.findByRole('button', { name: 'Close' }));

    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
  });

  it('asks why before unpublishing and shows the API’s problems with the reason (US5#3, FR-025)', async () => {
    let calls = 0;
    const { user, api } = renderActions(PUBLISHED, {
      'POST /api/v1/plans/P-1003/unpublish': () => {
        calls += 1;
        if (calls === 1) return reasonProblem('Enter a reason.');
        if (calls === 2) return reasonProblem('Use 500 characters or fewer.');
        return { body: plan({ id: 'P-1003', name: PUBLISHED.name, status: 'draft' }) };
      },
    });

    await choose(user, PUBLISHED.name, 'Unpublish');
    const dialog = await screen.findByRole('dialog', {
      name: "Unpublish plan 'FY2025 Lubricants Europe'?",
    });
    const reason = within(dialog).getByRole('textbox', {
      name: /Why are you unpublishing this plan\?/,
    });
    expect(reason).toHaveFocus();
    expect(dialog).toHaveTextContent(
      'The plan returns to Draft. Your reason is recorded in the activity log.',
    );

    await user.click(within(dialog).getByRole('button', { name: 'Unpublish plan' }));
    expect(await within(dialog).findByText('Enter a reason.')).toBeInTheDocument();

    await user.click(reason);
    await user.paste('x'.repeat(501));
    await user.click(within(dialog).getByRole('button', { name: 'Unpublish plan' }));
    expect(await within(dialog).findByText('Use 500 characters or fewer.')).toBeInTheDocument();

    await user.clear(reason);
    await user.type(reason, 'Budget is being reworked.');
    await user.click(within(dialog).getByRole('button', { name: 'Unpublish plan' }));

    expect(
      await screen.findByText("Plan 'FY2025 Lubricants Europe' unpublished — it's a Draft again"),
    ).toBeInTheDocument();
    expect(api.bodiesOf('POST /api/v1/plans/P-1003/unpublish')).toEqual([
      { reason: '' },
      { reason: 'x'.repeat(501) },
      { reason: 'Budget is being reworked.' },
    ]);
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
  });
});

const COPY_OPTIONS = {
  name: 'FY2025 Lubricants Europe (copy)',
  sameYear: 2025,
  nextYear: { year: 2026, available: true, reason: '' },
  contents: { campaigns: 2, initiatives: 4, activities: 7, approvedActivities: 4, lineItems: 15 },
};

describe('usePlanActions: copy (US6#1–2, FR-026 – FR-028)', () => {
  it('copies into the next year and opens the copy with its messages and notes (US6#2)', async () => {
    const { user, api, router } = renderActions(PUBLISHED, {
      'GET /api/v1/plans/P-1003/copy-options': { body: COPY_OPTIONS },
      'POST /api/v1/plans/P-1003/copy': {
        status: 201,
        body: {
          plan: plan({ id: 'P-1004', name: 'FY2026 Lubricants Europe', year: 2026 }),
          notes: ['Default finance budget cleared — it belonged to 2025.'],
        },
      },
    });

    await choose(user, PUBLISHED.name, 'Copy');
    const dialog = await screen.findByRole('dialog', {
      name: "Copy plan 'FY2025 Lubricants Europe'",
    });
    const name = await within(dialog).findByRole('textbox', { name: /Name of the copy/ });
    await user.clear(name);
    await user.type(name, 'FY2026 Lubricants Europe');
    await user.click(within(dialog).getByRole('radio', { name: 'Next year (2026)' }));
    await user.click(within(dialog).getByRole('button', { name: 'Copy plan' }));

    await waitFor(() => expect(router.state.location.pathname).toBe('/plans/P-1004'));
    expect(
      await screen.findByText(
        "Plan 'FY2026 Lubricants Europe' created as a copy of 'FY2025 Lubricants Europe'",
      ),
    ).toBeInTheDocument();
    expect(
      screen.getByText('Default finance budget cleared — it belonged to 2025.'),
    ).toBeInTheDocument();
    expect(api.bodiesOf('POST /api/v1/plans/P-1003/copy')).toEqual([
      { name: 'FY2026 Lubricants Europe', year: 2026, includeContents: true },
    ]);
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
  });

  it('closes the dialog and says so when the plan is already gone', async () => {
    const gone = {
      status: 404,
      body: {
        error: {
          code: 'not_found',
          message: 'This plan no longer exists — someone may have deleted it.',
          fields: [],
        },
      },
    };
    const { user, router } = renderActions(PUBLISHED, {
      'GET /api/v1/plans/P-1003/copy-options': { body: COPY_OPTIONS },
      'POST /api/v1/plans/P-1003/copy': gone,
    });

    await choose(user, PUBLISHED.name, 'Copy');
    await user.click(await screen.findByRole('button', { name: 'Copy plan' }));

    expect(
      await screen.findByText('This plan no longer exists — someone may have deleted it.'),
    ).toBeInTheDocument();
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
    expect(router.state.location.pathname).toBe('/plans');
  });

  it('starts from the suggestion again each time it opens', async () => {
    const { user } = renderActions(CHINA, {
      'GET /api/v1/plans/P-1002/copy-options': {
        body: { ...COPY_OPTIONS, name: 'FY2026 Mobility China (copy)', sameYear: 2026 },
      },
    });

    await choose(user, CHINA.name, 'Copy');
    const name = await screen.findByRole('textbox', { name: /Name of the copy/ });
    await user.type(name, ' typed');
    await user.click(screen.getByRole('button', { name: 'Cancel' }));
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());

    await choose(user, CHINA.name, 'Copy');
    expect(await screen.findByRole('textbox', { name: /Name of the copy/ })).toHaveValue(
      'FY2026 Mobility China (copy)',
    );
  });
});

const DELETED = {
  recycleBinEntry: {
    id: 'RB-1001',
    recordType: 'plan',
    recordTypeLabel: 'plan',
    recordId: 'P-1003',
    recordName: 'FY2025 Lubricants Europe',
    path: [],
    parent: null,
    deletedAt: '2026-03-12T09:00:00Z',
    deletedBy: { id: 'U-1002', name: 'Priya Raman' },
    contents: [],
  },
};

describe('usePlanActions: delete with Undo (US6, FR-029, FR-030)', () => {
  it('deletes after the question, offers Undo, and Undo brings the plan back (US6#4)', async () => {
    const { user, api, router } = renderActions(PUBLISHED, {
      'GET /api/v1/plans/P-1003': {
        body: plan({ id: 'P-1003', name: PUBLISHED.name, status: 'published' }),
      },
      'DELETE /api/v1/plans/P-1003': { body: DELETED },
      'POST /api/v1/recycle-bin/RB-1001/undo': {
        body: plan({ id: 'P-1003', name: PUBLISHED.name, status: 'published' }),
      },
    });

    await choose(user, PUBLISHED.name, 'Delete');
    const dialog = await screen.findByRole('alertdialog', {
      name: "Delete plan 'FY2025 Lubricants Europe'?",
    });
    expect(await within(dialog).findByText('This plan is published.')).toBeInTheDocument();
    await user.click(within(dialog).getByRole('button', { name: 'Delete plan' }));

    expect(
      await screen.findByText("Plan 'FY2025 Lubricants Europe' moved to the recycle bin"),
    ).toBeInTheDocument();
    expect(router.state.location.pathname).toBe('/plans');
    await waitFor(() => expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument());

    const messages = await findMessages();
    await user.click(within(messages).getByRole('button', { name: 'Undo' }));

    expect(await screen.findByText("Plan 'FY2025 Lubricants Europe' restored")).toBeInTheDocument();
    expect(api.requests.map((request) => `${request.method} ${request.path}`)).toContain(
      'POST /api/v1/recycle-bin/RB-1001/undo',
    );
  });

  it('says so when the plan is already gone', async () => {
    const { user } = renderActions(PUBLISHED, {
      'GET /api/v1/plans/P-1003': {
        body: plan({ id: 'P-1003', name: PUBLISHED.name, status: 'published' }),
      },
      'DELETE /api/v1/plans/P-1003': {
        status: 404,
        body: {
          error: {
            code: 'not_found',
            message: 'This plan no longer exists — someone may have deleted it.',
            fields: [],
          },
        },
      },
    });

    await choose(user, PUBLISHED.name, 'Delete');
    await user.click(await screen.findByRole('button', { name: 'Delete plan' }));

    expect(
      await screen.findByText('This plan no longer exists — someone may have deleted it.'),
    ).toBeInTheDocument();
  });
});

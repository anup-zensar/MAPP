import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import type { HttpHandler } from 'msw';
import { mockApi } from '@/testing/mock-api';
import { renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import {
  ADMINISTRATOR,
  CATALOGUE,
  FINANCE_REVIEWER,
  PLANNER,
  VIEWER,
} from '@/testing/fixtures/access';
import { reply } from '@/testing/replies';
import { RoleFormPage } from './RoleFormPage';
import { RolePage } from './RolePage';

const ANSWERS: HttpHandler[] = [
  reply('GET /api/v1/me', { body: testUser() }),
  reply('GET /api/v1/access/permissions', { body: CATALOGUE }),
  reply('GET /api/v1/roles/{roleId}', { body: ADMINISTRATOR }, { roleId: ADMINISTRATOR.id }),
  reply('GET /api/v1/roles/{roleId}', { body: PLANNER }, { roleId: PLANNER.id }),
  reply('GET /api/v1/roles/{roleId}', { body: VIEWER }, { roleId: VIEWER.id }),
];

function renderRole(route: string, answers: HttpHandler[] = []) {
  const api = mockApi([...ANSWERS, ...answers]);
  return {
    api,
    ...renderWithProviders(<RolePage />, {
      user: testUser(),
      route,
      path: '/settings/access/roles/:roleId',
    }),
  };
}

function renderForm(route: string, answers: HttpHandler[] = []) {
  const api = mockApi([...ANSWERS, ...answers]);
  const path = route.includes('/edit')
    ? '/settings/access/roles/:roleId/edit'
    : '/settings/access/roles/new';
  return { api, ...renderWithProviders(<RoleFormPage />, { user: testUser(), route, path }) };
}

describe('RolePage (feature 014, US4)', () => {
  it('shows the permissions by group, ticked when the role has them, with Edit', async () => {
    renderRole(`/settings/access/roles/${PLANNER.id}`);

    expect(await screen.findByRole('heading', { level: 1, name: 'Planner' })).toBeInTheDocument();
    expect(screen.getByText('Held by 2 people')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Edit' })).toHaveAttribute(
      'href',
      `/settings/access/roles/${PLANNER.id}/edit`,
    );
    const plans = await screen.findByRole('region', { name: 'Plans and planning' });
    expect(within(plans).getByText('Publish and unpublish plans')).toHaveTextContent(
      'Publish and unpublish plans — allowed',
    );
    expect(within(plans).getByText('Delete plans')).toHaveTextContent('Delete plans — not allowed');
    expect(screen.getByText('Daniel Okoro')).toBeInTheDocument();
  });

  it('locks the Administrator role: every permission, a note and no Edit (US4-3)', async () => {
    renderRole(`/settings/access/roles/${ADMINISTRATOR.id}`);

    expect(
      await screen.findByText(
        "The Administrator role always has every permission and can't be changed.",
      ),
    ).toBeInTheDocument();
    expect(screen.queryByRole('link', { name: 'Edit' })).not.toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Copy as new role' })).toHaveAttribute(
      'href',
      `/settings/access/roles/new?copyOf=${ADMINISTRATOR.id}`,
    );
    await screen.findByRole('region', { name: 'Administration' });
    expect(screen.queryAllByText(/— not allowed$/)).toHaveLength(0);
  });

  it('says when the role does not exist', async () => {
    renderRole('/settings/access/roles/RL-1999', [
      reply(
        'GET /api/v1/roles/{roleId}',
        {
          status: 404,
          body: { error: { code: 'not_found', message: "This role doesn't exist.", fields: [] } },
        },
        { roleId: 'RL-1999' },
      ),
    ]);

    expect(await screen.findByText("This role doesn't exist.")).toBeInTheDocument();
  });
});

describe('RoleFormPage (feature 014, US4)', () => {
  it('creates a role as a copy of Viewer, named and ready to give (US4-2)', async () => {
    const { api, user, router } = renderForm(`/settings/access/roles/new?copyOf=${VIEWER.id}`, [
      reply('POST /api/v1/roles', { status: 201, body: FINANCE_REVIEWER }),
    ]);

    expect(await screen.findByRole('heading', { level: 1, name: 'New role' })).toBeInTheDocument();
    const name = await screen.findByRole('textbox', { name: /^Name/ });
    expect(name).toHaveValue('Copy of Viewer');
    expect(screen.getByRole('checkbox', { name: /^See financial data/ })).toBeChecked();
    await user.clear(name);
    await user.type(name, 'Finance reviewer');
    await user.click(screen.getByRole('button', { name: 'Create role' }));

    expect(await screen.findByText("Role 'Finance reviewer' created")).toBeInTheDocument();
    expect(api.bodiesOf('POST /api/v1/roles')).toEqual([
      {
        name: 'Finance reviewer',
        description: VIEWER.description,
        defaultScope: 'myTeams',
        permissions: ['plans.view', 'finance.view'],
      },
    ]);
    await waitFor(() =>
      expect(router.state.location.pathname).toBe(`/settings/access/roles/${FINANCE_REVIEWER.id}`),
    );
  });

  it('shows every problem at once', async () => {
    const { user } = renderForm('/settings/access/roles/new', [
      reply('POST /api/v1/roles', {
        status: 400,
        body: {
          error: {
            code: 'validation_failed',
            message: 'Check the highlighted fields.',
            fields: [
              { field: 'name', message: "A role called 'Planner' already exists." },
              { field: 'permissions', message: 'Choose at least one permission.' },
            ],
          },
        },
      }),
    ]);

    await user.type(await screen.findByRole('textbox', { name: /^Name/ }), 'Planner');
    await user.click(screen.getByRole('button', { name: 'Create role' }));

    expect(await screen.findAllByText("A role called 'Planner' already exists.")).toHaveLength(2);
    expect(screen.getAllByText('Choose at least one permission.')).toHaveLength(2);
  });

  it('saves only what changed with the version (US4-1)', async () => {
    const { api, user } = renderForm(`/settings/access/roles/${PLANNER.id}/edit`, [
      reply(
        'PATCH /api/v1/roles/{roleId}',
        {
          body: {
            ...PLANNER,
            permissions: PLANNER.permissions.filter((key) => key !== 'plans.publish'),
            version: 2,
          },
        },
        { roleId: PLANNER.id },
      ),
    ]);

    expect(
      await screen.findByRole('heading', { level: 1, name: "Edit role 'Planner'" }),
    ).toBeInTheDocument();
    await user.click(await screen.findByRole('checkbox', { name: /^Publish and unpublish plans/ }));
    await user.click(screen.getByRole('button', { name: 'Save' }));

    expect(await screen.findByText("Role 'Planner' saved")).toBeInTheDocument();
    expect(api.bodiesOf(`PATCH /api/v1/roles/${PLANNER.id}`)).toEqual([
      {
        permissions: PLANNER.permissions.filter((key) => key !== 'plans.publish'),
        version: 1,
      },
    ]);
  });
});

describe('038 FR-006 (H-4): link without import on the role form', () => {
  const NEEDS_IMPORT =
    'Linking financial transactions needs Import GSAP extracts too — only people who import see transactions that are not linked.';

  it("places the server's permissions problem on the permission list", async () => {
    const { user } = renderForm(`/settings/access/roles/${PLANNER.id}/edit`, [
      reply(
        'PATCH /api/v1/roles/{roleId}',
        {
          status: 400,
          body: {
            error: {
              code: 'validation_failed',
              message: 'Check the highlighted fields.',
              fields: [{ field: 'permissions', message: NEEDS_IMPORT }],
            },
          },
        },
        { roleId: PLANNER.id },
      ),
    ]);

    await user.click(await screen.findByRole('checkbox', { name: /^Link financial transactions/ }));
    await user.click(screen.getByRole('button', { name: 'Save' }));
    await user.click(await screen.findByRole('link', { name: NEEDS_IMPORT }));

    const list = screen.getByRole('group', { name: 'Permissions' });
    expect(list).toHaveFocus();
    expect(list).toHaveAccessibleDescription(NEEDS_IMPORT);
  });
});

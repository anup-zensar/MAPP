import { screen, waitFor, within } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { allWorkOptions, savedView, standard2026 } from '@/testing/fixtures/all-work';
import { breakdown, BRANDS, SPEND_TYPE } from '@/testing/fixtures/all-work-breakdown';
import { renderAllWorkPage } from '@/testing/fixtures/all-work-page';
import { http, reply } from '@/testing/replies';
import { resetAllWorkStore } from './all-work-store';

const RANGE = '/all-work?yearFrom=2026&yearTo=2027';

const answers = [
  reply('GET /api/v1/all-work/options', {
    body: allWorkOptions({ attributes: [BRANDS, SPEND_TYPE] }),
  }),
  reply('GET /api/v1/all-work/breakdown', { body: breakdown() }),
];

beforeEach(() => {
  URL.createObjectURL = () => 'blob:export';
  URL.revokeObjectURL = () => {};
  vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {});
});

afterEach(() => {
  resetAllWorkStore();
  vi.restoreAllMocks();
});

async function breakdownByThen(user: ReturnType<typeof renderAllWorkPage>['user']) {
  await screen.findByRole('link', { name: 'Helix launch' });
  await user.click(screen.getByRole('tab', { name: 'Breakdown' }));
  await screen.findByRole('table', { name: 'Breakdown' });
  await user.click(screen.getByRole('button', { name: 'More filters' }));
  const brands = await screen.findByRole('group', { name: 'Brands' });
  await user.click(within(brands).getByRole('checkbox', { name: 'Helix' }));
  await user.keyboard('{Escape}');
  await user.selectOptions(screen.getByRole('combobox', { name: 'then by' }), 'activityType');
}

describe('016 FR-043: saved views and export carry the Breakdown', () => {
  it('saves the years, the Shell-attribute filter and both groupings with the view', async () => {
    const created = http.post('/api/v1/saved-views', async ({ request, response }) => {
      const body = await request.json();
      return response(201).json(savedView({ id: 'SV-1009', name: body.name, state: body.state }));
    });
    const { api, user } = renderAllWorkPage([...answers, created], RANGE);
    await breakdownByThen(user);

    await user.click(screen.getByRole('button', { name: 'Views' }));
    await user.click(await screen.findByRole('menuitem', { name: 'Save current view…' }));
    const dialog = await screen.findByRole('dialog', { name: 'Save current view' });
    await user.type(within(dialog).getByRole('textbox', { name: 'Name' }), 'Helix by spend');
    await user.click(within(dialog).getByRole('button', { name: 'Save view' }));

    await waitFor(() => expect(api.bodiesOf('POST /api/v1/saved-views')).toHaveLength(1));
    expect(api.bodiesOf('POST /api/v1/saved-views')[0]).toMatchObject({
      state: {
        view: 'breakdown',
        yearFrom: 2026,
        yearTo: 2027,
        breakdown: { by: 'attr:SA-1030', thenBy: 'activityType' },
        filters: { attributes: [{ attributeId: 'SA-1020', values: ['SO-1201'] }] },
      },
    });
  });

  it('opens a saved Breakdown with its groupings', async () => {
    const saved = savedView({
      name: 'By brand',
      state: {
        ...standard2026(),
        view: 'breakdown',
        breakdown: { by: 'attr:SA-1020', thenBy: 'plan' },
      },
    });
    const { api, user } = renderAllWorkPage([
      ...answers,
      reply('GET /api/v1/saved-views', { body: { items: [saved], total: 1 } }),
    ]);
    await screen.findByRole('link', { name: 'Helix launch' });

    await user.click(screen.getByRole('button', { name: /^View/ }));
    await user.click(await screen.findByRole('menuitemcheckbox', { name: 'By brand' }));

    await waitFor(() => {
      const query = api.requests.filter((r) => r.path === '/api/v1/all-work/breakdown').at(-1);
      const params = new URLSearchParams(query?.search);
      expect([params.get('by'), params.get('thenBy')]).toEqual(['attr:SA-1020', 'plan']);
    });
  });

  it('exports the Breakdown’s group table with its groupings and filters', async () => {
    const file = reply(
      'GET /api/v1/all-work/export',
      () =>
        new Response('file', {
          headers: { 'Content-Disposition': 'attachment; filename="All work - Breakdown.xlsx"' },
        }),
    );
    const { api, user } = renderAllWorkPage([...answers, file]);
    await breakdownByThen(user);

    await user.click(screen.getByRole('button', { name: 'Export' }));
    await user.click(await screen.findByRole('menuitem', { name: 'Spreadsheet (.xlsx)' }));

    await waitFor(() =>
      expect(api.requests.some((r) => r.path === '/api/v1/all-work/export')).toBe(true),
    );
    const request = api.requests.find((r) => r.path === '/api/v1/all-work/export');
    const query = new URLSearchParams(request?.search);
    expect({
      view: query.get('view'),
      by: query.get('by'),
      thenBy: query.get('thenBy'),
      attr: query.getAll('attr'),
    }).toEqual({
      view: 'breakdown',
      by: 'attr:SA-1030',
      thenBy: 'activityType',
      attr: ['SA-1020:SO-1201'],
    });
  });
});

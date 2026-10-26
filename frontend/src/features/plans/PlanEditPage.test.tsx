import { screen, waitFor } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import type { HttpHandler } from 'msw';
import { mockApi } from '@/testing/mock-api';
import { renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import { FORM_ANSWERS, plan } from '@/testing/fixtures/plans';
import { looseReply } from '@/testing/replies';
import { PlanEditPage } from './PlanEditPage';

const NO_EFFECTS = {
  modelChanged: false,
  disallowedActivities: [],
  hiddenGroups: [],
  shownGroups: [],
};

function renderPage(answers: HttpHandler[] = [], route = '/plans/P-1001/edit') {
  const api = mockApi([
    ...FORM_ANSWERS,
    looseReply('GET /api/v1/plans/{planId}', { body: plan() }, { planId: 'P-1001' }),
    looseReply('PATCH /api/v1/plans/{planId}', { body: plan() }, { planId: 'P-1001' }),
    looseReply(
      'POST /api/v1/plans/{planId}/change-effects',
      { body: NO_EFFECTS },
      { planId: 'P-1001' },
    ),
    ...answers,
  ]);
  const result = renderWithProviders(<PlanEditPage />, {
    user: testUser(),
    route,
    path: '/plans/:planId/edit',
  });
  return { ...result, api };
}

describe('PlanEditPage (US4, FR-018 – FR-023)', () => {
  it('opens the form with the year locked (US4#1, FR-018)', async () => {
    renderPage();

    expect(screen.getByRole('heading', { level: 1, name: 'Edit plan' })).toBeInTheDocument();
    expect(
      await screen.findByText(
        "Year can't change once the plan is saved — copy the plan into the next year or create a new plan.",
      ),
    ).toBeInTheDocument();
    expect(screen.getByRole('textbox', { name: /Plan name/ })).toHaveValue(
      'FY2026 Lubricants Europe',
    );
    expect(screen.getByRole('link', { name: 'FY2026 Lubricants Europe' })).toHaveAttribute(
      'href',
      '/plans/P-1001',
    );
    await waitFor(() => expect(document.title).toBe('Edit FY2026 Lubricants Europe · MAPP'));
  });

  it('saves the whole form with the version it started from', async () => {
    const { api, user, router } = renderPage();

    const budget = await screen.findByRole('textbox', { name: /Plan budget/ });
    await user.clear(budget);
    await user.type(budget, '1,500,000');
    await user.click(screen.getByRole('button', { name: 'Save' }));

    await waitFor(() => expect(router.state.location.pathname).toBe('/plans/P-1001'));
    expect(await screen.findByText("Plan 'FY2026 Lubricants Europe' saved")).toBeInTheDocument();
    expect(api.bodiesOf('PATCH /api/v1/plans/P-1001')).toEqual([
      expect.objectContaining({
        version: 1,
        budget: { amount: '1500000.00', currency: 'USD' },
        modelId: 'MOD-1001',
        teamIds: ['T-1001'],
      }),
    ]);
    expect(api.bodiesOf('POST /api/v1/plans/P-1001/change-effects')).toEqual([]);
  });

  it('lists the activities a new model does not allow before saving (US4#2, FR-019)', async () => {
    const { api, user, router } = renderPage([
      looseReply(
        'POST /api/v1/plans/{planId}/change-effects',
        {
          body: {
            ...NO_EFFECTS,
            modelChanged: true,
            disallowedActivities: [
              { type: 'activity', id: 'A-1003', name: 'Fleet webinar' },
              { type: 'activity', id: 'A-1004', name: 'Trade show' },
            ],
          },
        },
        { planId: 'P-1001' },
      ),
    ]);

    await user.selectOptions(await screen.findByRole('combobox', { name: /Model/ }), 'MOD-1002');
    await user.click(screen.getByRole('button', { name: 'Save' }));

    const dialog = await screen.findByRole('alertdialog', { name: 'Change the model?' });
    expect(dialog).toHaveTextContent("2 activities have a type the new model doesn't allow:");
    expect(screen.getByRole('link', { name: 'Fleet webinar' })).toHaveAttribute(
      'href',
      '/plans/P-1001/activities/A-1003',
    );
    expect(dialog).toHaveTextContent(
      'They keep their type but need an allowed type before they can be submitted. Goals will be recalculated with the new model.',
    );
    expect(api.bodiesOf('POST /api/v1/plans/P-1001/change-effects')).toEqual([
      { modelId: 'MOD-1002', attributes: plan().attributes },
    ]);

    await user.click(screen.getByRole('button', { name: 'Keep editing' }));
    expect(api.bodiesOf('PATCH /api/v1/plans/P-1001')).toEqual([]);

    await user.click(screen.getByRole('button', { name: 'Save' }));
    await user.click(await screen.findByRole('button', { name: 'Change model' }));
    await waitFor(() => expect(router.state.location.pathname).toBe('/plans/P-1001'));
    expect(api.bodiesOf('PATCH /api/v1/plans/P-1001')).toHaveLength(1);
  });

  it('says which attribute groups a business-unit change hides (US4#3, FR-020)', async () => {
    const { user } = renderPage([
      looseReply(
        'POST /api/v1/plans/{planId}/change-effects',
        {
          body: {
            ...NO_EFFECTS,
            hiddenGroups: [
              { unit: 'Lubricants', levels: ['plan', 'activity'], attributes: ['Lubricant type'] },
            ],
          },
        },
        { planId: 'P-1001' },
      ),
    ]);

    await user.click(await screen.findByRole('checkbox', { name: 'Lubricants' }));
    await user.click(screen.getByRole('button', { name: 'Save' }));

    const dialog = await screen.findByRole('alertdialog', { name: 'Save these changes?' });
    expect(dialog).toHaveTextContent(
      'Lubricants attributes will be hidden on the plan and its activities. Their values are kept and come back if you add Lubricants again.',
    );
    expect(screen.getByRole('button', { name: 'Save changes' })).toBeInTheDocument();
  });

  it('names the owner who is in none of the chosen teams (US4#4, FR-021)', async () => {
    const { user } = renderPage();

    await user.click(await screen.findByRole('combobox', { name: /Teams/ }));
    await user.click(await screen.findByRole('option', { name: /Mobility China/ }));
    await user.click(screen.getByRole('button', { name: 'Remove Lubricants Europe' }));

    expect(
      await screen.findByText(
        "Priya Raman isn't in any of the chosen teams — choose another owner or clear the field",
      ),
    ).toBeInTheDocument();
  });

  it('says a published plan stays published and when changed defaults apply (US4#5, FR-022)', async () => {
    const { user } = renderPage([
      looseReply(
        'GET /api/v1/plans/{planId}',
        { body: plan({ status: 'published' }) },
        { planId: 'P-1001' },
      ),
    ]);

    expect(
      await screen.findByText(
        'This plan is published. Changes are saved straight away and recorded in the activity log.',
      ),
    ).toBeInTheDocument();
    await user.selectOptions(
      screen.getByRole('combobox', { name: 'Default line-item spread' }),
      'lastMonth',
    );
    expect(screen.getByText('Applies to line items created from now on.')).toBeInTheDocument();
  });

  it('refuses a save after someone else’s, keeping the typed values (FR-023)', async () => {
    let reads = 0;
    let saves = 0;
    const { api, user } = renderPage([
      looseReply(
        'GET /api/v1/plans/{planId}',
        () => {
          reads += 1;
          return { body: plan({ version: reads === 1 ? 1 : 2 }) };
        },
        { planId: 'P-1001' },
      ),
      looseReply(
        'PATCH /api/v1/plans/{planId}',
        () => {
          saves += 1;
          return saves === 1
            ? {
                status: 409,
                body: {
                  error: {
                    code: 'version_conflict',
                    message: 'Not saved — Daniel Okoro changed this plan',
                    fields: [],
                    conflict: { changedBy: 'Daniel Okoro', changedAt: '2026-03-12T09:05:00Z' },
                  },
                },
              }
            : { body: plan({ name: 'FY2026 Lubricants Europe (new)', version: 3 }) };
        },
        { planId: 'P-1001' },
      ),
    ]);

    const name = await screen.findByRole('textbox', { name: /Plan name/ });
    await user.clear(name);
    await user.type(name, 'FY2026 Lubricants Europe (new)');
    await user.click(screen.getByRole('button', { name: 'Save' }));

    expect(
      await screen.findByText('Not saved — Daniel Okoro changed this plan on 12 Mar 2026 at 09:05'),
    ).toBeInTheDocument();
    expect(screen.getByRole('textbox', { name: /Plan name/ })).toHaveValue(
      'FY2026 Lubricants Europe (new)',
    );
    await user.click(screen.getByRole('button', { name: 'Save' }));

    await waitFor(() => expect(saves).toBe(2));
    expect(
      api
        .bodiesOf('PATCH /api/v1/plans/P-1001')
        .map((body) => (body as { version: number }).version),
    ).toEqual([1, 2]);
  });

  it('focuses the field named in the address once the form is drawn', async () => {
    renderPage([], '/plans/P-1001/edit?focus=description');

    const description = await screen.findByRole('textbox', { name: /Description/ });
    await waitFor(() => expect(description).toHaveFocus());
  });

  it('asks before typed changes are discarded', async () => {
    const { user } = renderPage();

    await user.type(await screen.findByRole('textbox', { name: /Description/ }), ' More.');
    await user.click(screen.getByRole('button', { name: 'Cancel' }));

    expect(
      await screen.findByRole('alertdialog', { name: 'Discard changes?' }),
    ).toBeInTheDocument();
  });

  it('says when the plan does not exist', async () => {
    renderPage([
      looseReply(
        'GET /api/v1/plans/{planId}',
        {
          status: 404,
          body: {
            error: {
              code: 'not_found',
              message: 'This plan no longer exists — someone may have deleted it.',
              fields: [],
            },
          },
        },
        { planId: 'P-1001' },
      ),
    ]);

    expect(
      await screen.findByRole('heading', { level: 1, name: 'Plan not found' }),
    ).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Go to Plans' })).toBeInTheDocument();
  });
});

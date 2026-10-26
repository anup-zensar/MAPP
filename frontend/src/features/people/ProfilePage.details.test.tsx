import { screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import type { CurrentUser } from '@/session/useSession';
import { validation } from '@/testing/fixtures/people';
import { card, PRIYA_AS_PERSON, renderProfile } from '@/testing/fixtures/profile';
import { testUser } from '@/testing/fixtures/users';
import { reply } from '@/testing/replies';

describe('ProfilePage (feature 003, US4): Edit your details', () => {
  it('saves a new name and job title; the account shows them at once (US4-2, FR-022)', async () => {
    const renamed: CurrentUser = testUser({
      name: 'Priya Iyer',
      initials: 'PI',
      jobTitle: 'Senior marketer',
      version: 2,
    });
    const { api, user } = renderProfile([
      reply('PATCH /api/v1/me', { body: renamed }),
      reply('GET /api/v1/me', { body: renamed }),
    ]);

    await user.click(screen.getByRole('button', { name: 'Edit details' }));
    const panel = await screen.findByRole('dialog', { name: 'Edit your details' });
    const name = within(panel).getByRole('textbox', { name: /Name/ });
    expect(name).toHaveValue('Priya Raman');
    await user.clear(name);
    await user.type(name, 'Priya Iyer');
    const jobTitle = within(panel).getByRole('textbox', { name: /Job title/ });
    await user.clear(jobTitle);
    await user.type(jobTitle, 'Senior marketer');
    await user.click(within(panel).getByRole('button', { name: 'Save' }));

    expect(await screen.findByText('Your details saved')).toBeInTheDocument();
    expect(api.bodiesOf('PATCH /api/v1/me')).toEqual([
      { name: 'Priya Iyer', jobTitle: 'Senior marketer', version: 1 },
    ]);
    expect(screen.queryByRole('dialog', { name: 'Edit your details' })).not.toBeInTheDocument();
    expect(screen.getByText('Account: Priya Iyer (PI) · a date: 12 Mar 2026')).toBeInTheDocument();
    expect(card('Your details')).toHaveTextContent('Senior marketer');
  });

  it('shows "Enter a name" at the field and in the summary when the name is empty (US4-2, SC-007)', async () => {
    const { user } = renderProfile([
      reply('PATCH /api/v1/me', validation([{ field: 'name', message: 'Enter a name.' }])),
    ]);

    await user.click(screen.getByRole('button', { name: 'Edit details' }));
    const panel = await screen.findByRole('dialog', { name: 'Edit your details' });
    await user.clear(within(panel).getByRole('textbox', { name: /Name/ }));
    await user.click(within(panel).getByRole('button', { name: 'Save' }));

    expect(await within(panel).findAllByText('Enter a name.')).toHaveLength(2);
    expect(screen.queryByText('Your details saved')).not.toBeInTheDocument();
    expect(screen.getByText(/^Account: Priya Raman \(PR\)/)).toBeInTheDocument();
  });

  it('keeps typed values after a conflict and saves over it with the newer version (FR-022)', async () => {
    let patches = 0;
    const saved = testUser({ jobTitle: 'Brand lead', version: 3 });
    const { api, user } = renderProfile([
      reply('PATCH /api/v1/me', () => {
        patches += 1;
        return patches === 1
          ? {
              status: 409,
              body: {
                error: {
                  code: 'version_conflict',
                  message: 'Not saved — Alex Morgan changed this person.',
                  fields: [],
                  conflict: { changedBy: 'Alex Morgan', changedAt: '2026-03-10T14:32:00Z' },
                },
              },
            }
          : { body: saved };
      }),
      reply('GET /api/v1/me', () => ({
        body: patches === 1 ? testUser({ version: 2 }) : saved,
      })),
      reply('GET /api/v1/people/{personId}', { body: PRIYA_AS_PERSON }, { personId: 'U-1002' }),
    ]);

    await user.click(screen.getByRole('button', { name: 'Edit details' }));
    const panel = await screen.findByRole('dialog', { name: 'Edit your details' });
    const jobTitle = within(panel).getByRole('textbox', { name: /Job title/ });
    await user.clear(jobTitle);
    await user.type(jobTitle, 'Brand lead');
    await user.click(within(panel).getByRole('button', { name: 'Save' }));

    expect(
      await within(panel).findByText(/^Not saved — Alex Morgan changed this profile on/),
    ).toBeInTheDocument();
    expect(jobTitle).toHaveValue('Brand lead');

    await user.click(within(panel).getByRole('button', { name: 'Save' }));
    expect(await screen.findByText('Your details saved')).toBeInTheDocument();
    expect(api.bodiesOf('PATCH /api/v1/me')).toEqual([
      { name: 'Priya Raman', jobTitle: 'Brand lead', version: 1 },
      { name: 'Priya Raman', jobTitle: 'Brand lead', version: 2 },
    ]);
  });
});

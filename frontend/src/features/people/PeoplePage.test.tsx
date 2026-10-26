import { screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { openActions, PEOPLE, renderPeople, rowOf } from '@/testing/fixtures/people';
import { reply } from '@/testing/replies';

describe('PeoplePage (feature 003, US1): the list', () => {
  it('lists the active people first, and everyone under All (FR-001)', async () => {
    const { user } = renderPeople();

    expect(await screen.findByText('4 of 5 people')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Active 4' })).toHaveAttribute(
      'aria-pressed',
      'true',
    );
    expect(screen.queryByRole('row', { name: /^Jane Doe/ })).not.toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'All 5' }));
    expect(screen.getByText('5 of 5 people')).toBeInTheDocument();
    expect(screen.getByRole('heading', { level: 1, name: 'People' })).toBeInTheDocument();
    expect(screen.getByRole('navigation', { name: 'Breadcrumb' })).toHaveTextContent('Settings');
    const priya = rowOf('Priya Raman');
    expect(within(priya).getByRole('link', { name: 'Priya Raman (you)' })).toHaveAttribute(
      'href',
      '/settings/people/U-1002',
    );
    expect(priya).toHaveTextContent('priya.raman@shell.com');
    expect(priya).toHaveTextContent('Lubricants Europe');
    expect(priya).toHaveTextContent('Active');
    expect(rowOf('Daniel Okoro')).toHaveTextContent('—');
    const jane = rowOf('Jane Doe');
    expect(jane).toHaveTextContent('Jane Doe (inactive)');
    expect(jane).toHaveTextContent('No team');
    expect(jane).toHaveTextContent('Inactive');
  });

  it('searches name, e-mail and job title, filters by status, and clears the filters', async () => {
    const { user } = renderPeople();
    const search = await screen.findByRole('searchbox', {
      name: 'Search name, e-mail or job title',
    });

    await user.type(search, 'finance');
    expect(screen.getByText('1 of 5 people')).toBeInTheDocument();
    expect(rowOf('Sara Novak')).toBeInTheDocument();

    await user.clear(search);
    await user.type(search, 'okoro@');
    expect(rowOf('Daniel Okoro')).toBeInTheDocument();
    expect(screen.getAllByRole('row')).toHaveLength(2);

    await user.clear(search);
    await user.type(search, 'ALEX');
    expect(rowOf('Alex Morgan')).toBeInTheDocument();

    await user.clear(search);
    await user.click(screen.getByRole('button', { name: /^Inactive/ }));
    expect(screen.getByText('1 of 5 people')).toBeInTheDocument();
    expect(rowOf('Jane Doe')).toBeInTheDocument();

    await user.type(search, 'sara');
    expect(screen.getByText('Nothing matches these filters')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Clear filters' }));
    expect(screen.getByText('5 of 5 people')).toBeInTheDocument();
  });

  it('shows how to add the first person when there is none', async () => {
    renderPeople([reply('GET /api/v1/people', { body: { items: [], total: 0 } })]);

    expect(await screen.findByText('No people yet — add the first person')).toBeInTheDocument();
    expect(screen.getAllByRole('button', { name: 'New person' })).toHaveLength(2);
  });

  it('says when the people could not be loaded, with Try again', async () => {
    let calls = 0;
    const { user } = renderPeople([
      reply('GET /api/v1/people', () => {
        calls += 1;
        return calls === 1 ? { status: 500 } : { body: PEOPLE };
      }),
    ]);

    await user.click(await screen.findByRole('button', { name: 'Try again' }));

    expect(await screen.findByText('4 of 5 people')).toBeInTheDocument();
  });

  it('has no Delete; Deactivate sits in its place with the tip (US1-8, FR-008)', async () => {
    const { user } = renderPeople();

    const menu = await openActions(user, 'Alex Morgan');

    expect(
      within(menu)
        .getAllByRole('menuitem')
        .map((item) => item.firstChild?.textContent),
    ).toEqual(['View', 'Edit', 'Deactivate']);
    expect(within(menu).queryByRole('menuitem', { name: /Delete/ })).not.toBeInTheDocument();
    expect(within(menu).getByRole('menuitem', { name: 'Deactivate' })).toHaveAccessibleDescription(
      'People are never deleted, so the history keeps their name.',
    );
  });
});

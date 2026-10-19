import { screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { COMING_SOON_SECTIONS } from '@/app/navigation';
import { renderWithProviders } from '@/test/render';
import { ComingSoonPage } from './ComingSoonPage';

function section(label: string) {
  const found = COMING_SOON_SECTIONS.find((item) => item.label === label);
  if (!found) {
    throw new Error(`There is no section called ${label}.`);
  }
  return found;
}

describe('ComingSoonPage', () => {
  // The sentences of the plan's "Coming soon" table (FR-015), word for word.
  it.each([
    [
      'Plans',
      "Create yearly marketing plans, give them budgets and publish them when they're ready.",
    ],
    [
      'Approvals',
      'Approve or decline the activities waiting for you, and follow the ones you sent.',
    ],
    [
      'All work',
      'See every campaign, initiative, activity and line item across plans as a grid, a calendar or a budget view.',
    ],
    [
      'Settings',
      'Set up people and teams, models, activity types, Shell attributes, approval workflows, currencies and finance budget items.',
    ],
    [
      'Recycle bin',
      'Bring back anything that was deleted, together with everything that was under it.',
    ],
    ['Activity log', 'See who changed what, and when, in every plan and setting.'],
    ['My profile', 'See your details and teams, and choose how dates and money are shown to you.'],
  ])('says in one sentence what %s will do (FR-015, US2 scenario 2)', (label, sentence) => {
    renderWithProviders(<ComingSoonPage section={section(label)} />);

    expect(screen.getByRole('heading', { level: 1, name: label })).toBeInTheDocument();
    expect(screen.getByText(sentence)).toBeInTheDocument();
    expect(document.title).toBe(`${label} · NorthStar`);
  });

  it('says that the section is coming soon, with the section name as the eyebrow (FR-015)', () => {
    renderWithProviders(<ComingSoonPage section={section('Approvals')} />);

    expect(screen.getAllByText('Approvals')).toHaveLength(2);
    expect(screen.getByText('Coming soon')).toBeInTheDocument();
    expect(screen.getByText("This part of NorthStar isn't built yet.")).toBeInTheDocument();
  });

  it('offers a way back to Home (FR-015)', async () => {
    const { user, router } = renderWithProviders(<ComingSoonPage section={section('Plans')} />, {
      route: '/plans',
    });

    await user.click(screen.getByRole('link', { name: 'Go to Home' }));

    expect(router.state.location.pathname).toBe('/');
  });
});

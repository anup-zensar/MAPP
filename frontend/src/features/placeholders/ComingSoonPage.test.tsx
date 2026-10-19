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
  // The sentences of the plan's "Coming soon" table (FR-015), word for word, for the sections
  // that later features build (009–012); Plans, Approvals, Settings, My profile, Recycle bin and
  // Activity log (011) have pages.
  it.each([
    [
      'All work',
      'See every campaign, initiative, activity and line item across plans as a grid, a calendar or a budget view.',
    ],
  ])('says in one sentence what %s will do (FR-015, US2 scenario 2)', (label, sentence) => {
    renderWithProviders(<ComingSoonPage section={section(label)} />);

    expect(screen.getByRole('heading', { level: 1, name: label })).toBeInTheDocument();
    expect(screen.getByText(sentence)).toBeInTheDocument();
    expect(document.title).toBe(`${label} · NorthStar`);
  });

  it('is only for the sections still to come (Plan-module foundation)', () => {
    expect(COMING_SOON_SECTIONS.map((item) => item.label)).toEqual(['All work']);
  });

  it('says that the section is coming soon, with the section name as the eyebrow (FR-015)', () => {
    renderWithProviders(<ComingSoonPage section={section('All work')} />);

    expect(screen.getAllByText('All work')).toHaveLength(2);
    expect(screen.getByText('Coming soon')).toBeInTheDocument();
    expect(screen.getByText("This part of NorthStar isn't built yet.")).toBeInTheDocument();
  });

  it('offers a way back to Home (FR-015)', async () => {
    const { user, router } = renderWithProviders(<ComingSoonPage section={section('All work')} />, {
      route: '/all-work',
    });

    await user.click(screen.getByRole('link', { name: 'Go to Home' }));

    expect(router.state.location.pathname).toBe('/');
  });
});

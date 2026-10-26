import { screen } from '@testing-library/react';
import { Sparkles } from 'lucide-react';
import { describe, expect, it } from 'vitest';
import { renderWithProviders } from '@/testing/render';
import { ComingSoonPage } from './ComingSoonPage';
import type { ComingSoonSection } from './ComingSoonPage';

/**
 * Every section of the Plan module has its page since 011 built Recycle bin and Activity log, so
 * the page is shown here with a made-up section: it stays for a section a later module adds.
 */
const LATER: ComingSoonSection = {
  label: 'Later section',
  icon: Sparkles,
  comingSoon: 'Do something useful that is not built yet.',
};

describe('ComingSoonPage', () => {
  it('says in one sentence what the section will do (FR-015, US2 scenario 2)', () => {
    renderWithProviders(<ComingSoonPage section={LATER} />);

    expect(screen.getByRole('heading', { level: 1, name: 'Later section' })).toBeInTheDocument();
    expect(screen.getByText('Do something useful that is not built yet.')).toBeInTheDocument();
    expect(document.title).toBe('Later section · MAPP');
  });

  it('says that the section is coming soon, under the section name as the title (FR-015)', () => {
    renderWithProviders(<ComingSoonPage section={LATER} />);

    // The eyebrow is the title itself, so it is shown once (Shell look).
    expect(screen.getAllByText('Later section')).toHaveLength(1);
    expect(screen.getByText('Coming soon')).toBeInTheDocument();
    expect(screen.getByText("This part of MAPP isn't built yet.")).toBeInTheDocument();
  });

  it('offers a way back to Home (FR-015)', async () => {
    const { user, router } = renderWithProviders(<ComingSoonPage section={LATER} />, {
      route: '/later-section',
    });

    await user.click(screen.getByRole('link', { name: 'Go to Home' }));

    expect(router.state.location.pathname).toBe('/');
  });
});

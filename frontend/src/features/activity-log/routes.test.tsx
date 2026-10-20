import { screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { expectCurrentSection, renderApp } from '@/test/app';
import { logPage, logSummary } from './testData';

// Feature 011's address for the Activity log (build guide §5), under More in the sidebar.
describe('Activity log route (feature 011)', () => {
  it('opens /activity-log: "Activity log", current in the sidebar, no longer Coming soon (FR-001)', async () => {
    renderApp('/activity-log', {
      'GET /api/activity-log/summary': { body: logSummary() },
      'GET /api/activity-log': { body: logPage([]) },
    });

    expect(
      await screen.findByRole('heading', { level: 1, name: 'Every change, on the record.' }),
    ).toBeInTheDocument();
    expect(screen.queryByText('Coming soon')).not.toBeInTheDocument();
    expect(document.title).toBe('Activity log · NorthStar');
    expectCurrentSection('Activity log');
  });
});

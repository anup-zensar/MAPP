import { screen } from '@testing-library/react';
import type { HttpHandler } from 'msw';
import type { ReactElement } from 'react';
import { describe, expect, it } from 'vitest';
import { ApprovalWorkflowsPage } from '@/features/approval-workflows';
import { PeoplePage, TeamsPage } from '@/features/people';
import { PEOPLE as WORKFLOW_PEOPLE } from '@/testing/fixtures/approval-workflows';
import { workflowList } from '@/testing/fixtures/approval-workflows-page';
import { PEOPLE } from '@/testing/fixtures/people';
import { TEAMS } from '@/testing/fixtures/teams';
import { VIEWER } from '@/testing/fixtures/viewer';
import { expectNoAxeViolations } from '@/testing/axe';
import { mockApi } from '@/testing/mock-api';
import { renderWithProviders } from '@/testing/render';
import { reply } from '@/testing/replies';

const NOTE =
  'View only — only administrators change Settings. Ask Alex Morgan if something needs to change.';

const PAGES: [string, ReactElement, string, HttpHandler[]][] = [
  [
    'People',
    <PeoplePage key="people" />,
    '/settings/people/:personId?',
    [reply('GET /api/v1/people', { body: PEOPLE }), reply('GET /api/v1/teams', { body: TEAMS })],
  ],
  [
    'Teams',
    <TeamsPage key="teams" />,
    '/settings/teams/:teamId?',
    [reply('GET /api/v1/teams', { body: TEAMS }), reply('GET /api/v1/people', { body: PEOPLE })],
  ],
  [
    'Approval workflows',
    <ApprovalWorkflowsPage key="workflows" />,
    '/settings/approval-workflows/:workflowId?',
    [
      reply('GET /api/v1/approval-workflows', workflowList()),
      reply('GET /api/v1/people', WORKFLOW_PEOPLE),
    ],
  ],
];

describe('014 FR-011: People, Teams and Approval workflows for someone who may not change them', () => {
  it.each(PAGES)(
    '%s: one View only note and no change action',
    async (_name, page, path, answers) => {
      mockApi(answers);
      renderWithProviders(page, { user: VIEWER, path, route: path.split('/:')[0] });

      expect(await screen.findAllByRole('table')).not.toHaveLength(0);
      expect(screen.getByRole('note')).toHaveTextContent(NOTE);
      expect(screen.queryByRole('button', { name: /^New / })).not.toBeInTheDocument();
      expect(screen.queryByRole('link', { name: /^New / })).not.toBeInTheDocument();
      await expectNoAxeViolations();
    },
  );
});

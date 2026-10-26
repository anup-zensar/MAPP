import { screen } from '@testing-library/react';
import type { ComponentProps } from 'react';
import { vi } from 'vitest';
import type { HttpHandler } from 'msw';
import { ApprovalSection, DecisionDialog } from '@/features/approvals';
import type { ApprovalActivity } from '@/features/approvals';
import { mockApi } from '../mock-api';
import { renderWithProviders } from '../render';
import { reply } from '../replies';
import type { Answer } from '../replies';
import { approvalOf } from './approvals';
import { testUser } from './users';

type DecisionDialogProps = ComponentProps<typeof DecisionDialog>;

const ACTIVITY: ApprovalActivity = {
  id: 'A-1042',
  name: 'Launch film',
  status: 'submitted',
  version: 3,
};

export const NONE = { available: false, reason: '' };

/** The Approval section of "Launch film", whose approval `answer` gives. */
export function renderSection(
  answer: Answer<'GET /api/v1/activities/{activityId}/approval'>,
  extra: HttpHandler[] = [],
) {
  const api = mockApi([
    reply('GET /api/v1/activities/{activityId}/approval', answer, { activityId: 'A-1042' }),
    ...extra,
  ]);
  const rendered = renderWithProviders(
    <ApprovalSection activity={ACTIVITY} onShowProblems={vi.fn()} />,
    { user: testUser() },
  );
  return { ...rendered, api };
}

export const card = () => screen.getByRole('region', { name: 'Approval' });

export function renderDialog(
  props: Partial<DecisionDialogProps> & Pick<DecisionDialogProps, 'mode'>,
  answers: HttpHandler[] = [],
) {
  const api = mockApi(answers);
  const onClose = vi.fn();
  const onDecided = vi.fn();
  const rendered = renderWithProviders(
    <DecisionDialog approval={approvalOf()} onClose={onClose} onDecided={onDecided} {...props} />,
    { user: testUser() },
  );
  return { ...rendered, api, onClose, onDecided };
}

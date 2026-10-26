import { screen, waitFor } from '@testing-library/react';
import { useState } from 'react';
import { describe, expect, it } from 'vitest';
import { api, call } from '@/api/client';
import { ApiError } from '@/api/errors';
import { useAccess } from './access';
import { mockApi } from '@/testing/mockApi';
import { renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/users';
import type { CurrentUser } from './useSession';

const FORBIDDEN = {
  error: {
    code: 'forbidden',
    message:
      "Changing activities and line items isn't part of your role (Viewer). Ask an administrator if you need it.",
    fields: [],
  },
};

/** Sara Novak while she is a Viewer, and after an administrator made her a Planner. */
const VIEWER: CurrentUser = testUser({
  id: 'U-1006',
  name: 'Sara Novak',
  access: {
    roles: [],
    permissions: [{ key: 'plans.view', all: true, teams: [] }],
    administrators: [],
    noSelfApproval: true,
  },
});
const PLANNER: CurrentUser = {
  ...VIEWER,
  access: {
    ...VIEWER.access!,
    permissions: [
      { key: 'plans.view', all: true, teams: [] },
      { key: 'activities.edit', all: true, teams: [] },
    ],
  },
};

/** A page that tries a change and shows what the person may do. */
function ChangePage() {
  const { has } = useAccess();
  const [failure, setFailure] = useState('');
  async function change() {
    try {
      await call(
        api.DELETE('/api/v1/activities/{activityId}', {
          params: { path: { activityId: 'A-1001' } },
        }),
      );
    } catch (error) {
      setFailure(error instanceof ApiError ? `${error.kind}: ${error.message}` : 'other');
    }
  }
  return (
    <>
      <p>{has('activities.edit') ? 'May change activities' : 'View only'}</p>
      <button type="button" onClick={() => void change()}>
        Delete
      </button>
      <p role="status">{failure}</p>
    </>
  );
}

describe('SessionProvider after a 403', () => {
  it('reloads the person’s access when a call is refused by their roles (feature 014, FR-020)', async () => {
    const mocked = mockApi({
      'DELETE /api/v1/activities/A-1001': { status: 403, body: FORBIDDEN },
      'GET /api/v1/me': { body: PLANNER },
    });
    const { user } = renderWithProviders(<ChangePage />, { user: VIEWER });
    expect(screen.getByText('View only')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Delete' }));

    expect(await screen.findByRole('status')).toHaveTextContent(
      `forbidden: ${FORBIDDEN.error.message}`,
    );
    await waitFor(() => expect(screen.getByText('May change activities')).toBeInTheDocument());
    expect(
      mocked.requests.filter((r) => r.method === 'GET' && r.path === '/api/v1/me'),
    ).toHaveLength(1);
  });
});

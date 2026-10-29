import { Navigate } from 'react-router';
import type { RouteObject } from 'react-router';
import type { PageHandle } from '@/session/page-handle';
import { RequirePermission } from '@/session/RequirePermission';
import { DashboardsPage } from './pages';

// Feature 036's pages, apart from routes.tsx only to keep that file within 300 lines.
export const DASHBOARD_ROUTES: RouteObject[] = [
  { path: '/dashboards', element: <Navigate to="/dashboards/executive" replace /> },
  ...(['executive', 'regional'] as const).map((kind) => ({
    path: `/dashboards/${kind}`,
    element: (
      <RequirePermission permission="plans.view">
        <DashboardsPage kind={kind} />
      </RequirePermission>
    ),
    handle: {
      title: kind === 'executive' ? 'Executive dashboard' : 'Regional dashboard',
    } satisfies PageHandle,
  })),
];

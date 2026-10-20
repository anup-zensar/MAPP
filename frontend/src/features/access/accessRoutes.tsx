import type { RouteObject } from 'react-router';
import { AccessPage } from './AccessPage';
import { RoleFormPage } from './RoleFormPage';
import { RolePage } from './RolePage';

/**
 * The pages of Roles and access (014 plan, "Routes"), for `src/app/router.tsx` to add to the
 * Plan module's routes (`...ACCESS_ROUTES`). The guard (access.manage) comes with 014's
 * RequirePermission; each page names itself for the browser tab.
 */
export const ACCESS_ROUTES: RouteObject[] = [
  { path: '/settings/access', element: <AccessPage />, handle: { title: 'Roles and access' } },
  { path: '/settings/access/roles/new', element: <RoleFormPage />, handle: { title: 'New role' } },
  { path: '/settings/access/roles/:roleId', element: <RolePage />, handle: { title: 'Role' } },
  {
    path: '/settings/access/roles/:roleId/edit',
    element: <RoleFormPage />,
    handle: { title: 'Edit role' },
  },
];

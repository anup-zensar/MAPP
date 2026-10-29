import type { RouteObject } from 'react-router';
import { ImportPage, ImportResultPage } from './pages';

export const WORKBOOK_ROUTES: RouteObject[] = [
  {
    path: '/plans/:planId/import',
    element: <ImportPage />,
    handle: { title: 'Import from Excel' },
  },
  {
    path: '/imports/:importId',
    element: <ImportResultPage />,
    handle: { title: 'Import results' },
  },
];

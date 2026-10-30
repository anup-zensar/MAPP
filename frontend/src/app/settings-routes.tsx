import type { RouteObject } from 'react-router';
import {
  AccessPage,
  ActivityTypesPage,
  ApprovalWorkflowsPage,
  CurrenciesPage,
  FieldHelpSettingsPage,
  FinanceBudgetItemsPage,
  ImportRatesPage,
  ModelsPage,
  OverspendSettingsPage,
  PeoplePage,
  PurchaseRequestSettingsPage,
  RoleFormPage,
  RolePage,
  SettingsPage,
  ShellAttributeEditorPage,
  ShellAttributePage,
  ShellAttributesPage,
  TeamsPage,
} from './pages';
import { gated, titled } from './route-helpers';

// The Settings pages, apart from routes.tsx to keep that file within 300 lines.
export const SETTINGS_ROUTES: RouteObject[] = [
  { path: '/settings', element: <SettingsPage />, ...titled('Settings') },
  { path: '/settings/models', element: <ModelsPage />, ...titled('Models') },
  { path: '/settings/activity-types', element: <ActivityTypesPage />, ...titled('Activity types') },
  {
    path: '/settings/shell-attributes',
    element: <ShellAttributesPage />,
    ...titled('Shell attributes'),
  },
  {
    path: '/settings/shell-attributes/new',
    element: gated('settings.manage', <ShellAttributeEditorPage mode="create" />),
    ...titled('New Shell attribute'),
  },
  {
    path: '/settings/shell-attributes/:attributeId',
    element: <ShellAttributePage />,
    ...titled('Shell attribute'),
  },
  {
    path: '/settings/shell-attributes/:attributeId/edit',
    element: gated('settings.manage', <ShellAttributeEditorPage mode="edit" />),
    ...titled('Edit Shell attribute'),
  },
  { path: '/settings/currencies', element: <CurrenciesPage />, ...titled('Currencies') },
  {
    path: '/settings/finance-budget-items',
    element: <FinanceBudgetItemsPage />,
    ...titled('Finance budget items'),
  },
  { path: '/settings/people/:personId?', element: <PeoplePage />, ...titled('People') },
  { path: '/settings/teams/:teamId?', element: <TeamsPage />, ...titled('Teams') },
  {
    path: '/settings/approval-workflows/:workflowId?',
    element: <ApprovalWorkflowsPage />,
    ...titled('Approval workflows'),
  },
  {
    path: '/settings/purchase-requests',
    element: <PurchaseRequestSettingsPage />,
    ...titled('Purchase requests'),
  },
  {
    path: '/settings/overspend-control',
    element: <OverspendSettingsPage />,
    ...titled('Overspend control'),
  },
  {
    path: '/settings/currencies/import',
    element: gated('exchangeRates.manage', <ImportRatesPage />),
    ...titled('Import GSAP rates'),
  },
  { path: '/settings/field-help', element: <FieldHelpSettingsPage />, ...titled('Field help') },
  // Roles and access only for people whose role includes access.manage.
  ...[
    { path: '/settings/access', element: <AccessPage />, ...titled('Roles and access') },
    { path: '/settings/access/roles/new', element: <RoleFormPage />, ...titled('New role') },
    { path: '/settings/access/roles/:roleId', element: <RolePage />, ...titled('Role') },
    {
      path: '/settings/access/roles/:roleId/edit',
      element: <RoleFormPage />,
      ...titled('Edit role'),
    },
  ].map((route) => ({
    ...route,
    element: gated('access.manage', route.element),
  })),
];

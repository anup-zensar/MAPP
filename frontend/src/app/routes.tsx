import { Outlet } from 'react-router';
import type { RouteObject } from 'react-router';
import { ComingSoonPage } from '@/features/placeholders';
import { RequireSignIn } from '@/session/RequireSignIn';
import { AppLayout } from './AppLayout';
import { DASHBOARD_ROUTES } from './dashboard-routes';
import { DevOnly } from './DevOnly';
import { COMING_SOON_SECTIONS } from './navigation';
import { NotFoundRoute } from './NotFoundRoute';
import { WORKBOOK_ROUTES } from './routes-workbook';
import {
  AccessPage,
  ActivityFormPage,
  ActivityLogPage,
  ActivityPage,
  ActivityShortLink,
  ActivityTypesPage,
  AllWorkPage,
  ApprovalWorkflowsPage,
  ApprovalsPage,
  CampaignPanel,
  ComponentGalleryPage,
  CurrenciesPage,
  FieldHelpSettingsPage,
  FinanceBudgetItemsPage,
  FinancialsPage,
  GlobalBudgetPage,
  GlobalBudgetsPage,
  HomePage,
  ImportRatesPage,
  InitiativePanel,
  LineItemCreatePage,
  LineItemEditPage,
  LineItemPage,
  ModelsPage,
  NewCampaignPanel,
  NewInitiativePanel,
  OverspendSettingsPage,
  PeoplePage,
  PlanCreatePage,
  PlanEditPage,
  PlanPage,
  PlansPage,
  ProfilePage,
  PurchaseRequestCreatePage,
  PurchaseRequestEditPage,
  PurchaseRequestPage,
  PurchaseRequestSettingsPage,
  PurchaseRequestsPage,
  RecycleBinPage,
  RoleFormPage,
  RolePage,
  SampleRecordsPage,
  SettingsPage,
  ShellAttributeEditorPage,
  ShellAttributePage,
  ShellAttributesPage,
  SignInPage,
  SpaPage,
  TeamsPage,
  TransactionPanel,
} from './pages';
import { RootLayout } from './RootLayout';
import { gated, titled } from './route-helpers';
import { RouteErrorPage } from './RouteErrorPage';

/** Every page of the Plan module (features 002–016; docs/sdd/plan-module-build.md §5). */
const PLAN_MODULE_ROUTES: RouteObject[] = [
  // 002 Settings
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
  { path: '/profile', element: <ProfilePage />, ...titled('My profile') },
  // 004 Plans; the plan page holds 005's campaign and initiative panels (child routes)
  { path: '/plans', element: <PlansPage />, ...titled('Plans') },
  { path: '/plans/new', element: <PlanCreatePage />, ...titled('New plan') },
  {
    path: '/plans/:planId',
    element: <PlanPage />,
    ...titled('Plan'),
    children: [
      { path: 'campaigns/new', element: <NewCampaignPanel />, ...titled('Plans') },
      {
        path: 'campaigns/:campaignId',
        element: <CampaignPanel mode="details" />,
        ...titled('Plans'),
      },
      {
        path: 'campaigns/:campaignId/edit',
        element: <CampaignPanel mode="edit" />,
        ...titled('Plans'),
      },
      {
        path: 'campaigns/:campaignId/initiatives/new',
        element: <NewInitiativePanel />,
        ...titled('Plans'),
      },
      {
        path: 'initiatives/:initiativeId',
        element: <InitiativePanel mode="details" />,
        ...titled('Plans'),
      },
      {
        path: 'initiatives/:initiativeId/edit',
        element: <InitiativePanel mode="edit" />,
        ...titled('Plans'),
      },
    ],
  },
  { path: '/plans/:planId/edit', element: <PlanEditPage />, ...titled('Edit plan') },
  { path: '/plans/:planId/spa', element: <SpaPage />, ...titled('SP&A view') },
  // 006 Activities, and the short address that finds an activity's plan (D-194)
  {
    path: '/plans/:planId/initiatives/:initiativeId/activities/new',
    element: <ActivityFormPage mode="new" />,
    ...titled('New activity'),
  },
  {
    path: '/plans/:planId/activities/:activityId',
    element: <ActivityPage />,
    ...titled('Activity'),
  },
  {
    path: '/plans/:planId/activities/:activityId/edit',
    element: <ActivityFormPage mode="edit" />,
    ...titled('Edit activity'),
  },
  { path: '/activities/:activityId', element: <ActivityShortLink />, ...titled('Activity') },
  {
    path: '/activities/:activityId/line-items/new',
    element: <LineItemCreatePage />,
    ...titled('New line item'),
  },
  { path: '/line-items/:lineItemId', element: <LineItemPage />, ...titled('Line item') },
  {
    path: '/line-items/:lineItemId/edit',
    element: <LineItemEditPage />,
    ...titled('Edit line item'),
  },
  // 008 Approvals and Settings › Approval workflows (with an id the details panel is open)
  { path: '/approvals', element: <ApprovalsPage />, ...titled('Approvals') },
  {
    path: '/settings/approval-workflows/:workflowId?',
    element: <ApprovalWorkflowsPage />,
    ...titled('Approval workflows'),
  },
  { path: '/all-work', element: <AllWorkPage />, ...titled('All work') },
  {
    path: '/budgets',
    element: gated('globalBudgets.view', <GlobalBudgetsPage />),
    ...titled('Global budgets'),
  },
  {
    path: '/budgets/:globalBudgetId',
    element: gated('globalBudgets.view', <GlobalBudgetPage />),
    ...titled('Global budget'),
  },
  // 015 Financials; a transaction's details panel opens over the list (child route)
  {
    path: '/financials',
    element: <FinancialsPage />,
    ...titled('Financials'),
    children: [
      {
        path: 'transactions/:transactionId',
        element: <TransactionPanel />,
        ...titled('Financials'),
      },
    ],
  },
  // 032 Purchase requests; the list only for people holding purchaseRequests.list
  {
    path: '/purchase-requests',
    element: gated('purchaseRequests.list', <PurchaseRequestsPage />),
    ...titled('Purchase requests'),
  },
  {
    path: '/purchase-requests/new',
    element: <PurchaseRequestCreatePage />,
    ...titled('New purchase request'),
  },
  {
    path: '/purchase-requests/:purchaseRequestId',
    element: <PurchaseRequestPage />,
    ...titled('Purchase request'),
  },
  {
    path: '/purchase-requests/:purchaseRequestId/edit',
    element: <PurchaseRequestEditPage />,
    ...titled('Edit purchase request'),
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
  { path: '/activity-log', element: <ActivityLogPage />, ...titled('Activity log') },
  { path: '/recycle-bin', element: <RecycleBinPage />, ...titled('Recycle bin') },
  {
    path: '/settings/currencies/import',
    element: gated('exchangeRates.manage', <ImportRatesPage />),
    ...titled('Import GSAP rates'),
  },
  { path: '/settings/field-help', element: <FieldHelpSettingsPage />, ...titled('Field help') },
  // 014 Settings › Roles and access, for people whose role includes access.manage (FR-012)
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
  ...DASHBOARD_ROUTES,
];

/** Every page and its address (F1): a data router without loaders; pages load their own data. */
export const routes: RouteObject[] = [
  {
    path: '/',
    element: <RootLayout />,
    errorElement: <RouteErrorPage />,
    children: [
      { path: '/sign-in', element: <SignInPage /> },
      {
        element: (
          <RequireSignIn>
            <AppLayout>
              <Outlet />
            </AppLayout>
          </RequireSignIn>
        ),
        children: [
          { index: true, element: <HomePage />, ...titled('Home') },
          ...PLAN_MODULE_ROUTES,
          ...WORKBOOK_ROUTES,
          ...COMING_SOON_SECTIONS.map((section) => ({
            path: section.path,
            element: <ComingSoonPage section={section} />,
            ...titled(section.label),
          })),
          // Developer pages: only where demo sign-in is on (research R23, D-036).
          {
            path: '/dev/sample-records',
            element: (
              <DevOnly>
                <SampleRecordsPage />
              </DevOnly>
            ),
            ...titled('Sample records'),
          },
          {
            path: '/dev/components',
            element: (
              <DevOnly>
                <ComponentGalleryPage />
              </DevOnly>
            ),
            ...titled('Shared building blocks'),
          },
        ],
      },
      { path: '*', element: <NotFoundRoute /> },
    ],
  },
];

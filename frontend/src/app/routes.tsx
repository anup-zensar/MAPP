import { Outlet } from 'react-router';
import type { RouteObject } from 'react-router';
import type { PageHandle } from '@/session/page-handle';
import { RequirePermission } from '@/session/RequirePermission';
import { RequireSignIn } from '@/session/RequireSignIn';
import { AppLayout } from './AppLayout';
import { DevOnly } from './DevOnly';
import { COMING_SOON_SECTIONS } from './navigation';
import { NotFoundRoute } from './NotFoundRoute';
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
  ComingSoonPage,
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
  PeoplePage,
  PlanCreatePage,
  PlanEditPage,
  PlanPage,
  PlansPage,
  ProfilePage,
  RecycleBinPage,
  RoleFormPage,
  RolePage,
  SampleRecordsPage,
  SettingsPage,
  ShellAttributeEditorPage,
  ShellAttributePage,
  ShellAttributesPage,
  SignInPage,
  TeamsPage,
  TransactionPanel,
} from './pages';
import { RootLayout } from './RootLayout';
import { RouteErrorPage } from './RouteErrorPage';

/** The tab title shown while the session is checked (each page names itself afterwards). */
function titled(title: string): { handle: PageHandle } {
  return { handle: { title } };
}

/** Every page of the Plan module (features 002–016; docs/sdd/plan-module-build.md §5). */
const PLAN_MODULE_ROUTES: RouteObject[] = [
  // 002 Settings
  {
    path: '/settings',
    element: <SettingsPage />,
    ...titled('Settings'),
  },
  { path: '/settings/models', element: <ModelsPage />, ...titled('Models') },
  { path: '/settings/activity-types', element: <ActivityTypesPage />, ...titled('Activity types') },
  {
    path: '/settings/shell-attributes',
    element: <ShellAttributesPage />,
    ...titled('Shell attributes'),
  },
  {
    path: '/settings/shell-attributes/new',
    element: <ShellAttributeEditorPage mode="create" />,
    ...titled('New Shell attribute'),
  },
  {
    path: '/settings/shell-attributes/:attributeId',
    element: <ShellAttributePage />,
    ...titled('Shell attribute'),
  },
  {
    path: '/settings/shell-attributes/:attributeId/edit',
    element: <ShellAttributeEditorPage mode="edit" />,
    ...titled('Edit Shell attribute'),
  },
  { path: '/settings/currencies', element: <CurrenciesPage />, ...titled('Currencies') },
  {
    path: '/settings/finance-budget-items',
    element: <FinanceBudgetItemsPage />,
    ...titled('Finance budget items'),
  },
  // 003 People and teams; with an id the details panel is open
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
  // 007 Line items
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
  // 010 All work
  { path: '/all-work', element: <AllWorkPage />, ...titled('All work') },
  // 014 Global budgets (US6)
  { path: '/budgets', element: <GlobalBudgetsPage />, ...titled('Global budgets') },
  {
    path: '/budgets/:globalBudgetId',
    element: <GlobalBudgetPage />,
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
  // 011 Activity log and Recycle bin (under MORE in the sidebar)
  { path: '/activity-log', element: <ActivityLogPage />, ...titled('Activity log') },
  { path: '/recycle-bin', element: <RecycleBinPage />, ...titled('Recycle bin') },
  // 016 Settings › Import GSAP rates and Field help
  {
    path: '/settings/currencies/import',
    element: <ImportRatesPage />,
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
    element: <RequirePermission permission="access.manage">{route.element}</RequirePermission>,
  })),
];

/**
 * Every page of the web app and its address (plan "Router", constitution F1: routes in one module).
 * A data router without loaders or actions: pages load their own data with TanStack Query, and
 * their code with their feature (app/pages.tsx). Tests build a memory router from these.
 */
export const routes: RouteObject[] = [
  {
    path: '/',
    element: <RootLayout />,
    // Any page that fails to draw shows the error page instead of a blank screen.
    errorElement: <RouteErrorPage />,
    children: [
      { path: '/sign-in', element: <SignInPage /> },
      {
        // Every page in here needs a signed-in person and is drawn inside the app layout. Each
        // names itself in `handle`, for the browser tab while the session is being checked.
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
          ...COMING_SOON_SECTIONS.map((section) => ({
            path: section.path,
            element: <ComingSoonPage section={section} />,
            ...titled(section.label),
          })),
          // The developer pages: not in the navigation, and only where demo sign-in is on — the
          // local and development environments (research R23, D-036).
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
      // Any other address: "Page not found", for signed-in people and visitors alike (FR-015).
      { path: '*', element: <NotFoundRoute /> },
    ],
  },
];

import { createBrowserRouter } from 'react-router';
import type { RouteObject } from 'react-router';
import { ActivityFormPage } from '@/features/activities/ActivityFormPage';
import { ActivityPage } from '@/features/activities/ActivityPage';
import { ActivityShortLink } from '@/features/activities/ActivityShortLink';
import { ActivityLogPage } from '@/features/activity-log/ActivityLogPage';
import { AllWorkPage } from '@/features/all-work/AllWorkPage';
import { ApprovalWorkflowsPage } from '@/features/approval-workflows/ApprovalWorkflowsPage';
import { ApprovalsPage } from '@/features/approvals/ApprovalsPage';
import { CampaignPanel } from '@/features/campaigns/CampaignPanel';
import { InitiativePanel } from '@/features/campaigns/InitiativePanel';
import { NewCampaignPanel } from '@/features/campaigns/NewCampaignPanel';
import { NewInitiativePanel } from '@/features/campaigns/NewInitiativePanel';
import { ComponentGalleryPage } from '@/features/dev/ComponentGalleryPage';
import { FinancialsPage } from '@/features/financials/FinancialsPage';
import { TransactionPanel } from '@/features/financials/TransactionPanel';
import { SampleRecordsPage } from '@/features/dev/SampleRecordsPage';
import { HomePage } from '@/features/home/HomePage';
import { LineItemCreatePage } from '@/features/line-items/LineItemCreatePage';
import { LineItemEditPage } from '@/features/line-items/LineItemEditPage';
import { LineItemPage } from '@/features/line-items/LineItemPage';
import { PeoplePage } from '@/features/people/PeoplePage';
import { ProfilePage } from '@/features/people/ProfilePage';
import { TeamsPage } from '@/features/people/TeamsPage';
import { ComingSoonPage } from '@/features/placeholders/ComingSoonPage';
import { PlanCreatePage } from '@/features/plans/PlanCreatePage';
import { PlanEditPage } from '@/features/plans/PlanEditPage';
import { PlanPage } from '@/features/plans/PlanPage';
import { PlansPage } from '@/features/plans/PlansPage';
import { RecycleBinPage } from '@/features/recycle-bin/RecycleBinPage';
import { ActivityTypesPage } from '@/features/settings/activity-types/ActivityTypesPage';
import { CurrenciesPage } from '@/features/settings/currencies/CurrenciesPage';
import { FinanceBudgetItemsPage } from '@/features/settings/finance-budget-items/FinanceBudgetItemsPage';
import { ModelsPage } from '@/features/settings/models/ModelsPage';
import { SettingsPage } from '@/features/settings/SettingsPage';
import { ShellAttributeEditorPage } from '@/features/settings/shell-attributes/ShellAttributeEditorPage';
import { ShellAttributePage } from '@/features/settings/shell-attributes/ShellAttributePage';
import { ShellAttributesPage } from '@/features/settings/shell-attributes/ShellAttributesPage';
import { SignInPage } from '@/features/sign-in/SignInPage';
import { DevOnly } from './DevOnly';
import { COMING_SOON_SECTIONS } from './navigation';
import { NotFoundRoute } from './NotFoundRoute';
import { RequireSignIn } from './RequireSignIn';
import type { PageHandle } from './RequireSignIn';
import { RootLayout } from './RootLayout';
import { RouteErrorPage } from './RouteErrorPage';

/** The tab title shown while the session is checked (each page names itself afterwards). */
function titled(title: string): { handle: PageHandle } {
  return { handle: { title } };
}

/**
 * Every page of the Plan module (features 002–008; docs/sdd/plan-module-build.md §5). Each
 * element is its feature's page — a stub until the feature replaces the file — so no feature has
 * to change this file.
 */
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
];

/**
 * Every page of the web app and its address (plan "Router"). A data router without loaders or
 * actions: pages load their own data with TanStack Query. Tests build a memory router from these.
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
        element: <RequireSignIn />,
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

/** The router of the running app; the address comes from the browser. */
export const router = createBrowserRouter(routes);

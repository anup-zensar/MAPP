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
  ActivityFormPage,
  ActivityLogPage,
  ActivityPage,
  ActivityShortLink,
  AllWorkPage,
  ApprovalsPage,
  CampaignPanel,
  ComponentGalleryPage,
  FinancialsPage,
  GlobalBudgetPage,
  GlobalBudgetsPage,
  HomePage,
  InitiativePanel,
  LineItemCreatePage,
  LineItemEditPage,
  LineItemPage,
  NewCampaignPanel,
  NewInitiativePanel,
  PlanCreatePage,
  PlanEditPage,
  PlanPage,
  PlansPage,
  ProfilePage,
  PurchaseRequestCreatePage,
  PurchaseRequestEditPage,
  PurchaseRequestPage,
  PurchaseRequestsPage,
  RecycleBinPage,
  SampleRecordsPage,
  SignInPage,
  SpaPage,
  TransactionPanel,
} from './pages';
import { RootLayout } from './RootLayout';
import { gated, titled } from './route-helpers';
import { RouteErrorPage } from './RouteErrorPage';
import { SETTINGS_ROUTES } from './settings-routes';

/** Every page of the Plan module (features 002–016; docs/sdd/plan-module-build.md §5). */
const PLAN_MODULE_ROUTES: RouteObject[] = [
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
  // 008 Approvals
  { path: '/approvals', element: <ApprovalsPage />, ...titled('Approvals') },
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
  { path: '/activity-log', element: <ActivityLogPage />, ...titled('Activity log') },
  { path: '/recycle-bin', element: <RecycleBinPage />, ...titled('Recycle bin') },
  ...DASHBOARD_ROUTES,
  ...SETTINGS_ROUTES,
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

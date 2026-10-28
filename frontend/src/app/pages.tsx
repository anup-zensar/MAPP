import { lazy } from 'react';
import { loadSettingsSections } from './settings-sections';

// Each page comes with its feature's code, loaded the first time one of its pages is shown (F8).
const activities = () => import('@/features/activities');
const activityLog = () => import('@/features/activity-log');
const allWork = () => import('@/features/all-work');
const approvalWorkflows = () => import('@/features/approval-workflows');
const approvals = () => import('@/features/approvals');
const dashboards = () => import('@/features/dashboards');
const dev = () => import('@/features/dev');
const financials = () => import('@/features/financials');
const globalBudgets = () => import('@/features/global-budgets');
const home = () => import('@/features/home');
const lineItems = () => import('@/features/line-items');
const people = () => import('@/features/people');
const plans = () => import('@/features/plans');
const purchaseRequests = () => import('@/features/purchase-requests');
const recycleBin = () => import('@/features/recycle-bin');
const settings = () => import('@/features/settings');
const signIn = () => import('@/features/sign-in');

export const ActivityFormPage = lazy(() =>
  activities().then((feature) => ({ default: feature.ActivityFormPage })),
);
export const ActivityPage = lazy(() =>
  activities().then((feature) => ({ default: feature.ActivityPage })),
);
export const ActivityShortLink = lazy(() =>
  activities().then((feature) => ({ default: feature.ActivityShortLink })),
);
export const ActivityLogPage = lazy(() =>
  activityLog().then((feature) => ({ default: feature.ActivityLogPage })),
);
export const AllWorkPage = lazy(() =>
  allWork().then((feature) => ({ default: feature.AllWorkPage })),
);
export const ApprovalWorkflowsPage = lazy(() =>
  approvalWorkflows().then((feature) => ({ default: feature.ApprovalWorkflowsPage })),
);
export const ApprovalsPage = lazy(() =>
  approvals().then((feature) => ({ default: feature.ApprovalsPage })),
);
export const DashboardsPage = lazy(() =>
  dashboards().then((feature) => ({ default: feature.DashboardsPage })),
);
export const ComponentGalleryPage = lazy(() =>
  dev().then((feature) => ({ default: feature.ComponentGalleryPage })),
);
export const SampleRecordsPage = lazy(() =>
  dev().then((feature) => ({ default: feature.SampleRecordsPage })),
);
export const FinancialsPage = lazy(() =>
  financials().then((feature) => ({ default: feature.FinancialsPage })),
);
export const TransactionPanel = lazy(() =>
  financials().then((feature) => ({ default: feature.TransactionPanel })),
);
export const GlobalBudgetPage = lazy(() =>
  globalBudgets().then((feature) => ({ default: feature.GlobalBudgetPage })),
);
export const GlobalBudgetsPage = lazy(() =>
  globalBudgets().then((feature) => ({ default: feature.GlobalBudgetsPage })),
);
export const HomePage = lazy(() => home().then((feature) => ({ default: feature.HomePage })));
export const LineItemCreatePage = lazy(() =>
  lineItems().then((feature) => ({ default: feature.LineItemCreatePage })),
);
export const LineItemEditPage = lazy(() =>
  lineItems().then((feature) => ({ default: feature.LineItemEditPage })),
);
export const LineItemPage = lazy(() =>
  lineItems().then((feature) => ({ default: feature.LineItemPage })),
);
export const AccessPage = lazy(() => people().then((feature) => ({ default: feature.AccessPage })));
export const PeoplePage = lazy(() => people().then((feature) => ({ default: feature.PeoplePage })));
export const ProfilePage = lazy(() =>
  people().then((feature) => ({ default: feature.ProfilePage })),
);
export const RoleFormPage = lazy(() =>
  people().then((feature) => ({ default: feature.RoleFormPage })),
);
export const RolePage = lazy(() => people().then((feature) => ({ default: feature.RolePage })));
export const TeamsPage = lazy(() => people().then((feature) => ({ default: feature.TeamsPage })));
export const CampaignPanel = lazy(() =>
  plans().then((feature) => ({ default: feature.CampaignPanel })),
);
export const InitiativePanel = lazy(() =>
  plans().then((feature) => ({ default: feature.InitiativePanel })),
);
export const NewCampaignPanel = lazy(() =>
  plans().then((feature) => ({ default: feature.NewCampaignPanel })),
);
export const NewInitiativePanel = lazy(() =>
  plans().then((feature) => ({ default: feature.NewInitiativePanel })),
);
export const PlanCreatePage = lazy(() =>
  plans().then((feature) => ({ default: feature.PlanCreatePage })),
);
export const PlanEditPage = lazy(() =>
  plans().then((feature) => ({ default: feature.PlanEditPage })),
);
export const PlanPage = lazy(() => plans().then((feature) => ({ default: feature.PlanPage })));
export const PlansPage = lazy(() => plans().then((feature) => ({ default: feature.PlansPage })));
export const RecycleBinPage = lazy(() =>
  recycleBin().then((feature) => ({ default: feature.RecycleBinPage })),
);
export const ActivityTypesPage = lazy(() =>
  settings().then((feature) => ({ default: feature.ActivityTypesPage })),
);
export const CurrenciesPage = lazy(() =>
  settings().then((feature) => ({ default: feature.CurrenciesPage })),
);
export const FieldHelpSettingsPage = lazy(() =>
  settings().then((feature) => ({ default: feature.FieldHelpSettingsPage })),
);
export const FinanceBudgetItemsPage = lazy(() =>
  settings().then((feature) => ({ default: feature.FinanceBudgetItemsPage })),
);
export const ImportRatesPage = lazy(() =>
  settings().then((feature) => ({ default: feature.ImportRatesPage })),
);
export const ModelsPage = lazy(() =>
  settings().then((feature) => ({ default: feature.ModelsPage })),
);
export const ShellAttributeEditorPage = lazy(() =>
  settings().then((feature) => ({ default: feature.ShellAttributeEditorPage })),
);
export const ShellAttributePage = lazy(() =>
  settings().then((feature) => ({ default: feature.ShellAttributePage })),
);
export const ShellAttributesPage = lazy(() =>
  settings().then((feature) => ({ default: feature.ShellAttributesPage })),
);
export const SignInPage = lazy(() => signIn().then((feature) => ({ default: feature.SignInPage })));

/** The Settings page, with the cards of the features that add one (app/settings-sections.ts). */
export const SettingsPage = lazy(async () => {
  const [feature, sections] = await Promise.all([settings(), loadSettingsSections()]);
  return { default: () => <feature.SettingsPage sections={sections} /> };
});

export const PurchaseRequestsPage = lazy(() =>
  purchaseRequests().then((feature) => ({ default: feature.PurchaseRequestsPage })),
);
export const PurchaseRequestPage = lazy(() =>
  purchaseRequests().then((feature) => ({ default: feature.PurchaseRequestPage })),
);
export const PurchaseRequestCreatePage = lazy(() =>
  purchaseRequests().then((feature) => ({ default: feature.PurchaseRequestCreatePage })),
);
export const PurchaseRequestEditPage = lazy(() =>
  purchaseRequests().then((feature) => ({ default: feature.PurchaseRequestEditPage })),
);
export const PurchaseRequestSettingsPage = lazy(() =>
  purchaseRequests().then((feature) => ({ default: feature.PurchaseRequestSettingsPage })),
);

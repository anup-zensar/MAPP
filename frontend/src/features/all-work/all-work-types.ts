/** The contract's All work shapes under short names (backend/api/openapi.yaml, feature 010). */
import type { components, operations } from '@/api/generated/schema';

type Schemas = components['schemas'];

export type AllWorkResult = Schemas['AllWorkResult'];
export type AllWorkRow = Schemas['AllWorkRow'];
export type AllWorkCell = Schemas['AllWorkCell'];
export type AllWorkColumn = Schemas['AllWorkColumn'];
export type AllWorkColumnGroup = Schemas['AllWorkColumnGroup'];
export type AllWorkStatus = Schemas['AllWorkStatus'];
export type AllWorkSummary = Schemas['AllWorkSummary'];
export type AllWorkOptions = Schemas['AllWorkOptions'];
export type AllWorkOption = Schemas['AllWorkOption'];
export type AllWorkPreferences = Schemas['AllWorkPreferences'];
export type AllWorkRecordLocation = Schemas['AllWorkRecordLocation'];
export type Money = Schemas['Money'];
export type SavedView = Schemas['SavedView'];
export type SavedViewCreate = Schemas['SavedViewCreate'];
export type SavedViewPage = Schemas['SavedViewPage'];
export type SavedViewUpdate = Schemas['SavedViewUpdate'];

export type PlanType = Schemas['PlanType'];

export type FinanceFilter = NonNullable<
  NonNullable<operations['getAllWork']['parameters']['query']>['finance']
>[number];

// The Financials filter lives in the session and the address only (see savedStateOf).
export type Filters = Schemas['AllWorkFilters'] & { finance?: FinanceFilter[] };

export type ViewState = Omit<Schemas['AllWorkViewState'], 'filters'> & { filters: Filters };
export type FilterValue = Schemas['AllWorkFilterValue'];
export type Sort = Schemas['AllWorkSort'];
export type ViewType = ViewState['view'];

export type AllWorkQuery = NonNullable<operations['getAllWork']['parameters']['query']>;

export type AttributeFilter = Schemas['AllWorkAttributeFilter'];
export type FilterableAttribute = Schemas['AllWorkFilterableAttribute'];
export type BreakdownChoice = Schemas['AllWorkBreakdownChoice'];
export type AllWorkBreakdown = Schemas['AllWorkBreakdown'];
export type BreakdownGroup = Schemas['AllWorkBreakdownGroup'];
export type BreakdownSubgroup = Schemas['AllWorkBreakdownSubgroup'];
export type Grouping = Schemas['AllWorkGrouping'];

export type BreakdownQuery = NonNullable<operations['getAllWorkBreakdown']['parameters']['query']>;

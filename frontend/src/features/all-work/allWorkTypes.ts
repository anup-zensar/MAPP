/** The contract's All work shapes under short names (backend/api/openapi.yaml, feature 010). */
import type { components, operations } from '@/api/schema';

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

/** Everything one All work view is (data-model §1.2): kept in the session and in saved views. */
export type ViewState = Schemas['AllWorkViewState'];
export type Filters = Schemas['AllWorkFilters'];
export type FilterValue = Schemas['AllWorkFilterValue'];
export type Sort = Schemas['AllWorkSort'];
export type ViewType = ViewState['view'];

/** The query of GET /api/all-work. */
export type AllWorkQuery = NonNullable<operations['getAllWork']['parameters']['query']>;

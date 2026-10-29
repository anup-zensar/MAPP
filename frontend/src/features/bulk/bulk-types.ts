import type { components } from '@/api/generated/schema';

type Schemas = components['schemas'];

export type BulkRequest = Omit<Schemas['BulkActionRequest'], 'restartApproval'> & {
  restartApproval?: boolean;
};
export type BulkResult = Schemas['BulkActionResult'];
export type BulkKind = Schemas['BulkActionKind'];
export type BulkLevel = Schemas['BulkLevel'];
export type BulkField = Schemas['BulkField'];
export type BulkFieldValue = Schemas['BulkFieldValue'];
export type BulkTarget = Schemas['BulkTarget'];
export type BulkTargetPage = Schemas['BulkTargetPage'];
export type BulkOutcome = Schemas['BulkOutcome'];

export interface SelectedRecord {
  id: string;
  level: BulkLevel;
  name: string;
  planId: string;
}

export interface LevelCounts {
  campaigns: number;
  initiatives: number;
  activities: number;
}

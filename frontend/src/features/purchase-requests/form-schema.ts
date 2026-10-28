import { z } from 'zod';
import type { components } from '@/api/generated/schema';
import { parseMoneyInput, toCents } from '@/format/decimal';
import type { PurchaseRequestLineItemRef } from './api';
import { PROBLEMS } from './form-text';

type AttributeValues = components['schemas']['AttributeValues'];

const NAME_MAX = 120;
const DESCRIPTION_MAX = 2000;

function centsOf(text: string): bigint | null {
  const parsed = parseMoneyInput(text);
  return parsed ? toCents(parsed.amount) : null;
}

const amountText = z.string().refine((text) => (centsOf(text) ?? 0n) > 0n, PROBLEMS.amount);
const limitText = z
  .string()
  .refine((text) => text.trim() === '' || (centsOf(text) ?? -1n) >= 0n, PROBLEMS.overallLimit);

export const lineSchema = z.object({
  id: z.string(),
  lineItem: z.custom<PurchaseRequestLineItemRef>(),
  hxKey: z.string(),
  currency: z.string().min(1, PROBLEMS.currency),
  amount: amountText,
  deliveryDate: z.string().min(1, PROBLEMS.deliveryDate),
  overallLimit: limitText,
  description: z.string().max(DESCRIPTION_MAX, PROBLEMS.descriptionLength),
  attributes: z.custom<AttributeValues>(),
});

/** Fast feedback only (032 FR-053): every other rule is the server's, answered all at once. */
export const purchaseRequestFormSchema = z.object({
  name: z.string().trim().min(1, PROBLEMS.name).max(NAME_MAX, PROBLEMS.nameLength),
  description: z.string().max(DESCRIPTION_MAX, PROBLEMS.descriptionLength),
  ownerId: z.string().min(1, PROBLEMS.owner),
  moaApproverId: z.string(),
  approvalWorkflowId: z.string().min(1, PROBLEMS.approvalWorkflow),
  modificationWorkflowId: z.string().min(1, PROBLEMS.modificationWorkflow),
  attributes: z.custom<AttributeValues>(),
  lines: z.array(lineSchema).min(1, PROBLEMS.lines),
});

export type PurchaseRequestFormValues = z.infer<typeof purchaseRequestFormSchema>;
export type LineValues = z.infer<typeof lineSchema>;

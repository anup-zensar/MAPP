import { useAttributeForm, useCurrencies } from '@/features/settings';
import type { ChoiceOption } from '@/ui/ChoiceField/ChoiceField';
import { useWorkflowOptions } from './api';
import type { PurchaseRequest } from './api';
import type { PurchaseRequestFormOptions } from './PurchaseRequestForm';

interface Named {
  id: string;
  name: string;
  active: boolean;
}

/** The active choices, and the stored one kept (read "(inactive)") when it is no longer active. */
function choices(list: Named[], kept: Named | null | undefined): ChoiceOption[] {
  const options = list.map((each) => ({ id: each.id, label: each.name, active: true }));
  return kept && !options.some((option) => option.id === kept.id)
    ? [...options, { id: kept.id, label: kept.name, active: false }]
    : options;
}

export interface FormOptionsState {
  options: PurchaseRequestFormOptions | null;
  /** The first query that failed, for the page's error state. */
  failed: { error: unknown; refetch: () => unknown } | null;
}

/** Everything the form chooses from: workflows by kind, currencies and both attribute levels. */
export function useFormOptions(stored: PurchaseRequest | null | undefined): FormOptionsState {
  const attributeForm = useAttributeForm('purchaseRequest');
  const lineAttributeForm = useAttributeForm('purchaseRequestLine');
  const workflows = useWorkflowOptions('purchaseRequest');
  const modificationWorkflows = useWorkflowOptions('modifiedPurchaseRequest');
  const currencies = useCurrencies();
  const queries = [attributeForm, lineAttributeForm, workflows, modificationWorkflows, currencies];
  const failed = queries.find((query) => query.isError);
  if (failed) {
    return { options: null, failed: { error: failed.error, refetch: failed.refetch } };
  }
  if (
    !attributeForm.data ||
    !lineAttributeForm.data ||
    !workflows.data ||
    !modificationWorkflows.data ||
    !currencies.data
  ) {
    return { options: null, failed: null };
  }
  const storedCodes = new Set(stored?.lines.map((line) => line.currency) ?? []);
  return {
    failed: null,
    options: {
      attributeForm: attributeForm.data,
      lineAttributeForm: lineAttributeForm.data,
      workflows: choices(workflows.data, stored?.approvalWorkflow),
      modificationWorkflows: choices(modificationWorkflows.data, stored?.modificationWorkflow),
      currencies: currencies.data.items
        .filter((currency) => currency.active || storedCodes.has(currency.code))
        .map((currency) => ({
          id: currency.code,
          label: `${currency.code} — ${currency.name}`,
          active: currency.active,
        })),
    },
  };
}

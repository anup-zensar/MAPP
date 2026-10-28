import { useNavigate, useSearchParams } from 'react-router';
import { Card } from '@/ui/Card/Card';
import { PageHeader } from '@/ui/PageHeader/PageHeader';
import { purchaseRequestHref, recordHref } from '@/ui/record-links';
import { useToast } from '@/ui/Toast/useToast';
import { usePageTitle } from '@/ui/usePageTitle';
import { useCreatePurchaseRequest, usePurchaseRequestDefaults } from './api';
import type { PurchaseRequestDefaults } from './api';
import { fromDefaults } from './form-state';
import { createdToast, PICKER } from './form-text';
import { FormPageFailed, FormPageLoading } from './FormPageStates';
import { LineItemOptionsList } from './LineItemOptionsList';
import { NEW_PURCHASE_REQUEST } from './purchase-request-text';
import { PurchaseRequestForm } from './PurchaseRequestForm';
import type { PurchaseRequestFormOptions } from './PurchaseRequestForm';
import { useFormOptions } from './useFormOptions';
import styles from './PurchaseRequestForm.module.css';

/**
 * New purchase request (032 US1, FR-001 – FR-003) at /purchase-requests/new?lineItemId=…: the form
 * pre-filled from the line item; without a line item, choosing one comes first.
 */
export function PurchaseRequestCreatePage() {
  const [params, setParams] = useSearchParams();
  const lineItemId = params.get('lineItemId') ?? '';
  usePageTitle(NEW_PURCHASE_REQUEST);

  if (!lineItemId) {
    return (
      <div className={styles.step}>
        <PageHeader
          breadcrumb={[{ label: 'Plans', to: '/plans' }, { label: NEW_PURCHASE_REQUEST }]}
          title={NEW_PURCHASE_REQUEST}
          description={PICKER.stepDescription}
        />
        <Card title={PICKER.stepTitle}>
          <LineItemOptionsList
            query={{}}
            action={PICKER.choose}
            onChoose={(option) => setParams({ lineItemId: option.lineItem.id })}
          />
        </Card>
      </div>
    );
  }
  return <CreateForm key={lineItemId} lineItemId={lineItemId} />;
}

function CreateForm({ lineItemId }: { lineItemId: string }) {
  const defaults = usePurchaseRequestDefaults(lineItemId);
  const { options, failed } = useFormOptions(null);
  const loadFailed = defaults.isError ? defaults : failed;

  if (loadFailed) {
    return (
      <FormPageFailed
        title={NEW_PURCHASE_REQUEST}
        error={loadFailed.error}
        onRetry={() => void loadFailed.refetch()}
      />
    );
  }
  if (!defaults.data || !options) {
    return <FormPageLoading title={NEW_PURCHASE_REQUEST} />;
  }
  return <Loaded lineItemId={lineItemId} defaults={defaults.data} options={options} />;
}

interface LoadedProps {
  lineItemId: string;
  defaults: PurchaseRequestDefaults;
  options: PurchaseRequestFormOptions;
}

function Loaded({ lineItemId, defaults, options }: LoadedProps) {
  const navigate = useNavigate();
  const { show } = useToast();
  const create = useCreatePurchaseRequest();
  const { lineItem } = defaults.line;

  return (
    <PurchaseRequestForm
      title={NEW_PURCHASE_REQUEST}
      breadcrumb={[
        { label: 'Plans', to: '/plans' },
        { label: defaults.plan.name, to: recordHref('plan', defaults.plan.id) },
        { label: lineItem.name, to: recordHref('lineItem', lineItem.id) },
        { label: NEW_PURCHASE_REQUEST },
      ]}
      initial={fromDefaults(defaults)}
      stored={null}
      options={options}
      optionsQuery={{
        planId: defaults.plan.id,
        ...(defaults.companyCode ? { companyCode: defaults.companyCode } : {}),
      }}
      onSave={(input) => create.mutateAsync(input)}
      onSaved={(created) => {
        show({ message: createdToast(created.name) });
        void navigate(purchaseRequestHref(created.id));
      }}
      onCancel={() => void navigate(recordHref('lineItem', lineItemId))}
    />
  );
}

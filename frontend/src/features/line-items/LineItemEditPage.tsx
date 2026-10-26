import { useState } from 'react';
import { useNavigate, useParams, useSearchParams } from 'react-router';
import { useAttributeForm } from '@/features/settings';
import type { AttributeForm } from '@/ui/ShellAttributeFields/rules';
import { useToast } from '@/ui/Toast/useToast';
import { recordHref } from '@/ui/record-links';
import { usePageTitle } from '@/ui/usePageTitle';
import { LineItemForm } from './LineItemForm';
import { editFormState } from './line-item-form-state';
import { LineItemLoadFailed, LineItemLoading } from './LineItemPageStates';
import {
  useFetchLatestLineItem,
  useLineItem,
  useLineItemFormOptions,
  useUpdateLineItem,
} from './api';
import type { LineItemDetails, LineItemFormOptions } from './api';
import { EDIT_TITLE, savedToast } from './line-item-text';

const LOADING_TITLE = 'Edit line item';

/**
 * Edit a line item (US4, FR-027), at /line-items/:lineItemId/edit (?field=<field> focuses a field):
 * the form with the stored values; only a changed planned cost or currency of a submitted or
 * approved activity asks to restart its approval (FR-031, FR-032); a conflict keeps the values.
 */
export function LineItemEditPage() {
  const { lineItemId = '' } = useParams();
  const [search] = useSearchParams();
  const details = useLineItem(lineItemId);
  const activityId = details.data?.activity.id ?? '';
  const options = useLineItemFormOptions(activityId, Boolean(activityId));
  const attributeForm = useAttributeForm('lineItem');
  usePageTitle(details.data ? `Edit ${details.data.lineItem.name}` : LOADING_TITLE);

  const failed = [details, options, attributeForm].find((query) => query.isError);
  if (failed) {
    return (
      <LineItemLoadFailed
        title={LOADING_TITLE}
        error={failed.error}
        onRetry={() => void failed.refetch()}
      />
    );
  }
  if (!details.data || !options.data || !attributeForm.data) {
    return <LineItemLoading title={LOADING_TITLE} />;
  }
  return (
    <EditForm
      details={details.data}
      options={options.data}
      attributeForm={attributeForm.data}
      focusField={search.get('field')}
    />
  );
}

interface EditFormProps {
  details: LineItemDetails;
  options: LineItemFormOptions;
  attributeForm: AttributeForm;
  focusField: string | null;
}

/** The form once everything is loaded: the line item and its version are copied once. */
function EditForm({ details, options, attributeForm, focusField }: EditFormProps) {
  const { lineItem } = details;
  const navigate = useNavigate();
  const fetchLatest = useFetchLatestLineItem();
  const { show } = useToast();
  const update = useUpdateLineItem(lineItem.id);
  const [initial] = useState(() => editFormState(lineItem));
  // The edit-form rule: only a conflict replaces the version the person started from.
  const [version, setVersion] = useState(lineItem.version);
  const href = recordHref('lineItem', lineItem.id);

  return (
    <LineItemForm
      mode="edit"
      options={options}
      attributeForm={attributeForm}
      // The plan's Business unit options decide which conditional attributes show (D-212).
      businessUnit={details.activity.businessUnitOptionIds}
      initial={initial}
      stored={lineItem}
      title={EDIT_TITLE}
      breadcrumbEnd={lineItem.name}
      focusField={focusField}
      onSave={async (body) => {
        const saved = await update.mutateAsync({ ...body, version });
        show({ message: savedToast(saved.name) });
        return () => navigate(href);
      }}
      onConflict={async () => {
        const latest = await fetchLatest(lineItem.id);
        setVersion(latest.lineItem.version);
      }}
      onCancel={() => navigate(href)}
    />
  );
}

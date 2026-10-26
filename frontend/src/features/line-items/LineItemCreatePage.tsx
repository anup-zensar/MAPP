import { useState } from 'react';
import { useNavigate, useParams } from 'react-router';
import { useAttributeForm } from '@/features/settings/api';
import { useToast } from '@/ui/Toast/useToast';
import { usePageTitle } from '@/ui/usePageTitle';
import { LineItemForm } from './LineItemForm';
import { newFormState } from './lineItemFormState';
import { activityLineItemsHref } from './lineItemLinks';
import { LineItemLoadFailed, LineItemLoading } from './LineItemPageStates';
import { useCreateLineItem, useLineItemFormOptions } from './lineItemsApi';
import { createdToast, NEW_TITLE } from './lineItemText';

/**
 * New line item (US2, FR-006 – FR-010), at /activities/:activityId/line-items/new: the form filled
 * in with the defaults. Save returns to the activity with the new row highlighted; Save and add
 * another opens a new form whose planned cost is the new unassigned balance.
 */
export function LineItemCreatePage() {
  const { activityId = '' } = useParams();
  usePageTitle(NEW_TITLE);
  const navigate = useNavigate();
  const { show } = useToast();
  const options = useLineItemFormOptions(activityId);
  const attributeForm = useAttributeForm('lineItem');
  const create = useCreateLineItem(activityId);
  // Each Save and add another starts a new form.
  const [round, setRound] = useState(0);

  if (options.isError || attributeForm.isError) {
    const failed = options.isError ? options : attributeForm;
    return (
      <LineItemLoadFailed
        title={NEW_TITLE}
        error={failed.error}
        onRetry={() => void failed.refetch()}
      />
    );
  }
  if (!options.data || !attributeForm.data) {
    return <LineItemLoading title={NEW_TITLE} />;
  }

  const { activity } = options.data;
  const back = () => navigate(activityLineItemsHref(activity));

  return (
    <LineItemForm
      key={round}
      mode="create"
      options={options.data}
      attributeForm={attributeForm.data}
      // The plan's Business unit options decide which conditional attributes show (D-212).
      businessUnit={activity.businessUnitOptionIds}
      initial={newFormState(options.data)}
      title={NEW_TITLE}
      breadcrumbEnd={NEW_TITLE}
      focusName={round > 0}
      onSave={async (body, addAnother) => {
        const created = await create.mutateAsync(body);
        show({ message: createdToast(created.name) });
        if (addAnother) {
          // The form options were refreshed with the change: the new form has the new balance.
          return () => setRound((current) => current + 1);
        }
        return () =>
          navigate(activityLineItemsHref(activity), {
            state: { highlightLineItemId: created.id },
          });
      }}
      onCancel={back}
    />
  );
}

import { zodResolver } from '@hookform/resolvers/zod';
import { act, renderHook } from '@testing-library/react';
import { useForm } from 'react-hook-form';
import { describe, expect, it } from 'vitest';
import { z } from 'zod';
import { CHECK_ON_SAVE, useFormProblems } from './useFormProblems';

const schema = z
  .object({ name: z.string(), startDate: z.string(), endDate: z.string() })
  .superRefine((values, context) => {
    if (values.name === '') {
      context.addIssue({ code: 'custom', path: ['name'], message: 'Enter a name.' });
    }
    if (values.endDate < values.startDate) {
      context.addIssue({ code: 'custom', path: ['endDate'], message: 'End after the start.' });
    }
    if (values.endDate > '2030-12-31') {
      context.addIssue({ code: 'custom', path: ['endDate'], message: 'End by 2030.' });
    }
  });
type Values = z.infer<typeof schema>;
const FIELDS = ['name', 'startDate', 'endDate'] as const;

function renderForm(defaultValues: Values) {
  return renderHook(() => {
    const form = useForm<Values>({
      resolver: zodResolver(schema),
      defaultValues,
      ...CHECK_ON_SAVE,
    });
    return { form, ...useFormProblems(form, FIELDS) };
  });
}

describe('useFormProblems', () => {
  it('026 FR-006: lists every problem of the last Save in the form’s order, two for one field included', async () => {
    const { result } = renderForm({ name: '', startDate: '2031-05-01', endDate: '2031-01-01' });

    await act(() => result.current.form.handleSubmit(() => {}, result.current.onInvalid)());

    expect(result.current.problems).toEqual([
      { field: 'name', message: 'Enter a name.' },
      { field: 'endDate', message: 'End after the start.' },
      { field: 'endDate', message: 'End by 2030.' },
    ]);
    expect(result.current.form.getFieldState('endDate').error?.message).toBe(
      'End after the start.',
    );
  });

  it('026 FR-006: puts the API’s field problems at their fields and in the summary', () => {
    const { result } = renderForm({
      name: 'Launch',
      startDate: '2026-01-01',
      endDate: '2026-02-01',
    });

    act(() => result.current.showServerProblems([{ field: 'name', message: 'Name taken.' }]));

    expect(result.current.problems).toEqual([{ field: 'name', message: 'Name taken.' }]);
    expect(result.current.form.getFieldState('name').error?.message).toBe('Name taken.');
  });
});

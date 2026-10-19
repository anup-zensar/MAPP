/**
 * Feature 008's card on the Settings page (listed by features/settings/sections.ts), with the
 * count of active workflows from feature 008's own data.
 */
import { Workflow } from 'lucide-react';
import type { SettingsSection } from '@/features/settings/sections';
import { ActiveWorkflowsCount } from './ActiveWorkflowsCount';

export const APPROVAL_WORKFLOW_SETTINGS_SECTIONS: SettingsSection[] = [
  {
    title: 'Approval workflows',
    description: 'Who approves activities, level by level.',
    path: '/settings/approval-workflows',
    icon: Workflow,
    ActiveCount: ActiveWorkflowsCount,
  },
];

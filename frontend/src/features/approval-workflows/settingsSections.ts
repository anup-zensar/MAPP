/**
 * Feature 008's card on the Settings page (listed by features/settings/sections.ts). Feature 008
 * adds an `ActiveCount` component that shows "<n> active" from its own data.
 */
import { Workflow } from 'lucide-react';
import type { SettingsSection } from '@/features/settings/sections';

export const APPROVAL_WORKFLOW_SETTINGS_SECTIONS: SettingsSection[] = [
  {
    title: 'Approval workflows',
    description: 'Who approves activities, level by level.',
    path: '/settings/approval-workflows',
    icon: Workflow,
  },
];

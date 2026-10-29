import { ShieldAlert } from 'lucide-react';
import type { SettingsSection } from '@/features/settings';
import { OVERSPEND_CONTROL, SETTINGS_DESCRIPTION } from './overspend-text';

export const OVERSPEND_SETTINGS_SECTIONS: SettingsSection[] = [
  {
    title: OVERSPEND_CONTROL,
    description: SETTINGS_DESCRIPTION,
    path: '/settings/overspend-control',
    icon: ShieldAlert,
  },
];

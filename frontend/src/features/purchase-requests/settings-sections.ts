import { ShoppingCart } from 'lucide-react';
import type { SettingsSection } from '@/features/settings';

/** 032's card on the Settings page, after Approval workflows (FR-004). */
export const PURCHASE_REQUEST_SETTINGS_SECTIONS: SettingsSection[] = [
  {
    title: 'Purchase requests',
    description: 'Purchase Support and whether lines may be added to approved purchase requests.',
    path: '/settings/purchase-requests',
    icon: ShoppingCart,
  },
];

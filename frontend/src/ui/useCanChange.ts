import { useAccess } from '@/session/access';
import type { Permission } from '@/session/access';

/**
 * Whether the signed-in person's roles let them change a Settings list (feature 014, FR-011):
 * settings.manage for 002's lists and Field help, exchangeRates.manage for Currencies,
 * people.manage for People and Teams, workflows.manage for Approval workflows. Everything counts as
 * allowed while the API sends no access (session/access).
 */
export function useCanChange(permission: Permission = 'settings.manage'): boolean {
  return useAccess().has(permission);
}

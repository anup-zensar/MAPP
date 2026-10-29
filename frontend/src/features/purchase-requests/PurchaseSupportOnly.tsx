import type { ReactNode } from 'react';
import { usePurchaseRequestConfiguration } from './api';

/** Draws its children unless Purchase Support is known to be off (032 FR-004). */
export function PurchaseSupportOnly({ children }: { children: ReactNode }) {
  const configuration = usePurchaseRequestConfiguration();
  return configuration.data?.purchaseSupport === false ? null : children;
}

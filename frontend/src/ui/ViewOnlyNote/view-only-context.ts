import { createContext, useContext } from 'react';

export const HidesUnavailable = createContext(false);

export function useHidesUnavailable(): boolean {
  return useContext(HidesUnavailable);
}

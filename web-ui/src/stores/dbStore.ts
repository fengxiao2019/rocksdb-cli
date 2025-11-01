import { create } from 'zustand';

interface DbState {
  connected: boolean;
  currentCF: string | null;
  columnFamilies: string[];

  setConnected: (connected: boolean) => void;
  setCurrentCF: (cf: string | null) => void;
  setColumnFamilies: (cfs: string[]) => void;
}

export const useDbStore = create<DbState>((set) => ({
  connected: false,
  currentCF: null,
  columnFamilies: [],

  setConnected: (connected) => set({ connected }),
  setCurrentCF: (cf) => set({ currentCF: cf }),
  setColumnFamilies: (cfs) => set({ columnFamilies: cfs }),
}));

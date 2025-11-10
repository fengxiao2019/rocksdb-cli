import { create } from 'zustand';
import type { DatabaseConnectionInfo, AvailableDatabase } from '../types/api';

interface DbState {
  // Connection state
  connected: boolean;
  currentDatabase: DatabaseConnectionInfo | null;
  availableDatabases: AvailableDatabase[];

  // Column family state
  currentCF: string | null;
  columnFamilies: string[];

  // Actions
  setConnected: (connected: boolean) => void;
  setCurrentDatabase: (db: DatabaseConnectionInfo | null) => void;
  setAvailableDatabases: (databases: AvailableDatabase[]) => void;
  setCurrentCF: (cf: string | null) => void;
  setColumnFamilies: (cfs: string[]) => void;

  // Helper to disconnect and clear state
  disconnect: () => void;
}

export const useDbStore = create<DbState>((set) => ({
  // Initial state
  connected: false,
  currentDatabase: null,
  availableDatabases: [],
  currentCF: null,
  columnFamilies: [],

  // Actions
  setConnected: (connected) => set({ connected }),

  setCurrentDatabase: (db) => set({
    currentDatabase: db,
    connected: db !== null,
    columnFamilies: db?.column_families || [],
  }),

  setAvailableDatabases: (databases) => set({ availableDatabases: databases }),

  setCurrentCF: (cf) => set({ currentCF: cf }),

  setColumnFamilies: (cfs) => set({ columnFamilies: cfs }),

  disconnect: () => set({
    connected: false,
    currentDatabase: null,
    currentCF: null,
    columnFamilies: [],
  }),
}));

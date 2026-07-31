import { create } from "zustand";

import { taroStorage, type KeyValueStorage } from "./session-store";

const ACTIVE_ORDER_KEY = "spl.activeOrderId";

export interface ActiveOrderState {
  activeOrderId: string | undefined;
  hydrate: () => void;
  setActiveOrder: (orderId: string) => void;
  clearActiveOrder: () => void;
}

export interface ActiveOrderStoreDeps {
  storage: KeyValueStorage;
}

export function createActiveOrderStore({ storage }: ActiveOrderStoreDeps) {
  return create<ActiveOrderState>((set) => ({
    activeOrderId: undefined,
    hydrate: () => {
      const existing = storage.getItem(ACTIVE_ORDER_KEY);
      if (existing) {
        set({ activeOrderId: existing });
      }
    },
    setActiveOrder: (orderId) => {
      storage.setItem(ACTIVE_ORDER_KEY, orderId);
      set({ activeOrderId: orderId });
    },
    clearActiveOrder: () => {
      storage.removeItem(ACTIVE_ORDER_KEY);
      set({ activeOrderId: undefined });
    }
  }));
}

export const useActiveOrderStore = createActiveOrderStore({
  storage: taroStorage
});

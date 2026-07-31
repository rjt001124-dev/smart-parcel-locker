import Taro from "@tarojs/taro";
import { create } from "zustand";

const USER_ID_KEY = "spl.session.userId";

// Milestone simplification: there is no login system yet, every client is the
// fixed development user. The backend uses the same constant.
const DEV_USER_ID = "dev-user";

export interface KeyValueStorage {
  getItem: (key: string) => string | undefined;
  setItem: (key: string, value: string) => void;
  removeItem: (key: string) => void;
}

export interface SessionState {
  userId: string | undefined;
  hydrate: () => void;
  clearSession: () => void;
}

export interface SessionStoreDeps {
  storage: KeyValueStorage;
}

export function createSessionStore({ storage }: SessionStoreDeps) {
  return create<SessionState>((set) => ({
    userId: undefined,
    hydrate: () => {
      const existing = storage.getItem(USER_ID_KEY);
      if (existing) {
        set({ userId: existing });
        return;
      }
      storage.setItem(USER_ID_KEY, DEV_USER_ID);
      set({ userId: DEV_USER_ID });
    },
    clearSession: () => {
      storage.removeItem(USER_ID_KEY);
      set({ userId: undefined });
    }
  }));
}

export const taroStorage: KeyValueStorage = {
  getItem: (key) => {
    const value = Taro.getStorageSync<string>(key);
    return value === "" ? undefined : value;
  },
  setItem: (key, value) => Taro.setStorageSync(key, value),
  removeItem: (key) => Taro.removeStorageSync(key)
};

export const useSessionStore = createSessionStore({ storage: taroStorage });

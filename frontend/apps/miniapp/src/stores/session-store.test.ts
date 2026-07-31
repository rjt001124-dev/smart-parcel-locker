import { describe, expect, it, vi } from "vitest";

import { createSessionStore } from "./session-store";

function memoryStorage(initial: Record<string, string> = {}) {
  const data = new Map(Object.entries(initial));
  return {
    data,
    getItem: vi.fn((key: string) => data.get(key)),
    setItem: vi.fn((key: string, value: string) => {
      data.set(key, value);
    }),
    removeItem: vi.fn((key: string) => {
      data.delete(key);
    })
  };
}

describe("session store", () => {
  it("hydrates existing user id from storage", () => {
    const storage = memoryStorage({ "spl.session.userId": "dev-user" });
    const store = createSessionStore({ storage });

    store.getState().hydrate();

    expect(store.getState().userId).toBe("dev-user");
    expect(storage.getItem).toHaveBeenCalledWith("spl.session.userId");
  });

  it("creates and persists dev user when storage is empty", () => {
    const storage = memoryStorage();
    const store = createSessionStore({ storage });

    store.getState().hydrate();

    expect(store.getState().userId).toBe("dev-user");
    expect(storage.setItem).toHaveBeenCalledWith(
      "spl.session.userId",
      "dev-user"
    );
  });

  it("clearSession removes persisted user", () => {
    const storage = memoryStorage({ "spl.session.userId": "dev-user" });
    const store = createSessionStore({ storage });
    store.getState().hydrate();

    store.getState().clearSession();

    expect(store.getState().userId).toBeUndefined();
    expect(storage.removeItem).toHaveBeenCalledWith("spl.session.userId");
  });
});

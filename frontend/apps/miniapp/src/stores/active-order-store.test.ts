import { describe, expect, it, vi } from "vitest";

import { createActiveOrderStore } from "./active-order-store";

function memoryStorage(initial: Record<string, string> = {}) {
  const data = new Map(Object.entries(initial));
  return {
    getItem: vi.fn((key: string) => data.get(key)),
    setItem: vi.fn((key: string, value: string) => {
      data.set(key, value);
    }),
    removeItem: vi.fn((key: string) => {
      data.delete(key);
    })
  };
}

describe("active order store", () => {
  it("hydrates active order id from storage", () => {
    const storage = memoryStorage({ "spl.activeOrderId": "order-1" });
    const store = createActiveOrderStore({ storage });

    store.getState().hydrate();

    expect(store.getState().activeOrderId).toBe("order-1");
  });

  it("setActiveOrder persists the id", () => {
    const storage = memoryStorage();
    const store = createActiveOrderStore({ storage });

    store.getState().setActiveOrder("order-9");

    expect(store.getState().activeOrderId).toBe("order-9");
    expect(storage.setItem).toHaveBeenCalledWith(
      "spl.activeOrderId",
      "order-9"
    );
  });

  it("clearActiveOrder removes the persisted id", () => {
    const storage = memoryStorage({ "spl.activeOrderId": "order-1" });
    const store = createActiveOrderStore({ storage });
    store.getState().hydrate();

    store.getState().clearActiveOrder();

    expect(store.getState().activeOrderId).toBeUndefined();
    expect(storage.removeItem).toHaveBeenCalledWith("spl.activeOrderId");
  });

  it("hydrate leaves state empty when storage has no value", () => {
    const storage = memoryStorage();
    const store = createActiveOrderStore({ storage });

    store.getState().hydrate();

    expect(store.getState().activeOrderId).toBeUndefined();
  });
});

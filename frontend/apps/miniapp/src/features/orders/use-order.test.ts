import { renderHook, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const { getOrder, AppError } = vi.hoisted(() => {
  class MockAppError extends Error {
    constructor(
      public readonly code: string,
      message: string,
      public readonly traceId?: string
    ) {
      super(message);
      this.name = "AppError";
    }
  }
  return { getOrder: vi.fn(), AppError: MockAppError };
});

vi.mock("@tarojs/taro", () => ({
  default: {
    getStorageSync: vi.fn().mockReturnValue(""),
    setStorageSync: vi.fn(),
    removeStorageSync: vi.fn()
  },
  getCurrentInstance: () => ({ router: { params: {} } })
}));

vi.mock("@spl/api-client/http", () => ({
  createTaroRequest: () => vi.fn(),
  AppError: AppError
}));

vi.mock("@spl/api-client/order-client", () => ({
  createOrderClient: () => ({ getOrder })
}));

import { orderActionRoute, useOrder } from "./use-order";

const ORDER = {
  id: "ord-1",
  order_no: "SPL20260728001",
  site_id: "1",
  site_name: "人民广场寄存点",
  cell_id: "cell-1",
  cell_no: "A-01",
  size: "CELL_SIZE_SMALL",
  status: "ORDER_STATUS_AWAITING_DEPOSIT",
  duration_minutes: 120,
  fee_snapshot: {
    rent_fee_fen: 600,
    deposit_fen: 0,
    discount_fen: 0,
    total_fen: 600
  },
  overdue_fee_fen: 0,
  created_at: "2026-07-28T10:00:00Z",
  paid_at: "",
  deposited_at: "",
  expires_at: "",
  completed_at: ""
};

describe("orderActionRoute", () => {
  it("routes navigation actions to their pages", () => {
    expect(orderActionRoute("pay", "ord-1")).toBe(
      "/pages/payment/index?orderId=ord-1"
    );
    expect(orderActionRoute("openDepositDoor", "ord-1")).toBe(
      "/pages/store/index?orderId=ord-1"
    );
    expect(orderActionRoute("openPickupDoor", "ord-1")).toBe(
      "/pages/pickup/index?orderId=ord-1"
    );
    expect(orderActionRoute("settleOverdue", "ord-1")).toBe(
      "/pages/overdue/index?orderId=ord-1"
    );
  });

  it("returns null for inline actions (cancel, contactSupport)", () => {
    expect(orderActionRoute("cancel", "ord-1")).toBeNull();
    expect(orderActionRoute("contactSupport", "ord-1")).toBeNull();
  });
});

describe("useOrder", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });
  afterEach(() => {
    delete (globalThis as Record<string, unknown>).TARO_APP_API_BASE_URL;
  });

  it("loads the order and exposes it on success", async () => {
    getOrder.mockResolvedValue(ORDER);
    const { result } = renderHook(() => useOrder("ord-1"));

    expect(result.current.loadState).toBe("loading");

    await waitFor(() => {
      expect(result.current.loadState).toBe("success");
    });
    expect(result.current.order?.id).toBe("ord-1");
    expect(getOrder).toHaveBeenCalledWith("ord-1");
  });

  it("surfaces server error code and trace id", async () => {
    getOrder.mockRejectedValue(new AppError("ORDER_NOT_FOUND", "missing", "trace-9"));
    const { result } = renderHook(() => useOrder("ord-9"));

    await waitFor(() => {
      expect(result.current.loadState).toBe("error");
    });
    expect(result.current.code).toBe("ORDER_NOT_FOUND");
    expect(result.current.traceId).toBe("trace-9");
  });
});

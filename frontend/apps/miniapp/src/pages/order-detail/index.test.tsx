import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const { navigateTo, getOrder, cancelOrder } = vi.hoisted(() => ({
  navigateTo: vi.fn(),
  getOrder: vi.fn(),
  cancelOrder: vi.fn()
}));

let routerParams: Record<string, string> = {};

vi.mock("@tarojs/taro", () => ({
  default: {
    navigateTo: (...args: unknown[]) => navigateTo(...args),
    getStorageSync: vi.fn().mockReturnValue(""),
    setStorageSync: vi.fn(),
    removeStorageSync: vi.fn()
  },
  getCurrentInstance: () => ({ router: { params: routerParams } })
}));

vi.mock("@spl/api-client/http", () => ({
  createTaroRequest: () => vi.fn(),
  AppError: class MockAppError extends Error {
    constructor(
      public readonly code: string,
      message: string,
      public readonly traceId?: string
    ) {
      super(message);
      this.name = "AppError";
    }
  }
}));

vi.mock("@spl/api-client/order-client", () => ({
  createOrderClient: () => ({ getOrder, cancelOrder })
}));

import OrderDetailPage from "./index";

const AWAITING_DEPOSIT_ORDER = {
  id: "ord-1",
  order_no: "SPL20260728001",
  site_id: "1",
  site_name: "人民广场寄存点",
  cell_id: "cell-1",
  cell_no: "A-01",
  size: "CELL_SIZE_SMALL",
  status: "ORDER_STATUS_AWAITING_DEPOSIT",
  duration_minutes: 120,
  fee_snapshot: { rent_fee_fen: 600, deposit_fen: 0, discount_fen: 0, total_fen: 600 },
  overdue_fee_fen: 0,
  created_at: "", paid_at: "", deposited_at: "", expires_at: "", completed_at: ""
};

beforeEach(() => {
  vi.clearAllMocks();
  routerParams = { orderId: "ord-1" };
});
afterEach(() => {
  delete (globalThis as Record<string, unknown>).TARO_APP_API_BASE_URL;
});

describe("OrderDetailPage", () => {
  it("renders order status, info, timeline and fee summary", async () => {
    getOrder.mockResolvedValue(AWAITING_DEPOSIT_ORDER);
    render(<OrderDetailPage />);

    await waitFor(() => {
      expect(screen.getByText("人民广场寄存点")).toBeInTheDocument();
    });
    expect(screen.getAllByText("待存入").length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText("¥6.00").length).toBeGreaterThanOrEqual(1);
    expect(screen.getByText("寄存中")).toBeInTheDocument();
  });

  it("offers server-derived actions and navigates for door opening", async () => {
    getOrder.mockResolvedValue(AWAITING_DEPOSIT_ORDER);
    render(<OrderDetailPage />);

    await waitFor(() => {
      expect(screen.getByText("开门存入")).toBeInTheDocument();
    });
    expect(screen.getByText("联系客服")).toBeInTheDocument();

    fireEvent.click(screen.getByText("开门存入"));
    expect(navigateTo).toHaveBeenCalledWith({
      url: "/pages/store/index?orderId=ord-1"
    });
  });

  it("opens a contact panel when contact support is chosen", async () => {
    getOrder.mockResolvedValue(AWAITING_DEPOSIT_ORDER);
    render(<OrderDetailPage />);

    await waitFor(() => {
      expect(screen.getByText("联系客服")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("联系客服"));

    expect(
      screen.getByText((content) => content.includes("提供订单号"))
    ).toBeInTheDocument();
    expect(navigateTo).not.toHaveBeenCalled();
  });

  it("cancels the order inline and reloads", async () => {
    getOrder.mockResolvedValue({
      ...AWAITING_DEPOSIT_ORDER,
      status: "ORDER_STATUS_PENDING_PAYMENT"
    });
    cancelOrder.mockResolvedValue({
      ...AWAITING_DEPOSIT_ORDER,
      status: "ORDER_STATUS_CANCELLED"
    });
    render(<OrderDetailPage />);

    await waitFor(() => {
      expect(screen.getByText("取消订单")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("取消订单"));

    await waitFor(() => {
      expect(cancelOrder).toHaveBeenCalledWith("ord-1");
    });
  });
});

describe("OrderDetailPage auto-refresh", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.useFakeTimers();
    routerParams = { orderId: "ord-1" };
  });
  afterEach(() => {
    vi.useRealTimers();
    delete (globalThis as Record<string, unknown>).TARO_APP_API_BASE_URL;
  });

  it("polls order status every 5 seconds for non-terminal orders", async () => {
    getOrder.mockResolvedValue(AWAITING_DEPOSIT_ORDER);
    render(<OrderDetailPage />);

    await vi.advanceTimersByTimeAsync(0);
    expect(getOrder).toHaveBeenCalledTimes(1);

    await vi.advanceTimersByTimeAsync(5000);
    expect(getOrder).toHaveBeenCalledTimes(2);

    await vi.advanceTimersByTimeAsync(5000);
    expect(getOrder).toHaveBeenCalledTimes(3);
  });

  it("stops polling when order reaches a terminal status", async () => {
    const completedOrder = {
      ...AWAITING_DEPOSIT_ORDER,
      status: "ORDER_STATUS_COMPLETED" as const
    };
    getOrder.mockResolvedValue(completedOrder);
    render(<OrderDetailPage />);

    await vi.advanceTimersByTimeAsync(0);
    expect(getOrder).toHaveBeenCalledTimes(1);

    await vi.advanceTimersByTimeAsync(10000);
    expect(getOrder).toHaveBeenCalledTimes(1);
  });
});

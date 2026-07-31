import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const { navigateTo, listOrders } = vi.hoisted(() => ({
  navigateTo: vi.fn(),
  listOrders: vi.fn()
}));

vi.mock("@tarojs/taro", () => ({
  default: {
    navigateTo: (...args: unknown[]) => navigateTo(...args),
    getStorageSync: vi.fn().mockReturnValue(""),
    setStorageSync: vi.fn(),
    removeStorageSync: vi.fn()
  },
  getCurrentInstance: () => ({ router: { params: {} } })
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
  createOrderClient: () => ({ listOrders })
}));

import OrdersPage from "./index";

const ORDERS = [
  {
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
  },
  {
    id: "ord-2",
    order_no: "SPL20260728002",
    site_id: "2",
    site_name: "徐家汇寄存点",
    cell_id: "cell-2",
    cell_no: "B-03",
    size: "CELL_SIZE_MEDIUM",
    status: "ORDER_STATUS_COMPLETED",
    duration_minutes: 60,
    fee_snapshot: { rent_fee_fen: 300, deposit_fen: 0, discount_fen: 0, total_fen: 300 },
    overdue_fee_fen: 0,
    created_at: "", paid_at: "", deposited_at: "", expires_at: "", completed_at: ""
  }
];

beforeEach(() => {
  vi.clearAllMocks();
});
afterEach(() => {
  delete (globalThis as Record<string, unknown>).TARO_APP_API_BASE_URL;
});

describe("OrdersPage", () => {
  it("renders each order with site name, status and order number", async () => {
    listOrders.mockResolvedValue({ orders: ORDERS });
    render(<OrdersPage />);

    await waitFor(() => {
      expect(screen.getByText("人民广场寄存点")).toBeInTheDocument();
    });
    expect(screen.getByText("徐家汇寄存点")).toBeInTheDocument();
    expect(screen.getByText(/SPL20260728001/)).toBeInTheDocument();
    expect(screen.getByText("待存入")).toBeInTheDocument();
    expect(screen.getByText("已完成")).toBeInTheDocument();
  });

  it("navigates to order detail when a row is tapped", async () => {
    listOrders.mockResolvedValue({ orders: ORDERS });
    render(<OrdersPage />);

    await waitFor(() => {
      expect(screen.getByText("人民广场寄存点")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("人民广场寄存点"));

    expect(navigateTo).toHaveBeenCalledWith({
      url: "/pages/order-detail/index?orderId=ord-1"
    });
  });

  it("shows an empty state when there are no orders", async () => {
    listOrders.mockResolvedValue({ orders: [] });
    render(<OrdersPage />);

    await waitFor(() => {
      expect(screen.getByText("暂无订单")).toBeInTheDocument();
    });
  });

  it("shows a network error state on failure", async () => {
    listOrders.mockRejectedValue(new Error("boom"));
    render(<OrdersPage />);

    await waitFor(() => {
      expect(screen.getByText(/网络连接失败/)).toBeInTheDocument();
    });
  });
});

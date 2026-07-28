import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const {
  navigateTo,
  showToast,
  createOrder,
  getOrder,
  cancelOrder,
  confirmDevelopmentPayment
} = vi.hoisted(() => ({
  navigateTo: vi.fn(),
  showToast: vi.fn(),
  createOrder: vi.fn(),
  getOrder: vi.fn(),
  cancelOrder: vi.fn(),
  confirmDevelopmentPayment: vi.fn()
}));

let routerParams: Record<string, string> = {};

vi.mock("@tarojs/taro", () => ({
  default: {
    navigateTo: (...args: unknown[]) => navigateTo(...args),
    showToast: (...args: unknown[]) => showToast(...args),
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
  createOrderClient: () => ({
    createOrder,
    getOrder,
    cancelOrder,
    confirmDevelopmentPayment
  })
}));

import LockerSelectionPage from "./locker-selection";
import OrderConfirmPage from "./order-confirm";
import PaymentPage from "./payment";

const PENDING_ORDER = {
  id: "ord-1",
  order_no: "SPL20260728001",
  site_id: "1",
  site_name: "人民广场寄存点",
  cell_id: "cell-1",
  cell_no: "A-01",
  size: "CELL_SIZE_SMALL",
  status: "ORDER_STATUS_PENDING_PAYMENT",
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

beforeEach(() => {
  vi.clearAllMocks();
  routerParams = {};
});

afterEach(() => {
  delete (globalThis as Record<string, unknown>).TARO_APP_DEV_PAYMENT_SIMULATOR;
});

describe("LockerSelectionPage", () => {
  it("lets the user pick a size and duration then continue", () => {
    routerParams = { siteId: "1", siteName: "人民广场寄存点" };
    render(<LockerSelectionPage />);

    expect(screen.getByText("人民广场寄存点")).toBeInTheDocument();
    fireEvent.click(screen.getByText("中号"));
    fireEvent.click(screen.getByText("4小时"));
    fireEvent.click(screen.getByText("下一步"));

    expect(navigateTo).toHaveBeenCalledWith({
      url:
        "/pages/order-confirm/index?siteId=1&size=CELL_SIZE_MEDIUM&durationMinutes=240"
    });
  });
});

describe("OrderConfirmPage", () => {
  it("creates the order and renders the server fee snapshot", async () => {
    routerParams = {
      siteId: "1",
      size: "CELL_SIZE_SMALL",
      durationMinutes: "120"
    };
    createOrder.mockResolvedValue(PENDING_ORDER);
    render(<OrderConfirmPage />);

    await waitFor(() => {
      expect(screen.getByText("人民广场寄存点")).toBeInTheDocument();
    });
    expect(createOrder).toHaveBeenCalledWith({
      siteId: "1",
      size: "CELL_SIZE_SMALL",
      durationMinutes: 120
    });
    // server fee snapshot: rent 600 fen and total 600 fen
    expect(screen.getAllByText("¥6.00").length).toBeGreaterThanOrEqual(1);

    fireEvent.click(screen.getByText("去支付"));
    expect(navigateTo).toHaveBeenCalledWith({
      url: "/pages/payment/index?orderId=ord-1"
    });
  });

  it("shows a network error state when order creation fails", async () => {
    routerParams = {
      siteId: "1",
      size: "CELL_SIZE_SMALL",
      durationMinutes: "120"
    };
    createOrder.mockRejectedValue(new Error("boom"));
    render(<OrderConfirmPage />);

    await waitFor(() => {
      expect(screen.getByText(/网络连接失败/)).toBeInTheDocument();
    });
  });
});

describe("PaymentPage", () => {
  it("offers the development payment simulator when enabled", async () => {
    (globalThis as Record<string, unknown>).TARO_APP_DEV_PAYMENT_SIMULATOR =
      "true";
    routerParams = { orderId: "ord-1" };
    getOrder.mockResolvedValue(PENDING_ORDER);
    confirmDevelopmentPayment.mockResolvedValue({
      ...PENDING_ORDER,
      status: "ORDER_STATUS_AWAITING_DEPOSIT",
      paid_at: "2026-07-28T10:05:00Z"
    });
    render(<PaymentPage />);

    await waitFor(() => {
      expect(screen.getByText("模拟支付（开发环境）")).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText("模拟支付（开发环境）"));

    await waitFor(() => {
      expect(confirmDevelopmentPayment).toHaveBeenCalledWith("ord-1");
      expect(screen.getByText("待存入")).toBeInTheDocument();
    });
  });

  it("never fakes payment when the simulator is disabled", async () => {
    routerParams = { orderId: "ord-1" };
    getOrder.mockResolvedValue(PENDING_ORDER);
    render(<PaymentPage />);

    await waitFor(() => {
      expect(screen.getByText("待支付")).toBeInTheDocument();
    });
    expect(screen.queryByText("模拟支付（开发环境）")).not.toBeInTheDocument();
    expect(screen.getByText(/真实支付渠道尚未接入/)).toBeInTheDocument();
  });
});

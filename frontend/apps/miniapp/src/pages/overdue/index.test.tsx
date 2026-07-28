import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const { getOrder, settleOverdue } = vi.hoisted(() => ({
  getOrder: vi.fn(),
  settleOverdue: vi.fn()
}));

let routerParams: Record<string, string> = {};

vi.mock("@tarojs/taro", () => ({
  default: {
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
  createOrderClient: () => ({ getOrder, settleOverdue })
}));

import OverduePage from "./index";

const OVERDUE_ORDER = {
  id: "ord-1",
  order_no: "SPL20260728001",
  site_id: "1",
  site_name: "人民广场寄存点",
  cell_id: "cell-1",
  cell_no: "A-01",
  size: "CELL_SIZE_SMALL",
  status: "ORDER_STATUS_OVERDUE",
  duration_minutes: 120,
  fee_snapshot: { rent_fee_fen: 600, deposit_fen: 0, discount_fen: 0, total_fen: 600 },
  overdue_fee_fen: 300,
  created_at: "", paid_at: "", deposited_at: "", expires_at: "", completed_at: ""
};

beforeEach(() => {
  vi.clearAllMocks();
  routerParams = { orderId: "ord-1" };
});
afterEach(() => {
  delete (globalThis as Record<string, unknown>).TARO_APP_API_BASE_URL;
  delete (globalThis as Record<string, unknown>).TARO_APP_DEV_PAYMENT_SIMULATOR;
});

describe("OverduePage", () => {
  it("shows the development simulator and settles via the server", async () => {
    (globalThis as Record<string, unknown>).TARO_APP_DEV_PAYMENT_SIMULATOR = "true";
    getOrder.mockResolvedValueOnce(OVERDUE_ORDER);
    getOrder.mockResolvedValue({
      ...OVERDUE_ORDER,
      status: "ORDER_STATUS_AWAITING_PICKUP",
      overdue_fee_fen: 0
    });
    settleOverdue.mockResolvedValue({
      ...OVERDUE_ORDER,
      status: "ORDER_STATUS_AWAITING_PICKUP",
      overdue_fee_fen: 0
    });
    render(<OverduePage />);

    await waitFor(() => {
      expect(screen.getByText("补缴逾期费（开发环境）")).toBeInTheDocument();
    });
    expect(screen.getByText("¥3.00")).toBeInTheDocument();
    fireEvent.click(screen.getByText("补缴逾期费（开发环境）"));

    await waitFor(() => {
      expect(settleOverdue).toHaveBeenCalledWith("ord-1");
      expect(screen.getByText("逾期费用已结清")).toBeInTheDocument();
    });
  });

  it("never fakes payment when the simulator is disabled", async () => {
    getOrder.mockResolvedValue(OVERDUE_ORDER);
    render(<OverduePage />);

    await waitFor(() => {
      expect(screen.getByText("已逾期")).toBeInTheDocument();
    });
    expect(screen.queryByText("补缴逾期费（开发环境）")).not.toBeInTheDocument();
    expect(screen.getByText(/真实支付渠道尚未接入/)).toBeInTheDocument();
  });
});

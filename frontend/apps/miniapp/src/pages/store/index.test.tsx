import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const { navigateTo, getOrder, openDepositDoor } = vi.hoisted(() => ({
  navigateTo: vi.fn(),
  getOrder: vi.fn(),
  openDepositDoor: vi.fn()
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
  createOrderClient: () => ({ getOrder, openDepositDoor })
}));

import StorePage from "./index";

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

describe("StorePage", () => {
  it("renders order info and an idle door panel", async () => {
    getOrder.mockResolvedValue(AWAITING_DEPOSIT_ORDER);
    render(<StorePage />);

    await waitFor(() => {
      expect(screen.getByText("人民广场寄存点")).toBeInTheDocument();
    });
    expect(screen.getByText("开门存入")).toBeInTheDocument();
    expect(screen.getAllByText("¥6.00").length).toBeGreaterThanOrEqual(1);
  });

  it("opens the deposit door only after a server success", async () => {
    getOrder.mockResolvedValue(AWAITING_DEPOSIT_ORDER);
    openDepositDoor.mockResolvedValue(AWAITING_DEPOSIT_ORDER);
    render(<StorePage />);

    await waitFor(() => {
      expect(screen.getByText("开门存入")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("开门存入"));

    await waitFor(() => {
      expect(openDepositDoor).toHaveBeenCalledWith("ord-1");
      expect(screen.getByText(/柜门已打开/)).toBeInTheDocument();
    });
    expect(screen.queryByText("开门存入")).not.toBeInTheDocument();
  });

  it("shows a retry panel when the door fails to open", async () => {
    getOrder.mockResolvedValue(AWAITING_DEPOSIT_ORDER);
    openDepositDoor.mockRejectedValue(new Error("boom"));
    render(<StorePage />);

    await waitFor(() => {
      expect(screen.getByText("开门存入")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("开门存入"));

    await waitFor(() => {
      expect(screen.getByText("重试")).toBeInTheDocument();
    });
  });

  it("navigates to order detail after the door opens", async () => {
    getOrder.mockResolvedValue(AWAITING_DEPOSIT_ORDER);
    openDepositDoor.mockResolvedValue(AWAITING_DEPOSIT_ORDER);
    render(<StorePage />);

    await waitFor(() => {
      expect(screen.getByText("开门存入")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("开门存入"));

    await waitFor(() => {
      expect(screen.getByText("查看订单")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("查看订单"));
    expect(navigateTo).toHaveBeenCalledWith({
      url: "/pages/order-detail/index?orderId=ord-1"
    });
  });
});

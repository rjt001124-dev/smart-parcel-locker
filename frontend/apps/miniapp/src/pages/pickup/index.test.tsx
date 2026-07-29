import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const { navigateTo, getOrder, openPickupDoor } = vi.hoisted(() => ({
  navigateTo: vi.fn(),
  getOrder: vi.fn(),
  openPickupDoor: vi.fn()
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
  createOrderClient: () => ({ getOrder, openPickupDoor })
}));

import PickupPage from "./index";

const AWAITING_PICKUP_ORDER = {
  id: "ord-1",
  order_no: "SPL20260728001",
  site_id: "1",
  site_name: "人民广场寄存点",
  cell_id: "cell-1",
  cell_no: "A-01",
  size: "CELL_SIZE_SMALL",
  status: "ORDER_STATUS_AWAITING_PICKUP",
  duration_minutes: 120,
  fee_snapshot: { rent_fee_fen: 600, deposit_fen: 0, discount_fen: 0, total_fen: 600 },
  overdue_fee_fen: 0,
  created_at: "", paid_at: "", deposited_at: "", expires_at: "", completed_at: ""
};

async function enterPickupCode(code: string) {
  const input = screen.getByPlaceholderText(/请输入柜格号/);
  fireEvent.input(input, { target: { value: code } });
}

beforeEach(() => {
  vi.clearAllMocks();
  routerParams = { orderId: "ord-1" };
});
afterEach(() => {
  delete (globalThis as Record<string, unknown>).TARO_APP_API_BASE_URL;
});

describe("PickupPage", () => {
  it("renders order info and a pickup code input", async () => {
    getOrder.mockResolvedValue(AWAITING_PICKUP_ORDER);
    render(<PickupPage />);

    await waitFor(() => {
      expect(screen.getByText("人民广场寄存点")).toBeInTheDocument();
    });
    expect(screen.getByPlaceholderText(/请输入柜格号/)).toBeInTheDocument();
  });

  it("does not show the door-open button until the correct code is entered", async () => {
    getOrder.mockResolvedValue(AWAITING_PICKUP_ORDER);
    render(<PickupPage />);

    await waitFor(() => {
      expect(screen.getByText("人民广场寄存点")).toBeInTheDocument();
    });
    expect(screen.queryByText("开门取件")).not.toBeInTheDocument();

    await enterPickupCode("A-01");
    expect(screen.getByText("开门取件")).toBeInTheDocument();
  });

  it("shows an error when the pickup code does not match", async () => {
    getOrder.mockResolvedValue(AWAITING_PICKUP_ORDER);
    render(<PickupPage />);

    await waitFor(() => {
      expect(screen.getByText("人民广场寄存点")).toBeInTheDocument();
    });

    await enterPickupCode("B-99");
    expect(screen.getByText("柜格号不匹配")).toBeInTheDocument();
    expect(screen.queryByText("开门取件")).not.toBeInTheDocument();
  });

  it("clears the error and enables open when the correct code is entered", async () => {
    getOrder.mockResolvedValue(AWAITING_PICKUP_ORDER);
    render(<PickupPage />);

    await waitFor(() => {
      expect(screen.getByText("人民广场寄存点")).toBeInTheDocument();
    });

    await enterPickupCode("B-99");
    expect(screen.getByText("柜格号不匹配")).toBeInTheDocument();

    await enterPickupCode("A-01");
    expect(screen.queryByText("柜格号不匹配")).not.toBeInTheDocument();
    expect(screen.getByText("开门取件")).toBeInTheDocument();
  });

  it("opens the pickup door only after correct code and server success", async () => {
    getOrder.mockResolvedValue(AWAITING_PICKUP_ORDER);
    openPickupDoor.mockResolvedValue(AWAITING_PICKUP_ORDER);
    render(<PickupPage />);

    await waitFor(() => {
      expect(screen.getByText("人民广场寄存点")).toBeInTheDocument();
    });

    await enterPickupCode("A-01");
    fireEvent.click(screen.getByText("开门取件"));

    await waitFor(() => {
      expect(openPickupDoor).toHaveBeenCalledWith("ord-1");
      expect(screen.getByText(/柜门已打开/)).toBeInTheDocument();
    });
  });

  it("shows a retry panel when the door fails to open", async () => {
    getOrder.mockResolvedValue(AWAITING_PICKUP_ORDER);
    openPickupDoor.mockRejectedValue(new Error("boom"));
    render(<PickupPage />);

    await waitFor(() => {
      expect(screen.getByText("人民广场寄存点")).toBeInTheDocument();
    });

    await enterPickupCode("A-01");
    fireEvent.click(screen.getByText("开门取件"));

    await waitFor(() => {
      expect(screen.getByText("重试")).toBeInTheDocument();
    });
  });
});

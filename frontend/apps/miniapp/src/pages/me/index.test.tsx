import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const { navigateTo, listOrders } = vi.hoisted(() => ({
  navigateTo: vi.fn(),
  listOrders: vi.fn()
}));

vi.mock("@tarojs/taro", () => ({
  default: {
    navigateTo: (...args: unknown[]) => navigateTo(...args),
    getStorageSync: vi.fn().mockImplementation((key: string) => {
      if (key === "spl.session.userId") return "dev-user";
      return "";
    }),
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

import MePage from "./index";

beforeEach(() => {
  vi.clearAllMocks();
});
afterEach(() => {
  delete (globalThis as Record<string, unknown>).TARO_APP_DEV_PAYMENT_SIMULATOR;
});

describe("MePage", () => {
  it("renders the dev user id and app title", async () => {
    listOrders.mockResolvedValue({ orders: [] });
    render(<MePage />);

    await waitFor(() => {
      expect(screen.getByText("dev-user")).toBeInTheDocument();
    });
    expect(screen.getByText("智能快递柜")).toBeInTheDocument();
  });

  it("shows the order count with a link to the orders page", async () => {
    listOrders.mockResolvedValue({
      orders: [
        { id: "o1", status: "ORDER_STATUS_COMPLETED" },
        { id: "o2", status: "ORDER_STATUS_AWAITING_PICKUP" }
      ]
    });
    render(<MePage />);

    await waitFor(() => {
      expect(screen.getByText("2")).toBeInTheDocument();
    });
    expect(screen.getByText("我的订单")).toBeInTheDocument();

    fireEvent.click(screen.getByText("我的订单"));
    expect(navigateTo).toHaveBeenCalledWith({
      url: "/pages/orders/index"
    });
  });

  it("shows a help section with FAQ entries", async () => {
    listOrders.mockResolvedValue({ orders: [] });
    render(<MePage />);

    await waitFor(() => {
      expect(screen.getByText("帮助与反馈")).toBeInTheDocument();
    });
    expect(screen.getByText("如何存取物品？")).toBeInTheDocument();
    expect(screen.getByText("逾期费怎么算？")).toBeInTheDocument();
  });

  it("indicates the dev payment simulator status when enabled", async () => {
    (globalThis as Record<string, unknown>).TARO_APP_DEV_PAYMENT_SIMULATOR = "true";
    listOrders.mockResolvedValue({ orders: [] });
    render(<MePage />);

    await waitFor(() => {
      expect(screen.getByText(/开发模拟器已开启/)).toBeInTheDocument();
    });
  });

  it("does not show the dev simulator banner when disabled", async () => {
    listOrders.mockResolvedValue({ orders: [] });
    render(<MePage />);

    await waitFor(() => {
      expect(screen.getByText("dev-user")).toBeInTheDocument();
    });
    expect(screen.queryByText(/开发模拟器已开启/)).not.toBeInTheDocument();
  });
});

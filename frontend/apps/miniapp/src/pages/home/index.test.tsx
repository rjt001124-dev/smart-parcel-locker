import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const { navigateTo } = vi.hoisted(() => ({ navigateTo: vi.fn() }));

let activeOrderId = "";

vi.mock("@tarojs/taro", () => ({
  default: {
    navigateTo: (...args: unknown[]) => navigateTo(...args),
    getStorageSync: vi.fn().mockReturnValue(""),
    setStorageSync: vi.fn(),
    removeStorageSync: vi.fn()
  },
  getCurrentInstance: () => ({ router: { params: {} } })
}));

const MOCK_SITES = [
  {
    id: "1",
    name: "人民广场寄存点",
    address: "南京东路",
    distanceMeters: 500,
    onlineDeviceCount: 3,
    cellCounts: { small: 1, medium: 2, large: 0 }
  },
  {
    id: "2",
    name: "徐家汇寄存点",
    address: "肇嘉浜路",
    distanceMeters: 1200,
    onlineDeviceCount: 2,
    cellCounts: { small: 0, medium: 1, large: 1 }
  },
  {
    id: "3",
    name: "陆家嘴地铁站寄存点",
    address: "浦东陆家嘴",
    distanceMeters: 2000,
    onlineDeviceCount: 1,
    cellCounts: { small: 2, medium: 0, large: 0 }
  }
];

vi.mock("../../features/sites/use-sites", () => ({
  useSites: () => ({
    status: "success",
    sites: MOCK_SITES,
    retry: vi.fn()
  })
}));

vi.mock("../../features/orders/use-order", () => ({
  useOrder: () => ({
    order:
      activeOrderId === ""
        ? undefined
        : {
            id: activeOrderId,
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
    loadState: activeOrderId === "" ? "loading" : "success",
    reload: vi.fn()
  }),
  statusPillTone: () => "warning" as const
}));

import HomePage from "./index";
import { useLocationStore } from "../../stores/location-store";
import { useActiveOrderStore } from "../../stores/active-order-store";

beforeEach(() => {
  vi.clearAllMocks();
  activeOrderId = "";
  useLocationStore.setState({ cityName: "上海", cityCode: "021", latitude: 31, longitude: 121 });
  useActiveOrderStore.setState({ activeOrderId: "" });
});
afterEach(() => {
  delete (globalThis as Record<string, unknown>).TARO_APP_API_BASE_URL;
});

describe("HomePage active order card", () => {
  it("shows an empty current-order card when there is no active order", async () => {
    render(<HomePage />);
    await waitFor(() => {
      expect(screen.getByText("人民广场寄存点")).toBeInTheDocument();
    });
    expect(screen.getByText("暂无进行中的订单")).toBeInTheDocument();
  });

  it("renders the active order and navigates to detail on tap", async () => {
    activeOrderId = "ord-1";
    useActiveOrderStore.setState({ activeOrderId: "ord-1" });
    render(<HomePage />);

    await waitFor(() => {
      expect(screen.getByText("人民广场寄存点 · 柜格 A-01")).toBeInTheDocument();
    });
    expect(screen.getByText("当前订单")).toBeInTheDocument();

    fireEvent.click(screen.getByText("人民广场寄存点 · 柜格 A-01"));
    expect(navigateTo).toHaveBeenCalledWith({
      url: "/pages/order-detail/index?orderId=ord-1"
    });
  });
});

describe("HomePage search", () => {
  it("shows only the first two sites by default", async () => {
    render(<HomePage />);
    await waitFor(() => {
      expect(screen.getByText("人民广场寄存点")).toBeInTheDocument();
    });
    expect(screen.getByText("徐家汇寄存点")).toBeInTheDocument();
    expect(screen.queryByText("陆家嘴地铁站寄存点")).not.toBeInTheDocument();
  });

  it("filters sites by name when typing", async () => {
    render(<HomePage />);
    await waitFor(() => {
      expect(screen.getByText("人民广场寄存点")).toBeInTheDocument();
    });

    const input = screen.getByPlaceholderText("搜索商场、地铁站或地址");
    fireEvent.input(input, { target: { value: "陆家嘴" } });

    expect(screen.getByText("陆家嘴地铁站寄存点")).toBeInTheDocument();
    expect(screen.queryByText("人民广场寄存点")).not.toBeInTheDocument();
    expect(screen.queryByText("徐家汇寄存点")).not.toBeInTheDocument();
  });

  it("filters sites by address when typing", async () => {
    render(<HomePage />);
    await waitFor(() => {
      expect(screen.getByText("人民广场寄存点")).toBeInTheDocument();
    });

    const input = screen.getByPlaceholderText("搜索商场、地铁站或地址");
    fireEvent.input(input, { target: { value: "南京" } });

    expect(screen.getByText("人民广场寄存点")).toBeInTheDocument();
    expect(screen.queryByText("徐家汇寄存点")).not.toBeInTheDocument();
  });

  it("shows a no-results message when search matches nothing", async () => {
    render(<HomePage />);
    await waitFor(() => {
      expect(screen.getByText("人民广场寄存点")).toBeInTheDocument();
    });

    const input = screen.getByPlaceholderText("搜索商场、地铁站或地址");
    fireEvent.input(input, { target: { value: "不存在的地点" } });

    expect(screen.getByText("未找到匹配的网点")).toBeInTheDocument();
  });
});

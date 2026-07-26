import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import HomePage from "./home";
import SitesPage from "./sites";

vi.mock("@tarojs/taro", () => ({
  default: {
    navigateTo: vi.fn(),
    getLocation: vi.fn()
  },
  navigateTo: vi.fn(),
  getLocation: vi.fn()
}));

vi.mock("../features/sites/use-sites", () => ({
  useSites: () => ({
    status: "success",
    sites: [{
      id: "1",
      name: "万象城智能寄存点",
      address: "世纪大道88号B1层",
      distanceLabel: "320m",
      availabilityLabel: "可用24格",
      statusLabel: "有空柜",
      statusTone: "success"
    }],
    retry: vi.fn()
  })
}));

describe("site discovery pages", () => {
  it("keeps current-order space truthful when order API is not implemented", () => {
    render(<HomePage />);
    expect(screen.getByText("暂无进行中的订单")).toBeTruthy();
    expect(screen.getByText("万象城智能寄存点")).toBeTruthy();
  });

  it("renders the nearby-site result count", () => {
    render(<SitesPage />);
    expect(screen.getByText("共找到1个站点")).toBeTruthy();
  });
});

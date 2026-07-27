import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { StatePanel } from "./state-panel";

describe("StatePanel", () => {
  it("explains cause, impact, and next action for network failure", () => {
    const retry = vi.fn();
    render(<StatePanel kind="network" traceId="trace-123" onRetry={retry} />);

    expect(screen.getByText("网络连接失败")).toBeTruthy();
    expect(screen.getByText("尚未更改当前订单或柜格")).toBeTruthy();
    expect(screen.getByText("重试")).toBeTruthy();
    expect(screen.getByText("参考编号：trace-123")).toBeTruthy();
    fireEvent.click(screen.getByText("重试"));
    expect(retry).toHaveBeenCalledOnce();
  });

  it("gives an honest next action when no site is available", () => {
    render(<StatePanel kind="empty" onRetry={vi.fn()} />);
    expect(screen.getByText("暂无可用网点")).toBeTruthy();
    expect(screen.getByText("调整筛选")).toBeTruthy();
  });
});

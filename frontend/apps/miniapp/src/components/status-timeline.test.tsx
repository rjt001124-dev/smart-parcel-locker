import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { StatusTimeline } from "./status-timeline";

describe("StatusTimeline", () => {
  it("renders the standard storage flow steps", () => {
    render(<StatusTimeline status="ORDER_STATUS_IN_STORAGE" />);

    expect(screen.getByText("待支付")).toBeInTheDocument();
    expect(screen.getByText("待存入")).toBeInTheDocument();
    expect(screen.getByText("寄存中")).toBeInTheDocument();
    expect(screen.getByText("待取件")).toBeInTheDocument();
    expect(screen.getByText("已完成")).toBeInTheDocument();
  });

  it("marks current step and previous steps as done", () => {
    render(<StatusTimeline status="ORDER_STATUS_IN_STORAGE" />);

    const current = screen.getByText("寄存中");
    expect(current.className).toContain("status-timeline__label--current");

    const done = screen.getByText("待支付");
    expect(done.className).toContain("status-timeline__label--done");

    const upcoming = screen.getByText("待取件");
    expect(upcoming.className).toContain("status-timeline__label--upcoming");
  });

  it("shows overdue branch step when order is overdue", () => {
    render(<StatusTimeline status="ORDER_STATUS_OVERDUE" />);
    const overdue = screen.getByText("已逾期");
    expect(overdue).toBeInTheDocument();
    expect(overdue.className).toContain("status-timeline__label--current");
  });

  it("renders terminal cancelled state as a single step", () => {
    render(<StatusTimeline status="ORDER_STATUS_CANCELLED" />);
    const cancelled = screen.getByText("已取消");
    expect(cancelled.className).toContain("status-timeline__label--current");
    expect(screen.queryByText("寄存中")).not.toBeInTheDocument();
  });
});

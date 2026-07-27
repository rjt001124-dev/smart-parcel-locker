import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { PrimaryButton } from "./primary-button";
import { StatusPill } from "./status-pill";

describe("mini app primitives", () => {
  it("blocks duplicate action while loading", () => {
    const onClick = vi.fn();
    render(<PrimaryButton loading onClick={onClick} label="查看订单" />);
    fireEvent.click(screen.getByRole("button"));
    expect(onClick).not.toHaveBeenCalled();
    expect(screen.getByRole("button").className).toContain("primary-button--disabled");
    expect(screen.getByText("处理中…")).toBeTruthy();
  });

  it("renders semantic status copy", () => {
    render(<StatusPill tone="success" label="营业中" />);
    expect(screen.getByText("营业中")).toBeTruthy();
  });
});

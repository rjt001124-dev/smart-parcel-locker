import { render, screen, fireEvent } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { DoorStatusPanel } from "./door-status-panel";

describe("DoorStatusPanel", () => {
  it("renders idle state with action button enabled", () => {
    render(
      <DoorStatusPanel
        phase="idle"
        actionLabel="开门存入"
        onAction={() => undefined}
      />
    );
    expect(screen.getByText("开门存入")).toBeInTheDocument();
  });

  it("invokes onAction when the button is pressed", () => {
    const onAction = vi.fn();
    render(
      <DoorStatusPanel phase="idle" actionLabel="开门取件" onAction={onAction} />
    );
    fireEvent.click(screen.getByText("开门取件"));
    expect(onAction).toHaveBeenCalledTimes(1);
  });

  it("shows opening state and hides the action button", () => {
    render(
      <DoorStatusPanel
        phase="opening"
        actionLabel="开门存入"
        onAction={() => undefined}
      />
    );
    expect(screen.getByText("正在开门…")).toBeInTheDocument();
    expect(screen.queryByText("开门存入")).not.toBeInTheDocument();
  });

  it("shows server-confirmed open success message", () => {
    render(
      <DoorStatusPanel
        phase="opened"
        actionLabel="开门存入"
        onAction={() => undefined}
        message="柜门已打开，请存入物品"
      />
    );
    expect(screen.getByText("柜门已打开，请存入物品")).toBeInTheDocument();
  });

  it("shows failure message with retry action", () => {
    const onAction = vi.fn();
    render(
      <DoorStatusPanel
        phase="failed"
        actionLabel="开门存入"
        onAction={onAction}
        message="开门失败，请重试"
      />
    );
    expect(screen.getByText("开门失败，请重试")).toBeInTheDocument();
    fireEvent.click(screen.getByText("重试"));
    expect(onAction).toHaveBeenCalledTimes(1);
  });
});

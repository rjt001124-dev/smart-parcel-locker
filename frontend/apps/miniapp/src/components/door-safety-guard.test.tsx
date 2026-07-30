import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { DoorSafetyGuard } from "./door-safety-guard";

describe("DoorSafetyGuard", () => {
  it("shows the safety checklist, target cell, and ack checkbox while idle", () => {
    render(
      <DoorSafetyGuard
        phase="idle"
        actionLabel="开门取件"
        onAction={vi.fn()}
        cellNo="A-12"
      />
    );

    expect(screen.getByText("开门前安全确认")).toBeInTheDocument();
    expect(screen.getByText("目标柜格：A-12")).toBeInTheDocument();
    expect(screen.getByText("我已知晓并确认安全")).toBeInTheDocument();
    expect(screen.getByText("开门取件")).toBeInTheDocument();
  });

  it("does not call onAction until the safety box is acknowledged", () => {
    const onAction = vi.fn();
    render(
      <DoorSafetyGuard phase="idle" actionLabel="开门取件" onAction={onAction} />
    );

    fireEvent.click(screen.getByText("开门取件"));

    expect(onAction).not.toHaveBeenCalled();
    expect(screen.getByText("请先勾选安全确认")).toBeInTheDocument();
  });

  it("calls onAction after the safety box is acknowledged", () => {
    const onAction = vi.fn();
    render(
      <DoorSafetyGuard phase="idle" actionLabel="开门取件" onAction={onAction} />
    );

    fireEvent.click(screen.getByText("我已知晓并确认安全"));
    fireEvent.click(screen.getByText("开门取件"));

    expect(onAction).toHaveBeenCalledTimes(1);
  });

  it("hides the action button while the door is opening", () => {
    const onAction = vi.fn();
    render(
      <DoorSafetyGuard phase="opening" actionLabel="开门取件" onAction={onAction} />
    );

    expect(screen.getByText("正在开门…")).toBeInTheDocument();
    expect(screen.queryByText("开门取件")).toBeNull();
  });

  it("offers retry on failure and still requires acknowledgement", () => {
    const onAction = vi.fn();
    render(
      <DoorSafetyGuard phase="failed" actionLabel="开门取件" onAction={onAction} />
    );

    expect(screen.getByText("重试")).toBeInTheDocument();
    fireEvent.click(screen.getByText("重试"));
    expect(onAction).not.toHaveBeenCalled();

    fireEvent.click(screen.getByText("我已知晓并确认安全"));
    fireEvent.click(screen.getByText("重试"));
    expect(onAction).toHaveBeenCalledTimes(1);
  });
});

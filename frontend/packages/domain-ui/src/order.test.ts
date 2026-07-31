import { describe, expect, it } from "vitest";

import {
  availableOrderActions,
  orderActionLabel,
  toOrderStatusView,
  type OrderStatus
} from "./order";

describe("toOrderStatusView", () => {
  const cases: Array<[OrderStatus, string, string]> = [
    ["ORDER_STATUS_PENDING_PAYMENT", "待支付", "warning"],
    ["ORDER_STATUS_PAID", "已支付", "success"],
    ["ORDER_STATUS_AWAITING_DEPOSIT", "待存入", "info"],
    ["ORDER_STATUS_IN_STORAGE", "寄存中", "info"],
    ["ORDER_STATUS_AWAITING_PICKUP", "待取件", "info"],
    ["ORDER_STATUS_OVERDUE", "已逾期", "danger"],
    ["ORDER_STATUS_COMPLETED", "已完成", "success"],
    ["ORDER_STATUS_CANCELLED", "已取消", "muted"]
  ];

  it.each(cases)("maps %s to label %s tone %s", (status, label, tone) => {
    const view = toOrderStatusView(status);
    expect(view.label).toBe(label);
    expect(view.tone).toBe(tone);
  });

  it("falls back to unknown for unspecified status", () => {
    const view = toOrderStatusView("ORDER_STATUS_UNSPECIFIED");
    expect(view.label).toBe("未知状态");
    expect(view.tone).toBe("muted");
  });
});

describe("availableOrderActions", () => {
  it("pending payment offers pay and cancel", () => {
    expect(availableOrderActions("ORDER_STATUS_PENDING_PAYMENT")).toEqual([
      "pay",
      "cancel",
      "contactSupport"
    ]);
  });

  it("awaiting deposit offers openDepositDoor", () => {
    expect(availableOrderActions("ORDER_STATUS_AWAITING_DEPOSIT")).toEqual([
      "openDepositDoor",
      "contactSupport"
    ]);
  });

  it("in storage offers openPickupDoor", () => {
    expect(availableOrderActions("ORDER_STATUS_IN_STORAGE")).toEqual([
      "openPickupDoor",
      "contactSupport"
    ]);
  });

  it("overdue offers settleOverdue only", () => {
    expect(availableOrderActions("ORDER_STATUS_OVERDUE")).toEqual([
      "settleOverdue",
      "contactSupport"
    ]);
  });

  it("awaiting pickup offers openPickupDoor", () => {
    expect(availableOrderActions("ORDER_STATUS_AWAITING_PICKUP")).toEqual([
      "openPickupDoor",
      "contactSupport"
    ]);
  });

  it("terminal states only offer contactSupport", () => {
    expect(availableOrderActions("ORDER_STATUS_COMPLETED")).toEqual([
      "contactSupport"
    ]);
    expect(availableOrderActions("ORDER_STATUS_CANCELLED")).toEqual([
      "contactSupport"
    ]);
  });
});

describe("orderActionLabel", () => {
  it("labels every action in Chinese", () => {
    expect(orderActionLabel("pay")).toBe("去支付");
    expect(orderActionLabel("cancel")).toBe("取消订单");
    expect(orderActionLabel("openDepositDoor")).toBe("开门存入");
    expect(orderActionLabel("openPickupDoor")).toBe("开门取件");
    expect(orderActionLabel("settleOverdue")).toBe("补缴逾期费");
    expect(orderActionLabel("contactSupport")).toBe("联系客服");
  });
});

import { describe, expect, it, vi } from "vitest";
import { createOrderClient } from "./order-client";

const RAW_ORDER = {
  id: "ord-1",
  orderNo: "SPL20260728001",
  siteId: "1",
  siteName: "人民广场寄存点",
  cellId: "cell-1",
  cellNo: "A-01",
  size: "CELL_SIZE_SMALL",
  status: "ORDER_STATUS_PENDING_PAYMENT",
  durationMinutes: 120,
  feeSnapshot: {
    rentFeeFen: "600",
    depositFen: "0",
    discountFen: "0",
    totalFen: "600"
  },
  overdueFeeFen: "0",
  createdAt: "2026-07-28T10:00:00Z",
  paidAt: "",
  depositedAt: "",
  expiresAt: "",
  completedAt: ""
};

describe("order client", () => {
  it("creates an order with site, size and duration", async () => {
    const request = vi.fn().mockResolvedValue({ order: RAW_ORDER });
    const client = createOrderClient({ request });

    await client.createOrder({
      siteId: "1",
      size: "CELL_SIZE_SMALL",
      durationMinutes: 120
    });

    expect(request).toHaveBeenCalledWith({
      path: "/v1/orders",
      method: "POST",
      data: {
        site_id: "1",
        size: "CELL_SIZE_SMALL",
        duration_minutes: 120
      }
    });
  });

  it("normalizes protobuf JSON order fields to snake_case integers", async () => {
    const request = vi.fn().mockResolvedValue({ order: RAW_ORDER });
    const client = createOrderClient({ request });

    const order = await client.getOrder("ord-1");

    expect(request).toHaveBeenCalledWith({ path: "/v1/orders/ord-1" });
    expect(order).toMatchObject({
      id: "ord-1",
      order_no: "SPL20260728001",
      site_name: "人民广场寄存点",
      cell_no: "A-01",
      status: "ORDER_STATUS_PENDING_PAYMENT",
      duration_minutes: 120,
      fee_snapshot: {
        rent_fee_fen: 600,
        deposit_fen: 0,
        discount_fen: 0,
        total_fen: 600
      },
      overdue_fee_fen: 0
    });
    expect(Number.isInteger(order.fee_snapshot.total_fen)).toBe(true);
  });

  it("lists orders newest first as returned by the server", async () => {
    const request = vi.fn().mockResolvedValue({ orders: [RAW_ORDER] });
    const client = createOrderClient({ request });

    const result = await client.listOrders();

    expect(request).toHaveBeenCalledWith({ path: "/v1/orders" });
    expect(result.orders).toHaveLength(1);
    expect(result.orders[0]?.order_no).toBe("SPL20260728001");
  });

  it("cancels an order via the cancellation endpoint", async () => {
    const request = vi.fn().mockResolvedValue({ order: RAW_ORDER });
    const client = createOrderClient({ request });

    await client.cancelOrder("ord-1");

    expect(request).toHaveBeenCalledWith({
      path: "/v1/orders/ord-1/cancellations",
      method: "POST",
      data: {}
    });
  });

  it("requests deposit and pickup door opens through server endpoints only", async () => {
    const request = vi.fn().mockResolvedValue({ order: RAW_ORDER });
    const client = createOrderClient({ request });

    await client.openDepositDoor("ord-1");
    await client.openPickupDoor("ord-1");

    expect(request).toHaveBeenNthCalledWith(1, {
      path: "/v1/orders/ord-1/deposit-door-requests",
      method: "POST",
      data: {}
    });
    expect(request).toHaveBeenNthCalledWith(2, {
      path: "/v1/orders/ord-1/pickup-door-requests",
      method: "POST",
      data: {}
    });
  });

  it("exposes development-only payment confirmation and overdue settlement", async () => {
    const request = vi.fn().mockResolvedValue({ order: RAW_ORDER });
    const client = createOrderClient({ request });

    await client.confirmDevelopmentPayment("ord-1");
    await client.settleOverdue("ord-1");

    expect(request).toHaveBeenNthCalledWith(1, {
      path: "/v1/development/orders/ord-1/payment-confirmations",
      method: "POST",
      data: {}
    });
    expect(request).toHaveBeenNthCalledWith(2, {
      path: "/v1/development/orders/ord-1/overdue-payments",
      method: "POST",
      data: {}
    });
  });

  it("never sends the internal token from a public client", async () => {
    const request = vi.fn().mockResolvedValue({ order: RAW_ORDER });
    const client = createOrderClient({ request });

    await client.getOrder("ord-1");

    expect(JSON.stringify(request.mock.calls)).not.toContain("X-Internal-Token");
  });
});

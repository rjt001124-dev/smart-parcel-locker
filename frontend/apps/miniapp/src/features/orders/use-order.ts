import { useCallback, useEffect, useState } from "react";
import { AppError, createTaroRequest } from "@spl/api-client/http";
import { createOrderClient, type OrderDto } from "@spl/api-client/order-client";
import {
  availableOrderActions,
  orderActionLabel,
  toOrderStatusView,
  type OrderAction,
  type OrderStatus,
  type OrderStatusView
} from "@spl/domain-ui/order";

const API_BASE_URL =
  typeof TARO_APP_API_BASE_URL !== "undefined" ? TARO_APP_API_BASE_URL : "";

const client = createOrderClient({ request: createTaroRequest(API_BASE_URL) });

export type { OrderStatus, OrderStatusView };
export { availableOrderActions, orderActionLabel, toOrderStatusView };

export type OrderLoadStatus = "loading" | "success" | "error";

export interface UseOrderResult {
  order: OrderDto | undefined;
  loadState: OrderLoadStatus;
  code: string | undefined;
  traceId: string | undefined;
  reload: () => Promise<void>;
}

/**
 * Loads a single order from the server and exposes a reload trigger.
 * The returned order — and therefore every UI state derived from it — comes
 * exclusively from the server; the hook never fabricates order data.
 */
export function useOrder(orderId: string): UseOrderResult {
  const [order, setOrder] = useState<OrderDto>();
  const [loadState, setLoadState] = useState<OrderLoadStatus>("loading");
  const [code, setCode] = useState<string>();
  const [traceId, setTraceId] = useState<string>();

  const reload = useCallback(async () => {
    if (!orderId) return;
    setLoadState("loading");
    try {
      setOrder(await client.getOrder(orderId));
      setLoadState("success");
    } catch (error) {
      const appError =
        error instanceof AppError ? error : new AppError("UNKNOWN", "请求失败");
      setCode(appError.code);
      setTraceId(appError.traceId);
      setLoadState("error");
    }
  }, [orderId]);

  useEffect(() => {
    void reload();
  }, [reload]);

  return { order, loadState, code, traceId, reload };
}

/**
 * Maps a server-derived order action to its target navigation page.
 * Returns null for actions that are handled inline in the current page
 * (cancel and contactSupport) — the caller must branch on the action for
 * those.
 */
export function orderActionRoute(action: OrderAction, orderId: string): string | null {
  const id = encodeURIComponent(orderId);
  switch (action) {
    case "pay":
      return `/pages/payment/index?orderId=${id}`;
    case "openDepositDoor":
      return `/pages/store/index?orderId=${id}`;
    case "openPickupDoor":
      return `/pages/pickup/index?orderId=${id}`;
    case "settleOverdue":
      return `/pages/overdue/index?orderId=${id}`;
    case "cancel":
    case "contactSupport":
      return null;
  }
}

/** Maps a domain order status tone to the StatusPill tone vocabulary. */
export function statusPillTone(
  status: OrderStatus
): "success" | "warning" | "danger" | "neutral" {
  const tone = toOrderStatusView(status).tone;
  if (tone === "success") return "success";
  if (tone === "warning") return "warning";
  if (tone === "danger") return "danger";
  return "neutral";
}

/** Development-only payment/overdue simulator switch; never enabled in production builds. */
export function devPaymentSimulatorEnabled(): boolean {
  return (
    typeof TARO_APP_DEV_PAYMENT_SIMULATOR !== "undefined" &&
    TARO_APP_DEV_PAYMENT_SIMULATOR === "true"
  );
}

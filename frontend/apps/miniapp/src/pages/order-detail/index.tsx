import { Text, View } from "@tarojs/components";
import Taro, { getCurrentInstance } from "@tarojs/taro";
import { useCallback, useEffect, useState } from "react";
import { createTaroRequest } from "@spl/api-client/http";
import { createOrderClient, type OrderDto } from "@spl/api-client/order-client";
import {
  availableOrderActions,
  orderActionLabel,
  toOrderStatusView,
  type OrderAction,
  type OrderStatus
} from "@spl/domain-ui/order";

import { FeeSummary } from "../../components/fee-summary";
import { PrimaryButton } from "../../components/primary-button";
import { StatePanel } from "../../components/state-panel";
import { StatusPill } from "../../components/status-pill";
import { StatusTimeline } from "../../components/status-timeline";
import {
  orderActionRoute,
  statusPillTone,
  useOrder
} from "../../features/orders/use-order";
import "./index.scss";

const API_BASE_URL =
  typeof TARO_APP_API_BASE_URL !== "undefined" ? TARO_APP_API_BASE_URL : "";
const client = createOrderClient({ request: createTaroRequest(API_BASE_URL) });

export default function OrderDetailPage() {
  const orderId = getCurrentInstance().router?.params.orderId ?? "";
  const { order, loadState, code, traceId, reload } = useOrder(orderId);
  const [contactOpen, setContactOpen] = useState(false);
  const [cancelling, setCancelling] = useState(false);

  // Auto-refresh order status every 5 seconds while the order is non-terminal.
  const POLL_INTERVAL_MS = 5000;
  useEffect(() => {
    if (!order) return;
    const isTerminal =
      order.status === "ORDER_STATUS_COMPLETED" ||
      order.status === "ORDER_STATUS_CANCELLED";
    if (isTerminal) return;

    const interval = setInterval(() => {
      void reload();
    }, POLL_INTERVAL_MS);

    return () => clearInterval(interval);
  }, [order, reload]);

  const onAction = useCallback(
    async (action: OrderAction) => {
      if (action === "contactSupport") {
        setContactOpen(true);
        return;
      }
      if (action === "cancel") {
        setCancelling(true);
        try {
          await client.cancelOrder(orderId);
          await reload();
        } finally {
          setCancelling(false);
        }
        return;
      }
      const route = orderActionRoute(action, orderId);
      if (route) {
        void Taro.navigateTo({ url: route });
      }
    },
    [orderId, reload]
  );

  if (loadState === "loading") {
    return (
      <View className="page order-detail-page">
        <View className="skeleton skeleton--card" />
      </View>
    );
  }

  if (loadState === "error" || !order) {
    return (
      <View className="page order-detail-page">
        <StatePanel
          kind="network"
          {...(traceId ? { traceId } : {})}
          onRetry={() => void reload()}
        />
      </View>
    );
  }

  const statusView = toOrderStatusView(order.status as OrderStatus);
  const actions = availableOrderActions(order.status as OrderStatus);

  return (
    <View className="page order-detail-page">
      <View className="order-detail-page__status">
        <Text className="page-title">订单详情</Text>
        <StatusPill label={statusView.label} tone={statusPillTone(order.status as OrderStatus)} />
      </View>

      <View className="order-detail-page__info">
        <Text className="order-detail-page__site">{order.site_name}</Text>
        <Text className="order-detail-page__meta">
          柜格 {order.cell_no} · 订单号 {order.order_no}
        </Text>
      </View>

      <StatusTimeline status={order.status as OrderStatus} />

      <FeeSummary
        rentFeeFen={order.fee_snapshot.rent_fee_fen}
        depositFen={order.fee_snapshot.deposit_fen}
        discountFen={order.fee_snapshot.discount_fen}
        totalFen={order.fee_snapshot.total_fen}
        overdueFeeFen={order.overdue_fee_fen}
      />

      <View className="order-detail-page__actions">
        {actions.map((action) => (
          <PrimaryButton
            key={action}
            label={orderActionLabel(action)}
            loading={action === "cancel" && cancelling}
            onClick={() => void onAction(action)}
          />
        ))}
      </View>

      {contactOpen ? (
        <View className="order-detail-page__contact">
          <Text className="order-detail-page__contact-title">联系客服</Text>
          <Text className="order-detail-page__contact-hint">
            如需帮助，请通过 App 内官方渠道联系客服，并提供订单号 {order.order_no}
          </Text>
        </View>
      ) : null}
    </View>
  );
}

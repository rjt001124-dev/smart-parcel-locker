import { Text, View } from "@tarojs/components";
import { getCurrentInstance } from "@tarojs/taro";
import { useCallback, useState } from "react";
import { createTaroRequest } from "@spl/api-client/http";
import { createOrderClient } from "@spl/api-client/order-client";
import { toOrderStatusView, type OrderStatus } from "@spl/domain-ui/order";

import { FeeSummary } from "../../components/fee-summary";
import { PrimaryButton } from "../../components/primary-button";
import { StatePanel } from "../../components/state-panel";
import { StatusPill } from "../../components/status-pill";
import {
  devPaymentSimulatorEnabled,
  statusPillTone,
  useOrder
} from "../../features/orders/use-order";
import "./index.scss";

const API_BASE_URL =
  typeof TARO_APP_API_BASE_URL !== "undefined" ? TARO_APP_API_BASE_URL : "";
const client = createOrderClient({ request: createTaroRequest(API_BASE_URL) });

export default function OverduePage() {
  const orderId = getCurrentInstance().router?.params.orderId ?? "";
  const { order, loadState, traceId, reload } = useOrder(orderId);
  const [settling, setSettling] = useState(false);

  const settle = useCallback(async () => {
    if (settling) return;
    setSettling(true);
    try {
      // Overdue status and amount always come from the server reply.
      await client.settleOverdue(orderId);
      await reload();
    } catch {
      void reload();
    } finally {
      setSettling(false);
    }
  }, [orderId, settling, reload]);

  if (loadState === "loading") {
    return (
      <View className="page overdue-page">
        <View className="skeleton skeleton--card" />
      </View>
    );
  }

  if (loadState === "error" || !order) {
    return (
      <View className="page overdue-page">
        <StatePanel
          kind="network"
          {...(traceId ? { traceId } : {})}
          onRetry={() => void reload()}
        />
      </View>
    );
  }

  const statusView = toOrderStatusView(order.status as OrderStatus);
  const overdue = order.status === "ORDER_STATUS_OVERDUE";

  return (
    <View className="page overdue-page">
      <View className="overdue-page__status">
        <Text className="page-title">逾期补缴</Text>
        <StatusPill label={statusView.label} tone={statusPillTone(order.status as OrderStatus)} />
      </View>

      <Text className="overdue-page__site">
        {order.site_name} · 柜格 {order.cell_no}
      </Text>

      <FeeSummary
        rentFeeFen={order.fee_snapshot.rent_fee_fen}
        depositFen={order.fee_snapshot.deposit_fen}
        discountFen={order.fee_snapshot.discount_fen}
        totalFen={order.fee_snapshot.total_fen}
        overdueFeeFen={order.overdue_fee_fen}
      />

      {overdue ? (
        devPaymentSimulatorEnabled() ? (
          <PrimaryButton
            label="补缴逾期费（开发环境）"
            loading={settling}
            onClick={() => void settle()}
          />
        ) : (
          <Text className="overdue-page__hint">
            真实支付渠道尚未接入，暂无法补缴逾期费
          </Text>
        )
      ) : (
        <Text className="overdue-page__hint">逾期费用已结清</Text>
      )}
    </View>
  );
}

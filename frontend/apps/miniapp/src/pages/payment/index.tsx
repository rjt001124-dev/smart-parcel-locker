import { Text, View } from "@tarojs/components";
import { getCurrentInstance } from "@tarojs/taro";
import { useCallback, useEffect, useState } from "react";
import { createTaroRequest } from "@spl/api-client/http";
import { createOrderClient, type OrderDto } from "@spl/api-client/order-client";

import { FeeSummary } from "../../components/fee-summary";
import { PrimaryButton } from "../../components/primary-button";
import { StatePanel } from "../../components/state-panel";
import { StatusPill } from "../../components/status-pill";
import { devPaymentSimulatorEnabled, statusPillTone } from "../../features/orders/use-order";
import { toOrderStatusView } from "@spl/domain-ui/order";
import "./index.scss";

const API_BASE_URL =
  typeof TARO_APP_API_BASE_URL !== "undefined" ? TARO_APP_API_BASE_URL : "";
const client = createOrderClient({ request: createTaroRequest(API_BASE_URL) });

type LoadState = "loading" | "success" | "error";

export default function PaymentPage() {
  const params = getCurrentInstance().router?.params ?? {};
  const orderId = params.orderId ?? "";

  const [order, setOrder] = useState<OrderDto>();
  const [loadState, setLoadState] = useState<LoadState>("loading");
  const [paying, setPaying] = useState(false);

  const load = useCallback(async () => {
    if (!orderId) return;
    setLoadState("loading");
    try {
      setOrder(await client.getOrder(orderId));
      setLoadState("success");
    } catch {
      setLoadState("error");
    }
  }, [orderId]);

  useEffect(() => {
    void load();
  }, [load]);

  const simulatePayment = async () => {
    if (paying) return;
    setPaying(true);
    try {
      // Order status comes ONLY from the server reply; no local fabrication.
      setOrder(await client.confirmDevelopmentPayment(orderId));
    } catch {
      setLoadState("error");
    } finally {
      setPaying(false);
    }
  };

  if (loadState === "loading") {
    return (
      <View className="page payment-page">
        <View className="skeleton skeleton--card" />
      </View>
    );
  }

  if (loadState === "error" || !order) {
    return (
      <View className="page payment-page">
        <StatePanel kind="network" onRetry={() => void load()} />
      </View>
    );
  }

  const statusView = toOrderStatusView(order.status);
  const pendingPayment = order.status === "ORDER_STATUS_PENDING_PAYMENT";

  return (
    <View className="page payment-page">
      <Text className="page-title">订单支付</Text>
      <View className="payment-page__status">
        <StatusPill label={statusView.label} tone={statusPillTone(order.status)} />
        <Text className="payment-page__order-no">订单号 {order.order_no}</Text>
      </View>
      <FeeSummary
        rentFeeFen={order.fee_snapshot.rent_fee_fen}
        depositFen={order.fee_snapshot.deposit_fen}
        discountFen={order.fee_snapshot.discount_fen}
        totalFen={order.fee_snapshot.total_fen}
        overdueFeeFen={order.overdue_fee_fen}
      />
      {pendingPayment ? (
        devPaymentSimulatorEnabled() ? (
          <PrimaryButton
            label="模拟支付（开发环境）"
            loading={paying}
            onClick={() => void simulatePayment()}
          />
        ) : (
          <Text className="payment-page__hint">
            真实支付渠道尚未接入，暂无法完成支付
          </Text>
        )
      ) : (
        <Text className="payment-page__hint">
          支付成功，请前往柜机存入物品
        </Text>
      )}
    </View>
  );
}

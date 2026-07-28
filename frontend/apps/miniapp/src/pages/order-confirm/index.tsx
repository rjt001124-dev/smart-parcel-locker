import { Text, View } from "@tarojs/components";
import Taro, { getCurrentInstance } from "@tarojs/taro";
import { useCallback, useEffect, useRef, useState } from "react";
import { createTaroRequest } from "@spl/api-client/http";
import { createOrderClient, type OrderDto } from "@spl/api-client/order-client";

import { FeeSummary } from "../../components/fee-summary";
import { PrimaryButton } from "../../components/primary-button";
import { StatePanel } from "../../components/state-panel";
import { useActiveOrderStore } from "../../stores/active-order-store";
import "./index.scss";

const API_BASE_URL =
  typeof TARO_APP_API_BASE_URL !== "undefined" ? TARO_APP_API_BASE_URL : "";
const client = createOrderClient({ request: createTaroRequest(API_BASE_URL) });

const SIZE_LABELS: Record<string, string> = {
  CELL_SIZE_SMALL: "小号",
  CELL_SIZE_MEDIUM: "中号",
  CELL_SIZE_LARGE: "大号"
};

function durationLabel(minutes: number): string {
  return minutes % 60 === 0 ? `${minutes / 60}小时` : `${minutes}分钟`;
}

type LoadState = "loading" | "success" | "error";

export default function OrderConfirmPage() {
  const params = getCurrentInstance().router?.params ?? {};
  const siteId = params.siteId ?? "";
  const size = params.size ?? "CELL_SIZE_SMALL";
  const durationMinutes = Number.parseInt(params.durationMinutes ?? "60", 10);

  const [order, setOrder] = useState<OrderDto>();
  const [loadState, setLoadState] = useState<LoadState>("loading");
  const setActiveOrder = useActiveOrderStore((state) => state.setActiveOrder);
  const creating = useRef(false);

  const createOrder = useCallback(async () => {
    if (creating.current) return;
    creating.current = true;
    setLoadState("loading");
    try {
      const created = await client.createOrder({ siteId, size, durationMinutes });
      setOrder(created);
      setActiveOrder(created.id);
      setLoadState("success");
    } catch {
      setLoadState("error");
    } finally {
      creating.current = false;
    }
  }, [siteId, size, durationMinutes, setActiveOrder]);

  useEffect(() => {
    void createOrder();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  if (loadState === "loading") {
    return (
      <View className="page order-confirm-page">
        <View className="skeleton skeleton--card" />
      </View>
    );
  }

  if (loadState === "error" || !order) {
    return (
      <View className="page order-confirm-page">
        <StatePanel kind="network" onRetry={() => void createOrder()} />
      </View>
    );
  }

  return (
    <View className="page order-confirm-page">
      <Text className="page-title">{order.site_name}</Text>
      <Text className="page-subtitle">
        订单号 {order.order_no}
      </Text>
      <View className="order-confirm-page__meta">
        <Text>柜格 {order.cell_no}（{SIZE_LABELS[order.size] ?? order.size}）</Text>
        <Text>寄存时长 {durationLabel(order.duration_minutes)}</Text>
      </View>
      <FeeSummary
        rentFeeFen={order.fee_snapshot.rent_fee_fen}
        depositFen={order.fee_snapshot.deposit_fen}
        discountFen={order.fee_snapshot.discount_fen}
        totalFen={order.fee_snapshot.total_fen}
      />
      <PrimaryButton
        label="去支付"
        onClick={() =>
          void Taro.navigateTo({
            url: `/pages/payment/index?orderId=${encodeURIComponent(order.id)}`
          })
        }
      />
    </View>
  );
}

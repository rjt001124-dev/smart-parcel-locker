import { Text, View } from "@tarojs/components";
import Taro, { getCurrentInstance } from "@tarojs/taro";
import { useCallback, useState } from "react";
import { createTaroRequest } from "@spl/api-client/http";
import { createOrderClient, type OrderDto } from "@spl/api-client/order-client";
import { toOrderStatusView, type OrderStatus } from "@spl/domain-ui/order";

import { DoorSafetyGuard, type DoorPhase } from "../../components/door-safety-guard";
import { FeeSummary } from "../../components/fee-summary";
import { PrimaryButton } from "../../components/primary-button";
import { StatePanel } from "../../components/state-panel";
import { StatusPill } from "../../components/status-pill";
import { statusPillTone, useOrder } from "../../features/orders/use-order";
import "./index.scss";

const API_BASE_URL =
  typeof TARO_APP_API_BASE_URL !== "undefined" ? TARO_APP_API_BASE_URL : "";
const client = createOrderClient({ request: createTaroRequest(API_BASE_URL) });

export default function StorePage() {
  const orderId = getCurrentInstance().router?.params.orderId ?? "";
  const { order, loadState, traceId, reload } = useOrder(orderId);
  const [phase, setPhase] = useState<DoorPhase>("idle");

  const open = useCallback(async () => {
    if (phase === "opening") return;
    setPhase("opening");
    try {
      // Door "opened" is driven ONLY by the server reply — never fabricated.
      await client.openDepositDoor(orderId);
      setPhase("opened");
      void reload();
    } catch {
      setPhase("failed");
    }
  }, [orderId, phase, reload]);

  if (loadState === "loading") {
    return (
      <View className="page store-page">
        <View className="skeleton skeleton--card" />
      </View>
    );
  }

  if (loadState === "error" || !order) {
    return (
      <View className="page store-page">
        <StatePanel
          kind="network"
          {...(traceId ? { traceId } : {})}
          onRetry={() => void reload()}
        />
      </View>
    );
  }

  const statusView = toOrderStatusView(order.status as OrderStatus);

  return (
    <View className="page store-page">
      <View className="store-page__status">
        <Text className="page-title">存入物品</Text>
        <StatusPill label={statusView.label} tone={statusPillTone(order.status as OrderStatus)} />
      </View>

      <View className="store-page__info">
        <Text className="store-page__site">{order.site_name}</Text>
        <Text className="store-page__cell">柜格 {order.cell_no}</Text>
      </View>

      <FeeSummary
        rentFeeFen={order.fee_snapshot.rent_fee_fen}
        depositFen={order.fee_snapshot.deposit_fen}
        discountFen={order.fee_snapshot.discount_fen}
        totalFen={order.fee_snapshot.total_fen}
        overdueFeeFen={order.overdue_fee_fen}
      />

      {phase === "opened" ? (
        <View className="store-page__done">
          <Text className="store-page__done-message">
            柜门已打开，请放入物品并关闭柜门
          </Text>
          <PrimaryButton
            label="查看订单"
            onClick={() =>
              void Taro.navigateTo({
                url: `/pages/order-detail/index?orderId=${encodeURIComponent(orderId)}`
              })
            }
          />
        </View>
      ) : (
        <DoorSafetyGuard
          phase={phase}
          actionLabel="开门存入"
          cellNo={order.cell_no}
          onAction={() => void open()}
        />
      )}
    </View>
  );
}

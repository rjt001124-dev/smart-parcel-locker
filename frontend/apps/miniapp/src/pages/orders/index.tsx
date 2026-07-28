import { Text, View } from "@tarojs/components";
import Taro from "@tarojs/taro";
import { useCallback, useEffect, useState } from "react";
import { AppError, createTaroRequest } from "@spl/api-client/http";
import { createOrderClient, type OrderDto } from "@spl/api-client/order-client";
import { toOrderStatusView, type OrderStatus } from "@spl/domain-ui/order";

import { StatePanel } from "../../components/state-panel";
import { StatusPill } from "../../components/status-pill";
import { statusPillTone } from "../../features/orders/use-order";
import "./index.scss";

const API_BASE_URL =
  typeof TARO_APP_API_BASE_URL !== "undefined" ? TARO_APP_API_BASE_URL : "";
const client = createOrderClient({ request: createTaroRequest(API_BASE_URL) });

type OrdersState =
  | { status: "loading"; orders: OrderDto[] }
  | { status: "success"; orders: OrderDto[] }
  | { status: "error"; orders: OrderDto[]; code?: string; traceId?: string };

export default function OrdersPage() {
  const [state, setState] = useState<OrdersState>({ status: "loading", orders: [] });

  const load = useCallback(async () => {
    setState({ status: "loading", orders: [] });
    try {
      const result = await client.listOrders();
      setState({ status: "success", orders: result.orders });
    } catch (error) {
      const appError =
        error instanceof AppError ? error : new AppError("UNKNOWN", "请求失败");
      setState({
        status: "error",
        orders: [],
        ...(appError.traceId ? { traceId: appError.traceId } : {}),
        code: appError.code
      });
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  if (state.status === "loading") {
    return (
      <View className="page orders-page">
        <View className="skeleton skeleton--card" />
      </View>
    );
  }

  if (state.status === "error") {
    return (
      <View className="page orders-page">
        <StatePanel
          kind="network"
          {...(state.traceId ? { traceId: state.traceId } : {})}
          onRetry={() => void load()}
        />
      </View>
    );
  }

  if (state.orders.length === 0) {
    return (
      <View className="page orders-page">
        <Text className="page-title">我的订单</Text>
        <View className="orders-page__empty">
          <Text className="orders-page__empty-title">暂无订单</Text>
          <Text className="orders-page__empty-hint">前往附近网点开始寄存</Text>
        </View>
      </View>
    );
  }

  return (
    <View className="page orders-page">
      <Text className="page-title">我的订单</Text>
      {state.orders.map((order) => {
        const view = toOrderStatusView(order.status as OrderStatus);
        return (
          <View
            key={order.id}
            className="order-row"
            onClick={() =>
              void Taro.navigateTo({
                url: `/pages/order-detail/index?orderId=${encodeURIComponent(order.id)}`
              })
            }
          >
            <View className="order-row__head">
              <Text className="order-row__site">{order.site_name}</Text>
              <StatusPill label={view.label} tone={statusPillTone(order.status as OrderStatus)} />
            </View>
            <Text className="order-row__meta">
              柜格 {order.cell_no} · 订单号 {order.order_no}
            </Text>
          </View>
        );
      })}
    </View>
  );
}

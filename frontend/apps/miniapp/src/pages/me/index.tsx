import { Text, View } from "@tarojs/components";
import Taro from "@tarojs/taro";
import { useCallback, useEffect, useState } from "react";
import { AppError, createTaroRequest } from "@spl/api-client/http";
import { createOrderClient } from "@spl/api-client/order-client";

import { useSessionStore } from "../../stores/session-store";
import { devPaymentSimulatorEnabled } from "../../features/orders/use-order";
import "./index.scss";

const API_BASE_URL =
  typeof TARO_APP_API_BASE_URL !== "undefined" ? TARO_APP_API_BASE_URL : "";
const client = createOrderClient({ request: createTaroRequest(API_BASE_URL) });

export default function MePage() {
  const userId = useSessionStore((s) => s.userId);
  const hydrate = useSessionStore((s) => s.hydrate);
  const [orderCount, setOrderCount] = useState<number | undefined>();

  useEffect(() => {
    hydrate();
  }, [hydrate]);

  const loadCount = useCallback(async () => {
    try {
      const result = await client.listOrders();
      setOrderCount(result.orders.length);
    } catch {
      setOrderCount(0);
    }
  }, []);

  useEffect(() => {
    void loadCount();
  }, [loadCount]);

  const simEnabled = devPaymentSimulatorEnabled();

  return (
    <View className="page me-page">
      <View className="me-page__header">
        <View className="me-page__avatar">
          <Text className="me-page__avatar-text">{userId ? userId.charAt(0).toUpperCase() : "U"}</Text>
        </View>
        <View className="me-page__user">
          <Text className="me-page__user-id">{userId ?? "未登录"}</Text>
          <Text className="me-page__app-name">智能快递柜</Text>
        </View>
      </View>

      {simEnabled && (
        <View className="me-page__dev-banner">
          <Text>开发模拟器已开启 — 支付与逾期操作为模拟行为</Text>
        </View>
      )}

      <View
        className="me-page__shortcut"
        onClick={() => Taro.navigateTo({ url: "/pages/orders/index" })}
      >
        <View className="me-page__shortcut-left">
          <Text className="me-page__shortcut-count">{orderCount ?? "—"}</Text>
          <Text className="me-page__shortcut-label">我的订单</Text>
        </View>
        <Text className="me-page__shortcut-arrow">›</Text>
      </View>

      <Text className="me-page__section-title">帮助与反馈</Text>
      <View className="me-page__help-list">
        <View className="me-page__help-item">
          <Text className="me-page__help-q">如何存取物品？</Text>
          <Text className="me-page__help-a">
            选择附近网点 → 选柜格 → 支付 → 存入物品并关门 → 取件时输入取件码开门
          </Text>
        </View>
        <View className="me-page__help-item">
          <Text className="me-page__help-q">逾期费怎么算？</Text>
          <Text className="me-page__help-a">
            超过寄存时长后按小时计费，可在订单详情页补缴逾期费后取件
          </Text>
        </View>
        <View className="me-page__help-item">
          <Text className="me-page__help-q">遇到问题怎么办？</Text>
          <Text className="me-page__help-a">
            在订单详情页点击「联系客服」，提供订单号即可获得帮助
          </Text>
        </View>
      </View>

      <Text className="me-page__version">版本 1.0.0 · 开发环境</Text>
    </View>
  );
}

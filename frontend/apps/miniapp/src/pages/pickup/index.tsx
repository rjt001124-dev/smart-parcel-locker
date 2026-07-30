import { Input, Text, View } from "@tarojs/components";
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
import { scanCode } from "../../features/scan/scan-code";
import "./index.scss";

const API_BASE_URL =
  typeof TARO_APP_API_BASE_URL !== "undefined" ? TARO_APP_API_BASE_URL : "";
const client = createOrderClient({ request: createTaroRequest(API_BASE_URL) });

export default function PickupPage() {
  const orderId = getCurrentInstance().router?.params.orderId ?? "";
  const { order, loadState, traceId, reload } = useOrder(orderId);
  const [phase, setPhase] = useState<DoorPhase>("idle");
  const [pickupCode, setPickupCode] = useState("");
  const [codeError, setCodeError] = useState(false);

  const codeMatches = order != null && pickupCode.trim() === order.cell_no;

  // Apply a pickup-code candidate (from typing or scanning) and validate it
  // against the server order — never fabricates a match.
  const applyCode = useCallback(
    (raw: string) => {
      const value = raw.trim();
      setPickupCode(value);
      if (value.length === 0) {
        setCodeError(false);
      } else if (order && value !== order.cell_no) {
        setCodeError(true);
      } else {
        setCodeError(false);
      }
    },
    [order]
  );

  const onCodeInput = useCallback(
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (e: any) => {
      const detail = e?.detail as { value?: string } | undefined;
      const target = e?.target as { value?: string } | undefined;
      applyCode(detail?.value ?? target?.value ?? "");
    },
    [applyCode]
  );

  // Scan only fills the pickup-code field; the server still validates it and
  // the door-safety acknowledgement still governs the actual open.
  const onScan = useCallback(async () => {
    if (!order) return;
    const result = await scanCode();
    if (result.ok) {
      applyCode(result.value);
    }
    // A cancelled or unsupported scan is ignored; the user can still type.
  }, [order, applyCode]);

  const verifyCode = useCallback(() => {
    if (!order) return;
    if (pickupCode.trim() !== order.cell_no) {
      setCodeError(true);
      return;
    }
    setCodeError(false);
  }, [order, pickupCode]);

  const open = useCallback(async () => {
    if (phase === "opening") return;
    if (!codeMatches) {
      setCodeError(true);
      return;
    }
    setPhase("opening");
    try {
      // Door "opened" is driven ONLY by the server reply — never fabricated.
      await client.openPickupDoor(orderId);
      setPhase("opened");
      void reload();
    } catch {
      setPhase("failed");
    }
  }, [orderId, phase, reload, codeMatches]);

  if (loadState === "loading") {
    return (
      <View className="page pickup-page">
        <View className="skeleton skeleton--card" />
      </View>
    );
  }

  if (loadState === "error" || !order) {
    return (
      <View className="page pickup-page">
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
    <View className="page pickup-page">
      <View className="pickup-page__status">
        <Text className="page-title">取出物品</Text>
        <StatusPill label={statusView.label} tone={statusPillTone(order.status as OrderStatus)} />
      </View>

      <View className="pickup-page__info">
        <Text className="pickup-page__site">{order.site_name}</Text>
        <Text className="pickup-page__cell">柜格 {order.cell_no}</Text>
      </View>

      <FeeSummary
        rentFeeFen={order.fee_snapshot.rent_fee_fen}
        depositFen={order.fee_snapshot.deposit_fen}
        discountFen={order.fee_snapshot.discount_fen}
        totalFen={order.fee_snapshot.total_fen}
        overdueFeeFen={order.overdue_fee_fen}
      />

      {phase === "opened" ? (
        <View className="pickup-page__done">
          <Text className="pickup-page__done-message">
            柜门已打开，请取走物品并关闭柜门
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
        <View className="pickup-page__code-section">
          <Text className="pickup-page__code-hint">
            请输入柜格号 {order.cell_no} 确认取件
          </Text>
          <View className="pickup-page__scan-row">
            <View
              className="pickup-page__scan-btn"
              onClick={() => void onScan()}
            >
              <Text>扫码取件</Text>
            </View>
          </View>
          <Input
            className="pickup-page__code-input"
            placeholder="请输入柜格号"
            value={pickupCode}
            onInput={onCodeInput}
            onBlur={verifyCode}
          />
          {codeError && (
            <Text className="pickup-page__code-error">柜格号不匹配</Text>
          )}
          {codeMatches ? (
            <DoorSafetyGuard
              phase={phase}
              actionLabel="开门取件"
              cellNo={order.cell_no}
              onAction={() => void open()}
            />
          ) : null}
        </View>
      )}
    </View>
  );
}

import { Text, View } from "@tarojs/components";
import { useState } from "react";

import { DoorStatusPanel, type DoorPhase } from "./door-status-panel";
import { PrimaryButton } from "./primary-button";
import "./door-safety-guard.scss";

export type { DoorPhase } from "./door-status-panel";

export interface DoorSafetyGuardProps {
  phase: DoorPhase;
  actionLabel: string;
  onAction: () => void;
  cellNo?: string;
  message?: string;
}

const SAFETY_ITEMS = [
  "柜门周边无人窥视",
  "取物 / 存物通道畅通",
  "已核对目标柜格号"
];

/**
 * Wraps the door-open action with a mandatory safety acknowledgement.
 * The door itself still opens ONLY through the server-driven `onAction`
 * call — this guard never fabricates door success, it just refuses to
 * trigger the call until the operator confirms the surroundings are safe.
 */
export function DoorSafetyGuard({
  phase,
  actionLabel,
  onAction,
  cellNo,
  message
}: DoorSafetyGuardProps) {
  const [acknowledged, setAcknowledged] = useState(false);
  const [hint, setHint] = useState<string | null>(null);

  const showSafety = phase === "idle" || phase === "failed";

  const handleAction = () => {
    if (!acknowledged) {
      setHint("请先勾选安全确认");
      return;
    }
    setHint(null);
    onAction();
  };

  return (
    <View className="door-safety-guard">
      {showSafety ? (
        <View className="door-safety-guard__safety">
          <Text className="door-safety-guard__safety-title">开门前安全确认</Text>
          <View className="door-safety-guard__list">
            {SAFETY_ITEMS.map((item, index) => (
              <Text key={index} className="door-safety-guard__item">
                · {item}
              </Text>
            ))}
          </View>
          {cellNo ? (
            <Text className="door-safety-guard__cell">目标柜格：{cellNo}</Text>
          ) : null}
          <View
            className="door-safety-guard__ack"
            onClick={() => setAcknowledged((value) => !value)}
          >
            <View
              className={
                "door-safety-guard__checkbox" +
                (acknowledged ? " door-safety-guard__checkbox--checked" : "")
              }
            />
            <Text className="door-safety-guard__ack-label">
              我已知晓并确认安全
            </Text>
          </View>
        </View>
      ) : null}

      {hint && showSafety ? (
        <Text className="door-safety-guard__hint">{hint}</Text>
      ) : null}

      {phase === "idle" || phase === "failed" ? (
        <PrimaryButton
          label={phase === "failed" ? "重试" : actionLabel}
          onClick={handleAction}
        />
      ) : (
        <DoorStatusPanel
          phase={phase}
          actionLabel={actionLabel}
          onAction={handleAction}
          {...(message ? { message } : {})}
        />
      )}
    </View>
  );
}

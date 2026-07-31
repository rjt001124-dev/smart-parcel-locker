import { Text, View } from "@tarojs/components";

import { PrimaryButton } from "./primary-button";
import "./door-status-panel.scss";

export type DoorPhase = "idle" | "opening" | "opened" | "failed";

export interface DoorStatusPanelProps {
  phase: DoorPhase;
  actionLabel: string;
  onAction: () => void;
  message?: string;
}

/**
 * Door interaction panel. The phase is driven ONLY by server responses:
 * "opened" must come from a successful door-open API result — the frontend
 * never fabricates door success.
 */
export function DoorStatusPanel({
  phase,
  actionLabel,
  onAction,
  message
}: DoorStatusPanelProps) {
  return (
    <View className={`door-status-panel door-status-panel--${phase}`}>
      {phase === "opening" ? (
        <Text className="door-status-panel__message">正在开门…</Text>
      ) : null}
      {message && phase !== "opening" ? (
        <Text className="door-status-panel__message">{message}</Text>
      ) : null}
      {phase === "idle" ? (
        <PrimaryButton label={actionLabel} onClick={onAction} />
      ) : null}
      {phase === "failed" ? (
        <PrimaryButton label="重试" onClick={onAction} />
      ) : null}
    </View>
  );
}

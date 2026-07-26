import { Text } from "@tarojs/components";

export interface StatusPillProps {
  label: string;
  tone: "success" | "warning" | "danger" | "neutral";
}

export function StatusPill({ label, tone }: StatusPillProps) {
  return <Text className={`status-pill status-pill--${tone}`}>{label}</Text>;
}

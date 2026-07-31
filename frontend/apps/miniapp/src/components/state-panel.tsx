import { Button, Text, View } from "@tarojs/components";
import "./state-panel.scss";

const copy = {
  empty: {
    title: "暂无可用网点",
    impact: "当前筛选条件下无法开始寄存",
    action: "调整筛选"
  },
  network: {
    title: "网络连接失败",
    impact: "尚未更改当前订单或柜格",
    action: "重试"
  },
  offline: {
    title: "设备已离线",
    impact: "当前站点暂时无法开门",
    action: "查看附近网点"
  },
  location: {
    title: "需要位置权限",
    impact: "授权定位后才能查找附近寄存点",
    action: "重新定位"
  }
} as const;

export function StatePanel(props: {
  kind: keyof typeof copy;
  traceId?: string;
  onRetry: () => void;
}) {
  const value = copy[props.kind];
  return (
    <View className={`state-panel state-panel--${props.kind}`}>
      <Text className="state-panel__title">{value.title}</Text>
      <Text className="state-panel__impact">{value.impact}</Text>
      {props.traceId && (
        <Text className="state-panel__trace">参考编号：{props.traceId}</Text>
      )}
      <Button onClick={props.onRetry}>{value.action}</Button>
    </View>
  );
}

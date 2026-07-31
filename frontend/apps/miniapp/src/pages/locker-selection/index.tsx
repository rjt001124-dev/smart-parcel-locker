import { Text, View } from "@tarojs/components";
import Taro, { getCurrentInstance } from "@tarojs/taro";
import { useState } from "react";

import { PrimaryButton } from "../../components/primary-button";
import "./index.scss";

const SIZE_OPTIONS = [
  { value: "CELL_SIZE_SMALL", label: "小号" },
  { value: "CELL_SIZE_MEDIUM", label: "中号" },
  { value: "CELL_SIZE_LARGE", label: "大号" }
] as const;

const DURATION_OPTIONS = [
  { minutes: 60, label: "1小时" },
  { minutes: 120, label: "2小时" },
  { minutes: 240, label: "4小时" },
  { minutes: 1440, label: "24小时" }
] as const;

export default function LockerSelectionPage() {
  const params = getCurrentInstance().router?.params ?? {};
  const siteId = params.siteId ?? "";
  const siteName = decodeURIComponent(params.siteName ?? "");
  const [size, setSize] = useState<string>("CELL_SIZE_SMALL");
  const [durationMinutes, setDurationMinutes] = useState<number>(60);

  const next = () => {
    void Taro.navigateTo({
      url:
        `/pages/order-confirm/index?siteId=${encodeURIComponent(siteId)}` +
        `&size=${size}&durationMinutes=${durationMinutes}`
    });
  };

  return (
    <View className="page locker-selection-page">
      <Text className="page-title">{siteName || "选择柜格"}</Text>
      <Text className="page-subtitle">选择柜格大小与寄存时长</Text>

      <Text className="locker-selection-page__section">柜格大小</Text>
      <View className="locker-selection-page__options">
        {SIZE_OPTIONS.map((option) => (
          <View
            key={option.value}
            className={`locker-selection-page__option${
              size === option.value ? " locker-selection-page__option--active" : ""
            }`}
            onClick={() => setSize(option.value)}
          >
            <Text>{option.label}</Text>
          </View>
        ))}
      </View>

      <Text className="locker-selection-page__section">寄存时长</Text>
      <View className="locker-selection-page__options">
        {DURATION_OPTIONS.map((option) => (
          <View
            key={option.minutes}
            className={`locker-selection-page__option${
              durationMinutes === option.minutes
                ? " locker-selection-page__option--active"
                : ""
            }`}
            onClick={() => setDurationMinutes(option.minutes)}
          >
            <Text>{option.label}</Text>
          </View>
        ))}
      </View>

      <Text className="locker-selection-page__hint">
        费用以下单后服务端返回的金额为准
      </Text>
      <PrimaryButton label="下一步" onClick={next} />
    </View>
  );
}

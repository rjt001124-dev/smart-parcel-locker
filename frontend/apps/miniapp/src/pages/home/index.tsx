import { Input, Text, View } from "@tarojs/components";
import Taro from "@tarojs/taro";
import { useEffect, useState } from "react";
import { toOrderStatusView, type OrderStatus } from "@spl/domain-ui/order";

import { SiteCard } from "../../features/sites/site-card";
import { useSites } from "../../features/sites/use-sites";
import { useLocationStore } from "../../stores/location-store";
import { useActiveOrderStore } from "../../stores/active-order-store";
import { statusPillTone, useOrder } from "../../features/orders/use-order";
import { StatePanel } from "../../components/state-panel";
import { StatusPill } from "../../components/status-pill";
import "./index.scss";

export default function HomePage() {
  const cityName = useLocationStore((state) => state.cityName);
  const sites = useSites();
  const [keyword, setKeyword] = useState("");

  const activeOrderId = useActiveOrderStore((state) => state.activeOrderId);
  const hydrateActiveOrder = useActiveOrderStore((state) => state.hydrate);
  const clearActiveOrder = useActiveOrderStore((state) => state.clearActiveOrder);
  const { order: activeOrder, loadState: activeLoadState } = useOrder(activeOrderId ?? "");

  useEffect(() => {
    hydrateActiveOrder();
  }, [hydrateActiveOrder]);

  useEffect(() => {
    if (
      activeOrder &&
      (activeOrder.status === "ORDER_STATUS_COMPLETED" ||
        activeOrder.status === "ORDER_STATUS_CANCELLED")
    ) {
      clearActiveOrder();
    }
  }, [activeOrder, clearActiveOrder]);

  const renderActiveOrder = () => {
    if (!activeOrderId) {
      return (
        <View className="current-order current-order--empty">
          <Text className="current-order__label">当前订单</Text>
          <Text className="current-order__title">暂无进行中的订单</Text>
          <Text className="current-order__hint">选择附近网点开始寄存</Text>
        </View>
      );
    }

    if (activeLoadState === "loading" || !activeOrder) {
      return (
        <View className="current-order">
          <View className="skeleton skeleton--title" />
        </View>
      );
    }

    return (
      <View
        className="current-order"
        onClick={() =>
          void Taro.navigateTo({
            url: `/pages/order-detail/index?orderId=${encodeURIComponent(activeOrderId)}`
          })
        }
      >
        <Text className="current-order__label">当前订单</Text>
        <Text className="current-order__title">
          {activeOrder.site_name} · 柜格 {activeOrder.cell_no}
        </Text>
        <View className="current-order__status">
          <StatusPill
            label={toOrderStatusView(activeOrder.status as OrderStatus).label}
            tone={statusPillTone(activeOrder.status as OrderStatus)}
          />
          <Text className="current-order__hint">点击查看</Text>
        </View>
      </View>
    );
  };

  if (sites.status === "loading") {
    return (
      <View className="page">
        <View className="skeleton skeleton--title" />
        <View className="skeleton skeleton--card" />
      </View>
    );
  }

  if (sites.status === "error") {
    return (
      <View className="page">
        <StatePanel
          kind={sites.errorCode === "LOCATION_UNAVAILABLE"
            ? "location"
            : sites.errorCode === "DEVICE_OFFLINE"
              ? "offline"
              : "network"}
          {...(sites.traceId ? { traceId: sites.traceId } : {})}
          onRetry={() => void sites.retry()}
        />
      </View>
    );
  }

  if (sites.sites.length === 0) {
    return (
      <View className="page">
        <StatePanel kind="empty" onRetry={() => void sites.retry()} />
      </View>
    );
  }

  const filteredSites = keyword
    ? sites.sites.filter(
        (s) =>
          s.name.includes(keyword) || s.address.includes(keyword)
      )
    : sites.sites.slice(0, 2);

  const renderSearchResults = () => {
    if (keyword && filteredSites.length === 0) {
      return (
        <View className="home-page__no-results">
          <Text>未找到匹配的网点</Text>
        </View>
      );
    }
    return filteredSites.map((site) => (
      <SiteCard
        key={site.id}
        site={site}
        onSelect={(id) => Taro.navigateTo({
          url: `/pages/site-detail/index?id=${encodeURIComponent(id)}`
        })}
      />
    ));
  };

  return (
    <View className="page home-page">
      <Text
        className="home-page__city"
        onClick={() => Taro.navigateTo({ url: "/pages/location/index" })}
      >
        {cityName} · 定位成功
      </Text>
      <Input
        className="home-page__search"
        placeholder="搜索商场、地铁站或地址"
        value={keyword}
        onInput={(e) => {
          const detail = (e as unknown as { detail?: { value?: string } }).detail;
          const target = e as unknown as React.ChangeEvent<HTMLInputElement>;
          setKeyword((detail?.value ?? target.target?.value ?? "").trim());
        }}
      />
      {renderActiveOrder()}
      <View className="section-heading">
        <Text>{keyword ? "搜索结果" : "附近寄存点"}</Text>
        <Text onClick={() => Taro.navigateTo({ url: "/pages/sites/index" })}>查看全部</Text>
      </View>
      {renderSearchResults()}
    </View>
  );
}

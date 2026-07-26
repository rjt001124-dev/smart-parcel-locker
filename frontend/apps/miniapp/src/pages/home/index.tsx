import { Input, Text, View } from "@tarojs/components";
import Taro from "@tarojs/taro";
import { SiteCard } from "../../features/sites/site-card";
import { useSites } from "../../features/sites/use-sites";
import { useLocationStore } from "../../stores/location-store";
import "./index.scss";

export default function HomePage() {
  const cityName = useLocationStore((state) => state.cityName);
  const sites = useSites();

  return (
    <View className="page home-page">
      <Text
        className="home-page__city"
        onClick={() => Taro.navigateTo({ url: "/pages/location/index" })}
      >
        {cityName} · 定位成功
      </Text>
      <Input className="home-page__search" placeholder="搜索商场、地铁站或地址" />
      <View className="current-order current-order--empty">
        <Text className="current-order__label">当前订单</Text>
        <Text className="current-order__title">暂无进行中的订单</Text>
        <Text className="current-order__hint">选择附近网点开始寄存</Text>
      </View>
      <View className="section-heading">
        <Text>附近寄存点</Text>
        <Text onClick={() => Taro.navigateTo({ url: "/pages/sites/index" })}>查看全部</Text>
      </View>
      {sites.sites.slice(0, 2).map((site) => (
        <SiteCard
          key={site.id}
          site={site}
          onSelect={(id) => Taro.navigateTo({
            url: `/pages/site-detail/index?id=${encodeURIComponent(id)}`
          })}
        />
      ))}
    </View>
  );
}

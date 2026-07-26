import { Text, View } from "@tarojs/components";
import Taro from "@tarojs/taro";
import { SiteCard } from "../../features/sites/site-card";
import { useSites } from "../../features/sites/use-sites";
import "./index.scss";

export default function SitesPage() {
  const result = useSites();

  return (
    <View className="page sites-page">
      <Text className="page-title">附近寄存点</Text>
      <Text className="page-subtitle">共找到{result.sites.length}个站点</Text>
      <View className="sites-page__list">
        {result.sites.map((site) => (
          <SiteCard
            key={site.id}
            site={site}
            onSelect={(id) => Taro.navigateTo({
              url: `/pages/site-detail/index?id=${encodeURIComponent(id)}`
            })}
          />
        ))}
      </View>
    </View>
  );
}

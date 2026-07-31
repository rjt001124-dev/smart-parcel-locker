import { Text, View } from "@tarojs/components";
import Taro from "@tarojs/taro";
import { StatePanel } from "../../components/state-panel";
import { SiteCard } from "../../features/sites/site-card";
import { useSites } from "../../features/sites/use-sites";
import "./index.scss";

export default function SitesPage() {
  const result = useSites();

  if (result.status === "loading") {
    return (
      <View className="page sites-page">
        <View className="skeleton skeleton--title" />
        <View className="skeleton skeleton--card" />
      </View>
    );
  }

  if (result.status === "error") {
    return (
      <View className="page sites-page">
        <StatePanel
          kind={result.errorCode === "LOCATION_UNAVAILABLE"
            ? "location"
            : result.errorCode === "DEVICE_OFFLINE"
              ? "offline"
              : "network"}
          {...(result.traceId ? { traceId: result.traceId } : {})}
          onRetry={() => void result.retry()}
        />
      </View>
    );
  }

  if (result.sites.length === 0) {
    return (
      <View className="page sites-page">
        <StatePanel kind="empty" onRetry={() => void result.retry()} />
      </View>
    );
  }

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

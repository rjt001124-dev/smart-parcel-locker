import { Text, View } from "@tarojs/components";
import type { SiteCardView } from "@spl/domain-ui/site";
import { StatusPill } from "../../components/status-pill";
import "./site-card.scss";

export interface SiteCardProps {
  site: SiteCardView;
  onSelect: (id: string) => void;
}

export function SiteCard({ site, onSelect }: SiteCardProps) {
  return (
    <View className="site-card" onClick={() => onSelect(site.id)}>
      <View className="site-card__header">
        <Text className="site-card__name">{site.name}</Text>
        <StatusPill tone={site.statusTone} label={site.statusLabel} />
      </View>
      <Text className="site-card__meta">
        {site.distanceLabel} · {site.availabilityLabel}
      </Text>
      <Text className="site-card__address">{site.address}</Text>
    </View>
  );
}

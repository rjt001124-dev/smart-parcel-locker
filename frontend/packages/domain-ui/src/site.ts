import type { SiteSummaryDto } from "@spl/api-client/site-client";

export interface SiteCardView {
  id: string;
  name: string;
  address: string;
  distanceLabel: string;
  availabilityLabel: string;
  statusLabel: string;
  statusTone: "success" | "warning" | "danger";
}

export function toSiteCardView(site: SiteSummaryDto): SiteCardView {
  const distanceLabel = site.distance_m === undefined
    ? "距离未知"
    : site.distance_m < 1000
      ? `${site.distance_m}m`
      : `${(site.distance_m / 1000).toFixed(1)}km`;
  const availableCells = site.availability.reduce(
    (sum, item) => sum + item.available_count,
    0
  );

  return {
    id: site.id,
    name: site.name,
    address: site.address,
    distanceLabel,
    availabilityLabel: `可用${availableCells}格`,
    statusLabel: availableCells > 0 ? "有空柜" : "暂无空柜",
    statusTone: availableCells > 0 ? "success" : "warning"
  };
}

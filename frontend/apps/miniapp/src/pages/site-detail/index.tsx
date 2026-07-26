import { Text, View } from "@tarojs/components";
import Taro, { getCurrentInstance } from "@tarojs/taro";
import { useEffect, useMemo, useState } from "react";
import { createTaroRequest } from "@spl/api-client/http";
import {
  createSiteClient,
  type CellDto,
  type SiteDetailDto
} from "@spl/api-client/site-client";
import { PrimaryButton } from "../../components/primary-button";
import "./index.scss";

const client = createSiteClient({ request: createTaroRequest(TARO_APP_API_BASE_URL) });

export default function SiteDetailPage() {
  const siteId = getCurrentInstance().router?.params.id ?? "";
  const [site, setSite] = useState<SiteDetailDto>();
  const [cells, setCells] = useState<CellDto[]>([]);

  useEffect(() => {
    if (!siteId) return;
    void Promise.all([client.getSite(siteId), client.listCells(siteId)]).then(
      ([siteResult, cellResult]) => {
        setSite(siteResult);
        setCells(cellResult.cells);
      }
    );
  }, [siteId]);

  const counts = useMemo(() => ({
    small: cells.filter((cell) =>
      cell.size === "CELL_SIZE_SMALL" && cell.status === "CELL_STATUS_IDLE"
    ).length,
    medium: cells.filter((cell) =>
      cell.size === "CELL_SIZE_MEDIUM" && cell.status === "CELL_STATUS_IDLE"
    ).length,
    large: cells.filter((cell) =>
      cell.size === "CELL_SIZE_LARGE" && cell.status === "CELL_STATUS_IDLE"
    ).length
  }), [cells]);

  return (
    <View className="page site-detail-page">
      <Text className="page-title">{site?.name ?? "正在加载网点"}</Text>
      <Text className="page-subtitle">{site?.address ?? ""}</Text>
      <View className="locker-counts">
        <View className="locker-count"><Text>小号</Text><Text>{counts.small}</Text></View>
        <View className="locker-count"><Text>中号</Text><Text>{counts.medium}</Text></View>
        <View className="locker-count"><Text>大号</Text><Text>{counts.large}</Text></View>
      </View>
      <PrimaryButton
        label="选择柜格"
        onClick={() => void Taro.showToast({
          title: "柜格预约将在下一阶段开放",
          icon: "none"
        })}
      />
    </View>
  );
}

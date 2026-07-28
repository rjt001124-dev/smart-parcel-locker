import { Text, View } from "@tarojs/components";
import Taro, { getCurrentInstance } from "@tarojs/taro";
import { useCallback, useEffect, useMemo, useState } from "react";
import { AppError, createTaroRequest } from "@spl/api-client/http";
import {
  createSiteClient,
  type CellDto,
  type SiteDetailDto
} from "@spl/api-client/site-client";
import { PrimaryButton } from "../../components/primary-button";
import { StatePanel } from "../../components/state-panel";
import "./index.scss";

const client = createSiteClient({ request: createTaroRequest(TARO_APP_API_BASE_URL) });

type DetailLoadState =
  | { status: "loading" | "success" }
  | { status: "network" | "offline"; traceId?: string };

export default function SiteDetailPage() {
  const siteId = getCurrentInstance().router?.params.id ?? "";
  const [site, setSite] = useState<SiteDetailDto>();
  const [cells, setCells] = useState<CellDto[]>([]);
  const [loadState, setLoadState] = useState<DetailLoadState>({ status: "loading" });

  const load = useCallback(async () => {
    if (!siteId) return;
    setLoadState({ status: "loading" });
    try {
      const [siteResult, cellResult] = await Promise.all([
        client.getSite(siteId),
        client.listCells(siteId)
      ]);
      setSite(siteResult);
      setCells(cellResult.cells);
      setLoadState({
        status: siteResult.online_device_count === 0 ? "offline" : "success"
      });
    } catch (error) {
      const appError = error instanceof AppError
        ? error
        : new AppError("UNKNOWN", "请求失败");
      setLoadState({
        status: appError.code === "DEVICE_OFFLINE" ? "offline" : "network",
        ...(appError.traceId ? { traceId: appError.traceId } : {})
      });
    }
  }, [siteId]);

  useEffect(() => {
    void load();
  }, [load]);

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

  if (loadState.status === "loading") {
    return (
      <View className="page site-detail-page">
        <View className="skeleton skeleton--card" />
      </View>
    );
  }

  if (loadState.status === "network") {
    return (
      <View className="page site-detail-page">
        <StatePanel
          kind="network"
          {...(loadState.traceId ? { traceId: loadState.traceId } : {})}
          onRetry={() => void load()}
        />
      </View>
    );
  }

  if (loadState.status === "offline") {
    return (
      <View className="page site-detail-page">
        <StatePanel
          kind="offline"
          {...(loadState.traceId ? { traceId: loadState.traceId } : {})}
          onRetry={() => void Taro.switchTab({ url: "/pages/sites/index" })}
        />
      </View>
    );
  }

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
        onClick={() => void Taro.navigateTo({
          url:
            `/pages/locker-selection/index?siteId=${encodeURIComponent(siteId)}` +
            `&siteName=${encodeURIComponent(site?.name ?? "")}`
        })}
      />
    </View>
  );
}

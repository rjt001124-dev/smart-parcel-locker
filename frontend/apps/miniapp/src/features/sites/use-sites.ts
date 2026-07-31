import { useCallback, useEffect, useState } from "react";
import { AppError, createTaroRequest } from "@spl/api-client/http";
import { createSiteClient } from "@spl/api-client/site-client";
import { toSiteCardView, type SiteCardView } from "@spl/domain-ui/site";
import { useLocationStore } from "../../stores/location-store";
import { getCurrentCoordinates } from "./location";

const client = createSiteClient({ request: createTaroRequest(TARO_APP_API_BASE_URL) });

type SitesState =
  | { status: "loading"; sites: SiteCardView[] }
  | { status: "success"; sites: SiteCardView[] }
  | { status: "error"; sites: SiteCardView[]; errorCode: string; traceId?: string };

export function useSites() {
  const cityCode = useLocationStore((state) => state.cityCode);
  const latitude = useLocationStore((state) => state.latitude);
  const longitude = useLocationStore((state) => state.longitude);
  const setCoordinates = useLocationStore((state) => state.setCoordinates);
  const [state, setState] = useState<SitesState>({ status: "loading", sites: [] });

  const locate = useCallback(async () => {
    setState({ status: "loading", sites: [] });
    try {
      const coordinates = await getCurrentCoordinates();
      setCoordinates(coordinates.latitude, coordinates.longitude);
    } catch {
      setState({
        status: "error",
        sites: [],
        errorCode: "LOCATION_UNAVAILABLE"
      });
    }
  }, [setCoordinates]);

  const load = useCallback(async () => {
    setState({ status: "loading", sites: [] });
    if (latitude === undefined || longitude === undefined) return;
    try {
      const result = await client.listSites({
        cityCode,
        radiusM: 5000,
        latitude,
        longitude
      });
      setState({ status: "success", sites: result.sites.map(toSiteCardView) });
    } catch (error) {
      const appError = error instanceof AppError
        ? error
        : new AppError("UNKNOWN", "请求失败");
      setState({
        status: "error",
        sites: [],
        errorCode: appError.code,
        ...(appError.traceId ? { traceId: appError.traceId } : {})
      });
    }
  }, [cityCode, latitude, longitude]);

  useEffect(() => {
    if (latitude !== undefined && longitude !== undefined) return;
    void locate();
  }, [latitude, longitude, locate]);

  useEffect(() => {
    void load();
  }, [load]);

  return {
    ...state,
    retry: latitude === undefined || longitude === undefined ? locate : load
  };
}

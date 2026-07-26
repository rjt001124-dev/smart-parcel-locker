import type { RequestFn } from "./http";

export interface CityDto {
  id: string;
  code: string;
  name: string;
  province: string;
}

export interface CellAvailabilityDto {
  size: string;
  available_count: number;
}

export interface SiteSummaryDto {
  id: string;
  site_no: string;
  name: string;
  address: string;
  latitude: number;
  longitude: number;
  distance_m?: number;
  availability: CellAvailabilityDto[];
}

export interface SiteDetailDto {
  id: string;
  site_no: string;
  name: string;
  address: string;
  latitude: number;
  longitude: number;
  open_time: string;
  close_time: string;
  online_device_count: number;
  availability: CellAvailabilityDto[];
}

export interface CellDto {
  id: string;
  cell_no: string;
  size: string;
  status: string;
}

export function createSiteClient(deps: { request: RequestFn }) {
  return {
    listCities: () => deps.request<{ cities: CityDto[] }>({ path: "/v1/cities" }),
    listSites: (input: {
      cityCode: string;
      latitude?: number;
      longitude?: number;
      radiusM?: number;
    }) => deps.request<{ sites: SiteSummaryDto[] }>({
      path: "/v1/sites",
      query: {
        city_code: input.cityCode,
        latitude: input.latitude,
        longitude: input.longitude,
        radius_m: input.radiusM
      }
    }),
    getSite: (siteId: string) =>
      deps.request<SiteDetailDto>({ path: `/v1/sites/${encodeURIComponent(siteId)}` }),
    listCells: (siteId: string, size?: string) =>
      deps.request<{ cells: CellDto[] }>({
        path: `/v1/sites/${encodeURIComponent(siteId)}/cells`,
        query: { size }
      })
  };
}

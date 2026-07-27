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

interface RawAvailabilityDto {
  size: string;
  available_count?: number;
  availableCount?: number;
}

interface RawSiteSummaryDto {
  id: string;
  site_no?: string;
  siteNo?: string;
  name: string;
  address: string;
  latitude: number;
  longitude: number;
  distance_m?: number;
  distanceM?: number;
  availability?: RawAvailabilityDto[];
}

interface RawSiteDetailDto extends RawSiteSummaryDto {
  open_time?: string;
  openTime?: string;
  close_time?: string;
  closeTime?: string;
  online_device_count?: number;
  onlineDeviceCount?: number;
}

interface RawCellDto {
  id: string;
  cell_no?: string;
  cellNo?: string;
  size: string;
  status: string;
}

function normalizeAvailability(item: RawAvailabilityDto): CellAvailabilityDto {
  return {
    size: item.size,
    available_count: item.available_count ?? item.availableCount ?? 0
  };
}

function normalizeSiteSummary(site: RawSiteSummaryDto): SiteSummaryDto {
  const distanceM = site.distance_m ?? site.distanceM;
  return {
    id: site.id,
    site_no: site.site_no ?? site.siteNo ?? "",
    name: site.name,
    address: site.address,
    latitude: site.latitude,
    longitude: site.longitude,
    ...(distanceM === undefined ? {} : { distance_m: distanceM }),
    availability: (site.availability ?? []).map(normalizeAvailability)
  };
}

function normalizeSiteDetail(site: RawSiteDetailDto): SiteDetailDto {
  return {
    ...normalizeSiteSummary(site),
    open_time: site.open_time ?? site.openTime ?? "",
    close_time: site.close_time ?? site.closeTime ?? "",
    online_device_count: site.online_device_count ?? site.onlineDeviceCount ?? 0
  };
}

function normalizeCell(cell: RawCellDto): CellDto {
  return {
    id: cell.id,
    cell_no: cell.cell_no ?? cell.cellNo ?? "",
    size: cell.size,
    status: cell.status
  };
}

export function createSiteClient(deps: { request: RequestFn }) {
  return {
    listCities: () => deps.request<{ cities: CityDto[] }>({ path: "/v1/cities" }),
    listSites: async (input: {
      cityCode: string;
      latitude?: number;
      longitude?: number;
      radiusM?: number;
    }) => {
      const result = await deps.request<{ sites: RawSiteSummaryDto[] }>({
        path: "/v1/sites",
        query: {
          city_code: input.cityCode,
          latitude: input.latitude,
          longitude: input.longitude,
          radius_m: input.radiusM
        }
      });
      return { sites: result.sites.map(normalizeSiteSummary) };
    },
    getSite: async (siteId: string) => {
      const result = await deps.request<RawSiteDetailDto>({
        path: `/v1/sites/${encodeURIComponent(siteId)}`
      });
      return normalizeSiteDetail(result);
    },
    listCells: async (siteId: string, size?: string) => {
      const result = await deps.request<{ cells: RawCellDto[] }>({
        path: `/v1/sites/${encodeURIComponent(siteId)}/cells`,
        query: { size }
      });
      return { cells: result.cells.map(normalizeCell) };
    }
  };
}

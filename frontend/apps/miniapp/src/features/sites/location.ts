import Taro from "@tarojs/taro";

export interface Coordinates {
  latitude: number;
  longitude: number;
}

type Platform = "weapp" | "h5";
type LocationGetter = (options: { type: "gcj02" }) => Promise<Coordinates>;

interface LocationDependencies {
  platform: Platform;
  previewLatitude: string;
  previewLongitude: string;
  getLocation: LocationGetter;
}

function validCoordinates(coordinates: Coordinates) {
  return Number.isFinite(coordinates.latitude)
    && Number.isFinite(coordinates.longitude)
    && coordinates.latitude >= -90
    && coordinates.latitude <= 90
    && coordinates.longitude >= -180
    && coordinates.longitude <= 180;
}

function previewCoordinates(latitude: string, longitude: string) {
  if (latitude.trim() === "" || longitude.trim() === "") return undefined;
  const coordinates = {
    latitude: Number(latitude),
    longitude: Number(longitude)
  };
  return validCoordinates(coordinates) ? coordinates : undefined;
}

function defaultDependencies(): LocationDependencies {
  return {
    platform: Taro.getEnv() === Taro.ENV_TYPE.WEB ? "h5" : "weapp",
    previewLatitude: TARO_APP_PREVIEW_LATITUDE,
    previewLongitude: TARO_APP_PREVIEW_LONGITUDE,
    getLocation: Taro.getLocation as unknown as LocationGetter
  };
}

export async function getCurrentCoordinates(
  dependencies: LocationDependencies = defaultDependencies()
): Promise<Coordinates> {
  if (dependencies.platform === "h5") {
    const preview = previewCoordinates(
      dependencies.previewLatitude,
      dependencies.previewLongitude
    );
    if (preview) return preview;
  }

  const coordinates = await dependencies.getLocation({ type: "gcj02" });
  if (!validCoordinates(coordinates)) throw new Error("Invalid coordinates");
  return coordinates;
}

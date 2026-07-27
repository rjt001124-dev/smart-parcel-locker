import Taro from "@tarojs/taro";

export interface Coordinates {
  latitude: number;
  longitude: number;
}

type LocationGetter = (options: { type: "gcj02" }) => Promise<Coordinates>;

export async function getCurrentCoordinates(
  getLocation: LocationGetter = Taro.getLocation as unknown as LocationGetter
): Promise<Coordinates> {
  const result = await getLocation({ type: "gcj02" });
  return {
    latitude: result.latitude,
    longitude: result.longitude
  };
}

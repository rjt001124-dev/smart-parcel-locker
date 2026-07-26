import { create } from "zustand";

interface LocationState {
  cityCode: string;
  cityName: string;
  latitude: number | undefined;
  longitude: number | undefined;
  selectCity: (city: { code: string; name: string }) => void;
  setCoordinates: (latitude: number, longitude: number) => void;
  reset: () => void;
}

const initialLocation = {
  cityCode: "310100",
  cityName: "上海市",
  latitude: undefined,
  longitude: undefined
};

export const useLocationStore = create<LocationState>((set) => ({
  ...initialLocation,
  selectCity: (city) => set({ cityCode: city.code, cityName: city.name }),
  setCoordinates: (latitude, longitude) => set({ latitude, longitude }),
  reset: () => set(initialLocation)
}));

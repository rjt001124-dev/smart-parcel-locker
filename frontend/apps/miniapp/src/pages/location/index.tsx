import { Text, View } from "@tarojs/components";
import Taro from "@tarojs/taro";
import { useEffect, useState } from "react";
import { createTaroRequest } from "@spl/api-client/http";
import { createSiteClient, type CityDto } from "@spl/api-client/site-client";
import { useLocationStore } from "../../stores/location-store";
import "./index.scss";

const client = createSiteClient({ request: createTaroRequest(TARO_APP_API_BASE_URL) });

export default function LocationPage() {
  const [cities, setCities] = useState<CityDto[]>([]);
  const selectCity = useLocationStore((state) => state.selectCity);

  useEffect(() => {
    void client.listCities().then((result) => setCities(result.cities));
  }, []);

  return (
    <View className="page location-page">
      <Text className="page-title">选择城市</Text>
      <View className="location-page__list">
        {cities.map((city) => (
          <View
            className="city-row"
            key={city.code}
            onClick={() => {
              selectCity(city);
              void Taro.navigateBack();
            }}
          >
            <Text>{city.name}</Text>
            <Text className="city-row__province">{city.province}</Text>
          </View>
        ))}
      </View>
    </View>
  );
}

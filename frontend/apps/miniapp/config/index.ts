import { defineConfig } from "@tarojs/cli";

const isH5 = process.env.TARO_ENV === "h5";

export default defineConfig({
  projectName: "smart-parcel-locker-miniapp",
  date: "2026-07-25",
  designWidth: 375,
  deviceRatio: { 375: 2 },
  sourceRoot: "src",
  outputRoot: isH5 ? "dist-h5" : "dist",
  framework: "react",
  compiler: "vite",
  mini: {},
  h5: {
    publicPath: "/",
    devServer: {
      host: "127.0.0.1",
      port: 10086,
      proxy: {
        "/api": {
          target: "http://127.0.0.1:8000",
          changeOrigin: true,
          rewrite: (path: string) => path.replace(/^\/api/, "")
        }
      }
    }
  },
  defineConstants: {
    TARO_APP_API_BASE_URL: JSON.stringify(
      process.env.TARO_APP_API_BASE_URL ?? "http://127.0.0.1:8000"
    ),
    TARO_APP_PREVIEW_LATITUDE: JSON.stringify(
      process.env.TARO_APP_PREVIEW_LATITUDE ?? ""
    ),
    TARO_APP_PREVIEW_LONGITUDE: JSON.stringify(
      process.env.TARO_APP_PREVIEW_LONGITUDE ?? ""
    )
  }
});

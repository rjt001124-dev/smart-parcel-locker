import { defineConfig } from "@tarojs/cli";

export default defineConfig({
  projectName: "smart-parcel-locker-miniapp",
  date: "2026-07-25",
  designWidth: 375,
  deviceRatio: { 375: 2 },
  sourceRoot: "src",
  outputRoot: "dist",
  framework: "react",
  compiler: "vite",
  mini: {},
  h5: {},
  env: {
    TARO_APP_API_BASE_URL: JSON.stringify(
      process.env.TARO_APP_API_BASE_URL ?? "http://127.0.0.1:8000"
    )
  }
});

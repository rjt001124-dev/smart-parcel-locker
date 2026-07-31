# Mini App H5 Preview Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a browser-accessible H5 development build for the existing Taro mini app while preserving the WeChat build and using the real Kratos site APIs.

**Architecture:** Keep `frontend/apps/miniapp` as the single UI implementation and add Taro's H5 platform plugin. Route browser API traffic through the H5 development proxy, and extend the existing location adapter so H5 development can use an explicit Shanghai preview coordinate while production and WeChat builds continue using real location APIs.

**Tech Stack:** Taro 4.2.1, React 18, TypeScript 5.8, Vite, Vitest, Zustand, Go Kratos HTTP API.

---

## File Structure

- Modify `frontend/apps/miniapp/package.json`: add H5 scripts and platform dependencies.
- Modify `frontend/package.json`: expose root-level H5 development and build commands.
- Modify `frontend/package-lock.json`: lock H5 plugin and `cross-env` versions.
- Modify `frontend/apps/miniapp/config/index.ts`: separate H5 and WeChat output directories, define preview constants, and configure the H5 API proxy.
- Modify `frontend/apps/miniapp/config/index.test.ts`: verify build constants, output paths, and proxy behavior.
- Modify `frontend/apps/miniapp/src/types/env.d.ts`: type the two preview-coordinate constants.
- Create `frontend/apps/miniapp/src/index.html`: provide the required Taro Vite H5 document entry.
- Modify `frontend/apps/miniapp/src/features/sites/location.ts`: validate coordinates and select preview or runtime positioning by platform.
- Modify `frontend/apps/miniapp/src/features/sites/location.test.ts`: cover WeChat, H5 preview, H5 browser location, and invalid values.
- Modify `frontend/apps/miniapp/src/app.scss`: add H5 mobile viewport styling without changing mini-app styling.
- Modify `docs/operations/miniapp-local-development.md`: document one-command browser startup and validation.

### Task 1: Add and verify the H5 build configuration

**Files:**
- Modify: `frontend/apps/miniapp/package.json`
- Modify: `frontend/package.json`
- Modify: `frontend/package-lock.json`
- Modify: `frontend/apps/miniapp/config/index.ts`
- Test: `frontend/apps/miniapp/config/index.test.ts`
- Modify: `frontend/apps/miniapp/src/types/env.d.ts`
- Create: `frontend/apps/miniapp/src/index.html`

- [ ] **Step 1: Replace the config test with failing H5 expectations**

Update `frontend/apps/miniapp/config/index.test.ts` so it reloads the config after setting build variables and checks both platforms:

```ts
// @vitest-environment node

import { afterEach, describe, expect, it, vi } from "vitest";

afterEach(() => {
  vi.unstubAllEnvs();
  vi.resetModules();
});

async function loadConfig() {
  return (await import("./index")).default as {
    outputRoot?: string;
    defineConstants?: Record<string, unknown>;
    h5?: {
      devServer?: {
        port?: number;
        proxy?: Record<string, { target?: string; changeOrigin?: boolean }>;
      };
    };
  };
}

describe("Taro build configuration", () => {
  it("keeps the WeChat API URL and output directory", async () => {
    vi.stubEnv("TARO_ENV", "weapp");
    const config = await loadConfig();

    expect(config.outputRoot).toBe("dist");
    expect(config.defineConstants?.TARO_APP_API_BASE_URL).toBe(
      JSON.stringify("http://127.0.0.1:8000")
    );
  });

  it("uses the H5 proxy and a separate output directory", async () => {
    vi.stubEnv("TARO_ENV", "h5");
    vi.stubEnv("TARO_APP_API_BASE_URL", "/api");
    vi.stubEnv("TARO_APP_PREVIEW_LATITUDE", "31.2304");
    vi.stubEnv("TARO_APP_PREVIEW_LONGITUDE", "121.4737");
    const config = await loadConfig();

    expect(config.outputRoot).toBe("dist-h5");
    expect(config.defineConstants).toMatchObject({
      TARO_APP_API_BASE_URL: JSON.stringify("/api"),
      TARO_APP_PREVIEW_LATITUDE: JSON.stringify("31.2304"),
      TARO_APP_PREVIEW_LONGITUDE: JSON.stringify("121.4737")
    });
    expect(config.h5?.devServer?.port).toBe(10086);
    expect(config.h5?.devServer?.proxy?.["/api"]).toMatchObject({
      target: "http://127.0.0.1:8000",
      changeOrigin: true
    });
  });
});
```

- [ ] **Step 2: Run the config test and verify it fails**

Run:

```powershell
Set-Location frontend
npm test -w @spl/miniapp -- config/index.test.ts
```

Expected: FAIL because the current config always writes to `dist`, does not define preview constants, and has no H5 proxy.

- [ ] **Step 3: Install the H5 platform packages**

Run:

```powershell
Set-Location frontend
npm install -w @spl/miniapp @tarojs/plugin-platform-h5@4.2.1
npm install -D -w @spl/miniapp cross-env@7.0.3
```

Expected: `frontend/package-lock.json` records both exact versions.

- [ ] **Step 4: Add H5 scripts**

Add these scripts to `frontend/apps/miniapp/package.json`:

```json
"dev:h5": "cross-env TARO_APP_API_BASE_URL=/api TARO_APP_PREVIEW_LATITUDE=31.2304 TARO_APP_PREVIEW_LONGITUDE=121.4737 taro build --type h5 --watch",
"build:h5": "taro build --type h5"
```

Add these scripts to `frontend/package.json`:

```json
"dev:h5": "npm run dev:h5 -w @spl/miniapp",
"build:h5": "npm run build:h5 -w @spl/miniapp"
```

- [ ] **Step 5: Implement platform-specific Taro configuration**

Update `frontend/apps/miniapp/config/index.ts` to use a separate H5 output directory, expose all compile-time constants, and proxy `/api` to Kratos:

```ts
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
```

- [ ] **Step 6: Type the new constants**

Update `frontend/apps/miniapp/src/types/env.d.ts`:

```ts
declare const TARO_APP_API_BASE_URL: string;
declare const TARO_APP_PREVIEW_LATITUDE: string;
declare const TARO_APP_PREVIEW_LONGITUDE: string;
```

- [ ] **Step 7: Run the config test and typecheck**

Run:

```powershell
Set-Location frontend
npm test -w @spl/miniapp -- config/index.test.ts
npm run typecheck -w @spl/miniapp
```

Expected: both commands exit 0.

- [ ] **Step 8: Commit the build configuration**

```powershell
git add frontend/package.json frontend/package-lock.json frontend/apps/miniapp/package.json frontend/apps/miniapp/config/index.ts frontend/apps/miniapp/config/index.test.ts frontend/apps/miniapp/src/types/env.d.ts
git commit -m "feat: add mini app H5 build"
```

### Task 2: Add a tested cross-platform location adapter

**Files:**
- Modify: `frontend/apps/miniapp/src/features/sites/location.ts`
- Test: `frontend/apps/miniapp/src/features/sites/location.test.ts`

- [ ] **Step 1: Add failing location-adapter tests**

Replace `frontend/apps/miniapp/src/features/sites/location.test.ts` with tests for all supported paths:

```ts
import { describe, expect, it, vi } from "vitest";
import { getCurrentCoordinates } from "./location";

describe("current location", () => {
  it("uses explicit preview coordinates for H5", async () => {
    const getLocation = vi.fn();

    await expect(getCurrentCoordinates({
      platform: "h5",
      previewLatitude: "31.2304",
      previewLongitude: "121.4737",
      getLocation
    })).resolves.toEqual({ latitude: 31.2304, longitude: 121.4737 });
    expect(getLocation).not.toHaveBeenCalled();
  });

  it("uses runtime positioning when the H5 preview pair is incomplete", async () => {
    const getLocation = vi.fn().mockResolvedValue({
      latitude: 30.2741,
      longitude: 120.1551
    });

    await expect(getCurrentCoordinates({
      platform: "h5",
      previewLatitude: "31.2304",
      previewLongitude: "",
      getLocation
    })).resolves.toEqual({ latitude: 30.2741, longitude: 120.1551 });
  });

  it("requests GCJ-02 coordinates for WeChat", async () => {
    const getLocation = vi.fn().mockResolvedValue({
      latitude: 31.2304,
      longitude: 121.4737
    });

    await expect(getCurrentCoordinates({
      platform: "weapp",
      previewLatitude: "",
      previewLongitude: "",
      getLocation
    })).resolves.toEqual({ latitude: 31.2304, longitude: 121.4737 });
    expect(getLocation).toHaveBeenCalledWith({ type: "gcj02" });
  });

  it("rejects invalid runtime coordinates", async () => {
    const getLocation = vi.fn().mockResolvedValue({
      latitude: Number.NaN,
      longitude: 121.4737
    });

    await expect(getCurrentCoordinates({
      platform: "h5",
      previewLatitude: "",
      previewLongitude: "",
      getLocation
    })).rejects.toThrow("Invalid coordinates");
  });
});
```

- [ ] **Step 2: Run the location test and verify it fails**

Run:

```powershell
Set-Location frontend
npm test -w @spl/miniapp -- src/features/sites/location.test.ts
```

Expected: FAIL because `getCurrentCoordinates` does not accept platform and preview dependencies.

- [ ] **Step 3: Implement the minimal adapter**

Update `frontend/apps/miniapp/src/features/sites/location.ts` with a dependency object, finite/range validation, and default runtime dependencies:

```ts
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
```

- [ ] **Step 4: Run focused and workspace tests**

Run:

```powershell
Set-Location frontend
npm test -w @spl/miniapp -- src/features/sites/location.test.ts
npm test
```

Expected: the location test reports 4 passing tests and all workspace tests exit 0.

- [ ] **Step 5: Commit the location adapter**

```powershell
git add frontend/apps/miniapp/src/features/sites/location.ts frontend/apps/miniapp/src/features/sites/location.test.ts
git commit -m "feat: support H5 preview location"
```

### Task 3: Make the H5 surface behave like the 375px prototype

**Files:**
- Modify: `frontend/apps/miniapp/src/app.scss`

- [ ] **Step 1: Add H5-only mobile viewport rules**

Append these rules to `frontend/apps/miniapp/src/app.scss`:

```scss
/* Taro H5 host elements; ignored by the WeChat stylesheet compiler. */
html,
body,
#app {
  min-height: 100%;
  margin: 0;
  background: $color-background;
}

@media (min-width: 376px) {
  #app {
    width: 375px;
    min-height: 100vh;
    margin: 0 auto;
    box-shadow: 0 0 32px rgb(15 23 42 / 10%);
  }
}
```

- [ ] **Step 2: Build both targets**

Run:

```powershell
Set-Location frontend
npm run build:miniapp
npm run build:h5
```

Expected: WeChat output remains in `apps/miniapp/dist`; H5 output is created in `apps/miniapp/dist-h5`; both commands exit 0.

- [ ] **Step 3: Commit H5 viewport styling**

```powershell
git add frontend/apps/miniapp/src/app.scss
git commit -m "style: frame H5 preview at mobile width"
```

### Task 4: Document, start, and verify the browser preview

**Files:**
- Modify: `docs/operations/miniapp-local-development.md`

- [ ] **Step 1: Add browser startup instructions**

Append a section containing these exact commands:

````markdown
## 启动 H5 浏览器预览

先启动后端服务：

```powershell
docker compose -f deploy/docker-compose.yml up -d mysql redis api worker
```

然后启动浏览器版：

```powershell
Set-Location frontend
npm ci
npm run dev:h5
```

浏览器访问 `http://127.0.0.1:10086/`。本地开发命令使用上海预览坐标，但网点数据仍来自 `http://127.0.0.1:8000` 的真实 API。生产 H5 构建不会自动写入预览坐标。
````

- [ ] **Step 2: Run the full verification suite**

Run:

```powershell
Set-Location frontend
npm run typecheck
npm test
npm run build:miniapp
npm run build:h5
```

Expected: every command exits 0, all tests pass, and both output directories exist.

- [ ] **Step 3: Start the H5 development server**

Run from `frontend`:

```powershell
npm run dev:h5
```

Expected: the process remains running and reports `http://127.0.0.1:10086/`.

- [ ] **Step 4: Verify the service boundaries**

In a second shell, run:

```powershell
(Invoke-WebRequest -UseBasicParsing http://127.0.0.1:8000/readyz).StatusCode
(Invoke-WebRequest -UseBasicParsing http://127.0.0.1:10086/).StatusCode
(Invoke-WebRequest -UseBasicParsing "http://127.0.0.1:10086/api/v1/sites?city_code=310100&latitude=31.2304&longitude=121.4737&radius_m=5000").Content
```

Expected: both status codes are `200`, and the proxied site response contains `SITE-SH-001` and `SITE-SH-002`.

- [ ] **Step 5: Complete browser visual acceptance**

Open `http://127.0.0.1:10086/` at a 375 x 812 viewport and verify:

- Home displays the two real Shanghai site cards.
- The city selector opens and can return to home.
- The site tab displays the same real sites.
- Selecting a site opens its detail page.
- Network and location error panels retain their approved copy.

- [ ] **Step 6: Commit documentation and final verification state**

```powershell
git add docs/operations/miniapp-local-development.md
git commit -m "docs: add H5 preview workflow"
```

- [ ] **Step 7: Push the branch and update Draft PR #3**

Run:

```powershell
git push -u origin codex/milestone-3-site-device
gh pr view 3 --repo rjt001124-dev/smart-parcel-locker --web
```

Expected: GitHub branch contains all H5 commits and Draft PR #3 lists them.

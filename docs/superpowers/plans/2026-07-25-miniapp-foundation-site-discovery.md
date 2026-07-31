# Mini App Foundation and Site Discovery Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a runnable Taro + React + TypeScript mini program foundation and complete pages 01–04: home, city selection, nearby sites, and site detail, backed by the existing public Go Kratos site APIs.

**Architecture:** Add an isolated npm workspace under `frontend/`. The mini program composes feature modules from shared design tokens, domain view models, and a typed HTTP client; page components never parse raw Kratos responses directly. This is implementation plan 1 of 3: plan 2 will add reservation/payment/order/device flows, and plan 3 will add notifications, scanning, door safety, the remaining state matrix, and full visual QA.

**Tech Stack:** Taro 4.2.1, React 18.3.1, TypeScript 5.8, Vite through Taro, Zustand 5, Vitest 4, Testing Library, Sass, existing Go Kratos HTTP APIs.

---

## File Structure

```text
frontend/
  package.json                         npm workspace scripts and pinned toolchain
  tsconfig.base.json                   shared strict TypeScript settings
  apps/miniapp/
    package.json                       Taro application dependencies and scripts
    config/index.ts                    Taro compiler and environment configuration
    project.config.json                WeChat developer-tool project metadata
    src/app.config.ts                  page registration and bottom tabs
    src/app.tsx                        application root
    src/app.scss                       global reset and token imports
    src/types/env.d.ts                 build-time environment declarations
    src/stores/location-store.ts       selected city and coordinates only
    src/pages/home/*                   page 01 composition
    src/pages/location/*               page 02 composition
    src/pages/sites/*                  page 03 composition
    src/pages/site-detail/*            page 04 composition
    src/test/setup.ts                  Vitest/Taro test setup
  packages/design-tokens/
    src/index.ts                       semantic token exports
    src/tokens.scss                    CSS/Sass variables for Taro pages
    src/index.test.ts                  invariant tests for approved palette and spacing
  packages/domain-ui/
    src/site.ts                        raw site data to UI view-model mapping
    src/site.test.ts                   mapping and long-copy tests
  packages/api-client/
    src/http.ts                        request, trace ID, and stable error mapping
    src/site-client.ts                 cities/sites/site/cells contract
    src/site-client.test.ts            typed transport tests
```

Do not modify generated Go files or expose `/v1/internal` routes to the mini program. The existing public routes used in this plan are `/v1/cities`, `/v1/sites`, `/v1/sites/{site_id}`, and `/v1/sites/{site_id}/cells`.

### Task 1: Create the frontend workspace and Taro application shell

**Files:**
- Create: `frontend/package.json`
- Create: `frontend/tsconfig.base.json`
- Create: `frontend/apps/miniapp/package.json`
- Create: `frontend/apps/miniapp/tsconfig.json`
- Create: `frontend/apps/miniapp/vitest.config.ts`
- Create: `frontend/apps/miniapp/config/index.ts`
- Create: `frontend/apps/miniapp/project.config.json`
- Create: `frontend/apps/miniapp/src/types/env.d.ts`
- Create: `frontend/apps/miniapp/src/test/setup.ts`
- Create: `frontend/apps/miniapp/src/app.tsx`
- Create: `frontend/apps/miniapp/src/app.config.ts`
- Create: `frontend/apps/miniapp/src/app.scss`
- Modify: `.gitignore`

- [ ] **Step 1: Write the workspace manifests**

Create `frontend/package.json`:

```json
{
  "name": "smart-parcel-locker-frontend",
  "private": true,
  "workspaces": ["apps/*", "packages/*"],
  "scripts": {
    "build:miniapp": "npm run build:weapp -w @spl/miniapp",
    "test": "npm run test --workspaces --if-present",
    "typecheck": "npm run typecheck --workspaces --if-present"
  },
  "engines": { "node": ">=22.0.0", "npm": ">=11.0.0" }
}
```

Create `frontend/tsconfig.base.json`:

```json
{
  "compilerOptions": {
    "target": "ES2022",
    "module": "ESNext",
    "moduleResolution": "Bundler",
    "strict": true,
    "noUncheckedIndexedAccess": true,
    "exactOptionalPropertyTypes": true,
    "skipLibCheck": true,
    "jsx": "react-jsx",
    "baseUrl": "."
  }
}
```

- [ ] **Step 2: Write the mini program package manifest**

Create `frontend/apps/miniapp/package.json`:

```json
{
  "name": "@spl/miniapp",
  "private": true,
  "version": "0.1.0",
  "scripts": {
    "dev:weapp": "taro build --type weapp --watch",
    "build:weapp": "taro build --type weapp",
    "test": "vitest run",
    "typecheck": "tsc --noEmit"
  },
  "dependencies": {
    "@tarojs/components": "4.2.1",
    "@tarojs/plugin-framework-react": "4.2.1",
    "@tarojs/plugin-platform-weapp": "4.2.1",
    "@tarojs/react": "4.2.1",
    "@tarojs/runtime": "4.2.1",
    "@tarojs/taro": "4.2.1",
    "react": "18.3.1",
    "react-dom": "18.3.1",
    "zustand": "5.0.8"
  },
  "devDependencies": {
    "@tarojs/cli": "4.2.1",
    "@testing-library/jest-dom": "6.6.3",
    "@testing-library/react": "16.3.0",
    "@types/react": "18.3.24",
    "@types/react-dom": "18.3.7",
    "jsdom": "26.1.0",
    "sass": "1.89.2",
    "typescript": "5.8.3",
    "vitest": "4.1.0"
  }
}
```

Create `frontend/apps/miniapp/tsconfig.json`:

```json
{
  "extends": "../../tsconfig.base.json",
  "compilerOptions": { "types": ["vitest/globals", "@tarojs/taro"] },
  "include": ["config/**/*.ts", "src/**/*.ts", "src/**/*.tsx", "vitest.config.ts"]
}
```

Create `frontend/apps/miniapp/vitest.config.ts`:

```ts
import { defineConfig } from "vitest/config";

export default defineConfig({
  test: {
    environment: "jsdom",
    setupFiles: ["./src/test/setup.ts"]
  }
});
```

- [ ] **Step 3: Configure Taro and the four initial pages**

Create `frontend/apps/miniapp/config/index.ts`:

```ts
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
```

Create `frontend/apps/miniapp/src/app.config.ts`:

```ts
export default defineAppConfig({
  pages: [
    "pages/home/index",
    "pages/location/index",
    "pages/sites/index",
    "pages/site-detail/index"
  ],
  window: {
    navigationBarTitleText: "智能快递柜",
    navigationBarBackgroundColor: "#FFFFFF",
    navigationBarTextStyle: "black",
    backgroundColor: "#F7F9FC"
  },
  tabBar: {
    color: "#64748B",
    selectedColor: "#1769E0",
    backgroundColor: "#FFFFFF",
    list: [
      { pagePath: "pages/home/index", text: "首页" },
      { pagePath: "pages/sites/index", text: "网点" }
    ]
  }
});
```

The two unavailable tabs, orders and profile, are not registered as inert buttons in this plan. Plan 2 will add them when their pages have working routes.

- [ ] **Step 4: Add root files and environment types**

Create `frontend/apps/miniapp/src/app.tsx`:

```tsx
import type { PropsWithChildren } from "react";
import "./app.scss";

export default function App({ children }: PropsWithChildren) {
  return children;
}
```

Create `frontend/apps/miniapp/src/types/env.d.ts`:

```ts
declare const TARO_APP_API_BASE_URL: string;
```

Create `frontend/apps/miniapp/src/test/setup.ts`:

```ts
import "@testing-library/jest-dom/vitest";
```

Create `frontend/apps/miniapp/src/app.scss`:

```scss
page {
  background: #f7f9fc;
  color: #0f172a;
  font-family: -apple-system, BlinkMacSystemFont, "PingFang SC", "Microsoft YaHei", sans-serif;
  font-size: 14px;
}

button::after { border: 0; }
```

- [ ] **Step 5: Add developer-tool metadata and ignore generated output**

Create `frontend/apps/miniapp/project.config.json` with `appid` deliberately blank for local import:

```json
{
  "miniprogramRoot": "dist/",
  "projectname": "smart-parcel-locker-miniapp",
  "description": "智能快递柜小程序",
  "appid": "touristappid",
  "setting": { "es6": true, "enhance": true, "minified": false }
}
```

Append to `.gitignore`:

```gitignore
frontend/node_modules/
frontend/apps/miniapp/dist/
frontend/coverage/
```

- [ ] **Step 6: Install and verify the empty shell**

Run:

```powershell
Set-Location frontend
npm install
npm run typecheck
npm run build:miniapp
```

Expected: npm creates `frontend/package-lock.json`; TypeScript exits 0; Taro creates `frontend/apps/miniapp/dist/app.json`.

- [ ] **Step 7: Commit**

```powershell
git add .gitignore frontend/package.json frontend/package-lock.json frontend/tsconfig.base.json frontend/apps/miniapp
git commit -m "feat: scaffold Taro mini app"
```

### Task 2: Add approved design tokens and primitive components

**Files:**
- Create: `frontend/packages/design-tokens/package.json`
- Create: `frontend/packages/design-tokens/tsconfig.json`
- Create: `frontend/packages/design-tokens/src/index.ts`
- Create: `frontend/packages/design-tokens/src/tokens.scss`
- Create: `frontend/packages/design-tokens/src/index.test.ts`
- Modify: `frontend/apps/miniapp/package.json`
- Modify: `frontend/apps/miniapp/src/app.scss`
- Create: `frontend/apps/miniapp/src/components/primary-button.tsx`
- Create: `frontend/apps/miniapp/src/components/status-pill.tsx`
- Test: `frontend/apps/miniapp/src/components/primitives.test.tsx`

- [ ] **Step 1: Write failing token invariant tests**

Create `frontend/packages/design-tokens/src/index.test.ts`:

```ts
import { describe, expect, it } from "vitest";
import { colors, radii, spacing } from "./index";

describe("approved design tokens", () => {
  it("keeps the approved blue and pure-white surfaces", () => {
    expect(colors.primary).toBe("#1769E0");
    expect(colors.surface).toBe("#FFFFFF");
    expect(colors.background).toBe("#F7F9FC");
  });

  it("uses the 8px spacing baseline and restrained radii", () => {
    expect(spacing[2]).toBe(8);
    expect(radii.card).toBe(16);
    expect(radii.button).toBe(12);
  });
});
```

- [ ] **Step 2: Run the test and verify it fails**

Run: `npm test -w @spl/design-tokens`

Expected: FAIL because the package and exports do not exist.

- [ ] **Step 3: Implement the token package**

Create `frontend/packages/design-tokens/package.json`:

```json
{
  "name": "@spl/design-tokens",
  "private": true,
  "version": "0.1.0",
  "type": "module",
  "exports": { ".": "./src/index.ts", "./src/tokens.scss": "./src/tokens.scss" },
  "scripts": { "test": "vitest run", "typecheck": "tsc --noEmit" },
  "devDependencies": { "typescript": "5.8.3", "vitest": "4.1.0" }
}
```

Create `frontend/packages/design-tokens/tsconfig.json`:

```json
{
  "extends": "../../tsconfig.base.json",
  "include": ["src/**/*.ts"]
}
```

Create `frontend/packages/design-tokens/src/index.ts`:

```ts
export const colors = {
  primary: "#1769E0",
  primaryPressed: "#1258BD",
  surface: "#FFFFFF",
  background: "#F7F9FC",
  textPrimary: "#0F172A",
  textSecondary: "#64748B",
  border: "#DBE5F2",
  success: "#16844B",
  warning: "#A46400",
  danger: "#B42318"
} as const;

export const spacing = [0, 4, 8, 12, 16, 24, 32, 40] as const;
export const radii = { button: 12, input: 12, card: 16, sheet: 18 } as const;
export const shadows = { card: "0 4px 18px rgba(51, 65, 85, 0.07)" } as const;
```

Create `frontend/packages/design-tokens/src/tokens.scss`:

```scss
$color-primary: #1769e0;
$color-primary-pressed: #1258bd;
$color-surface: #ffffff;
$color-background: #f7f9fc;
$color-text-primary: #0f172a;
$color-text-secondary: #64748b;
$color-border: #dbe5f2;
$color-success: #16844b;
$color-warning: #a46400;
$color-danger: #b42318;
$radius-button: 12px;
$radius-card: 16px;
```

Register the new workspace dependency and replace the literal global colors with token variables:

```powershell
Set-Location frontend
npm install
npm install -w @spl/miniapp "@spl/design-tokens@*"
```

Update `frontend/apps/miniapp/src/app.scss`:

```scss
@use "@spl/design-tokens/src/tokens.scss" as *;
page {
  background: $color-background;
  color: $color-text-primary;
  font-family: -apple-system, BlinkMacSystemFont, "PingFang SC", "Microsoft YaHei", sans-serif;
  font-size: 14px;
}
button::after { border: 0; }
```

- [ ] **Step 4: Write primitive component tests**

Create `frontend/apps/miniapp/src/components/primitives.test.tsx`:

```tsx
import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { PrimaryButton } from "./primary-button";
import { StatusPill } from "./status-pill";

describe("mini app primitives", () => {
  it("blocks duplicate action while loading", () => {
    const onClick = vi.fn();
    render(<PrimaryButton loading onClick={onClick} label="查看订单" />);
    fireEvent.click(screen.getByRole("button"));
    expect(onClick).not.toHaveBeenCalled();
    expect(screen.getByText("处理中…")).toBeTruthy();
  });

  it("renders semantic status copy", () => {
    render(<StatusPill tone="success" label="营业中" />);
    expect(screen.getByText("营业中")).toBeTruthy();
  });
});
```

- [ ] **Step 5: Implement the primitive components**

Create `frontend/apps/miniapp/src/components/primary-button.tsx`:

```tsx
import { Button } from "@tarojs/components";
import "./primary-button.scss";

export function PrimaryButton(props: {
  label: string;
  loading?: boolean;
  onClick: () => void;
}) {
  return (
    <Button
      className="primary-button"
      disabled={props.loading}
      onClick={props.onClick}
    >
      {props.loading ? "处理中…" : props.label}
    </Button>
  );
}
```

Create `frontend/apps/miniapp/src/components/status-pill.tsx`:

```tsx
import { Text } from "@tarojs/components";

export function StatusPill(props: {
  label: string;
  tone: "success" | "warning" | "danger" | "neutral";
}) {
  return <Text className={`status-pill status-pill--${props.tone}`}>{props.label}</Text>;
}
```

Create `frontend/apps/miniapp/src/components/primary-button.scss`:

```scss
@use "@spl/design-tokens/src/tokens.scss" as *;

.primary-button {
  width: 100%;
  min-height: 44px;
  border-radius: $radius-button;
  background: $color-primary;
  color: #fff;
  font-size: 16px;
  font-weight: 600;
}
.primary-button[disabled] { opacity: 0.6; }
```

- [ ] **Step 6: Run tests and type checks**

Run:

```powershell
Set-Location frontend
npm test -w @spl/design-tokens
npm test -w @spl/miniapp
npm run typecheck
```

Expected: all token and primitive tests PASS; TypeScript exits 0.

- [ ] **Step 7: Commit**

```powershell
git add frontend/package-lock.json frontend/packages/design-tokens frontend/apps/miniapp/package.json frontend/apps/miniapp/src/app.scss frontend/apps/miniapp/src/components
git commit -m "feat: add mini app design system primitives"
```

### Task 3: Add typed public-site API and stable error mapping

**Files:**
- Create: `frontend/packages/api-client/package.json`
- Create: `frontend/packages/api-client/tsconfig.json`
- Create: `frontend/packages/api-client/src/http.ts`
- Create: `frontend/packages/api-client/src/site-client.ts`
- Test: `frontend/packages/api-client/src/site-client.test.ts`
- Modify: `frontend/apps/miniapp/package.json`

- [ ] **Step 1: Create the API-client package**

Create `frontend/packages/api-client/package.json`:

```json
{
  "name": "@spl/api-client",
  "private": true,
  "version": "0.1.0",
  "type": "module",
  "exports": { "./*": "./src/*.ts" },
  "scripts": { "test": "vitest run", "typecheck": "tsc --noEmit" },
  "dependencies": { "@tarojs/taro": "4.2.1" },
  "devDependencies": { "typescript": "5.8.3", "vitest": "4.1.0" }
}
```

Create `frontend/packages/api-client/tsconfig.json`:

```json
{
  "extends": "../../tsconfig.base.json",
  "include": ["src/**/*.ts"]
}
```

Register the workspace and its mini-program dependency:

```powershell
Set-Location frontend
npm install
npm install -w @spl/miniapp "@spl/api-client@*"
```

- [ ] **Step 2: Write failing API-client tests**

Create `frontend/packages/api-client/src/site-client.test.ts`:

```ts
import { describe, expect, it, vi } from "vitest";
import { createSiteClient } from "./site-client";

describe("site client", () => {
  it("sends the approved nearby-site query", async () => {
    const request = vi.fn().mockResolvedValue({ sites: [] });
    const client = createSiteClient({ request });
    await client.listSites({
      cityCode: "310100",
      latitude: 31.2304,
      longitude: 121.4737,
      radiusM: 5000
    });
    expect(request).toHaveBeenCalledWith({
      path: "/v1/sites",
      query: {
        city_code: "310100",
        latitude: 31.2304,
        longitude: 121.4737,
        radius_m: 5000
      }
    });
  });

  it("never sends the internal token from a public client", async () => {
    const request = vi.fn().mockResolvedValue({ cities: [] });
    const client = createSiteClient({ request });
    await client.listCities();
    expect(JSON.stringify(request.mock.calls)).not.toContain("X-Internal-Token");
  });
});
```

- [ ] **Step 3: Run the tests and verify they fail**

Run: `npm test -w @spl/api-client`

Expected: FAIL because `createSiteClient` is not defined.

- [ ] **Step 4: Implement the WeChat-compatible HTTP transport and error normalization**

Create `frontend/packages/api-client/src/http.ts`:

```ts
import Taro from "@tarojs/taro";

export type AppErrorCode =
  | "NETWORK_FAILURE"
  | "NOT_FOUND"
  | "DEVICE_OFFLINE"
  | "PERMISSION_DENIED"
  | "UNKNOWN";

export class AppError extends Error {
  constructor(
    public readonly code: AppErrorCode,
    message: string,
    public readonly traceId?: string
  ) {
    super(message);
  }
}

export type RequestInput = {
  path: string;
  query?: Record<string, string | number | boolean | undefined>;
};

export type RequestFn = <T>(input: RequestInput) => Promise<T>;

export function createTaroRequest(baseUrl: string): RequestFn {
  return async <T>(input: RequestInput): Promise<T> => {
    const query = Object.entries(input.query ?? {})
      .filter((entry): entry is [string, string | number | boolean] => entry[1] !== undefined)
      .map(([key, value]) => `${encodeURIComponent(key)}=${encodeURIComponent(String(value))}`)
      .join("&");
    const url = `${baseUrl.replace(/\/$/, "")}${input.path}${query ? `?${query}` : ""}`;
    try {
      const response = await Taro.request<T>({ url, method: "GET" });
      const traceId = String(response.header["x-request-id"] ?? "") || undefined;
      if (response.statusCode < 200 || response.statusCode >= 300) {
        const code = response.statusCode === 404
          ? "NOT_FOUND"
          : response.statusCode === 403
            ? "PERMISSION_DENIED"
            : "UNKNOWN";
        throw new AppError(code, "请求失败", traceId);
      }
      return response.data;
    } catch (error) {
      if (error instanceof AppError) throw error;
      throw new AppError("NETWORK_FAILURE", "网络连接失败");
    }
  };
}
```

- [ ] **Step 5: Implement the public site contracts**

Create `frontend/packages/api-client/src/site-client.ts`:

```ts
import type { RequestFn } from "./http";

export type CityDto = { id: string; code: string; name: string; province: string };
export type CellAvailabilityDto = { size: string; available_count: number };
export type SiteSummaryDto = {
  id: string;
  site_no: string;
  name: string;
  address: string;
  latitude: number;
  longitude: number;
  distance_m?: number;
  availability: CellAvailabilityDto[];
};
export type SiteDetailDto = {
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
};
export type CellDto = { id: string; cell_no: string; size: string; status: string };

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
      deps.request<SiteDetailDto>({ path: `/v1/sites/${siteId}` }),
    listCells: (siteId: string, size?: string) =>
      deps.request<{ cells: CellDto[] }>({
        path: `/v1/sites/${siteId}/cells`,
        query: { size }
      })
  };
}
```

- [ ] **Step 6: Run the API tests and type checks**

Run:

```powershell
Set-Location frontend
npm test -w @spl/api-client
npm run typecheck
```

Expected: API tests PASS; no public request contains `X-Internal-Token`.

- [ ] **Step 7: Commit**

```powershell
git add frontend/package-lock.json frontend/packages/api-client frontend/apps/miniapp/package.json
git commit -m "feat: add typed public site client"
```

### Task 4: Add site view models and location state

**Files:**
- Create: `frontend/packages/domain-ui/package.json`
- Create: `frontend/packages/domain-ui/tsconfig.json`
- Create: `frontend/packages/domain-ui/src/site.ts`
- Test: `frontend/packages/domain-ui/src/site.test.ts`
- Create: `frontend/apps/miniapp/src/stores/location-store.ts`
- Test: `frontend/apps/miniapp/src/stores/location-store.test.ts`
- Modify: `frontend/apps/miniapp/package.json`

- [ ] **Step 1: Create the domain-UI package**

Create `frontend/packages/domain-ui/package.json`:

```json
{
  "name": "@spl/domain-ui",
  "private": true,
  "version": "0.1.0",
  "type": "module",
  "exports": { "./*": "./src/*.ts" },
  "scripts": { "test": "vitest run", "typecheck": "tsc --noEmit" },
  "dependencies": { "@spl/api-client": "*" },
  "devDependencies": { "typescript": "5.8.3", "vitest": "4.1.0" }
}
```

Create `frontend/packages/domain-ui/tsconfig.json`:

```json
{
  "extends": "../../tsconfig.base.json",
  "include": ["src/**/*.ts"]
}
```

Register the workspace and its mini-program dependency:

```powershell
Set-Location frontend
npm install
npm install -w @spl/miniapp "@spl/domain-ui@*"
```

- [ ] **Step 2: Write failing site mapping tests**

Create `frontend/packages/domain-ui/src/site.test.ts`:

```ts
import { describe, expect, it } from "vitest";
import { toSiteCardView } from "./site";

describe("site view model", () => {
  it("formats distance and availability without inventing data", () => {
    expect(toSiteCardView({
      id: "1",
      site_no: "SITE-SH-001",
      name: "万象城智能寄存点",
      address: "世纪大道88号B1层",
      latitude: 31.2304,
      longitude: 121.4737,
      distance_m: 320,
      availability: [
        { size: "CELL_SIZE_SMALL", available_count: 12 },
        { size: "CELL_SIZE_MEDIUM", available_count: 8 },
        { size: "CELL_SIZE_LARGE", available_count: 4 }
      ]
    })).toEqual({
      id: "1",
      name: "万象城智能寄存点",
      address: "世纪大道88号B1层",
      distanceLabel: "320m",
      availabilityLabel: "可用24格",
      statusLabel: "有空柜",
      statusTone: "success"
    });
  });
});
```

- [ ] **Step 3: Run the test and verify it fails**

Run: `npm test -w @spl/domain-ui`

Expected: FAIL because `toSiteCardView` does not exist.

- [ ] **Step 4: Implement mapping without leaking transport fields**

Create `frontend/packages/domain-ui/src/site.ts`:

```ts
import type { SiteSummaryDto } from "@spl/api-client/site-client";

export type SiteCardView = {
  id: string;
  name: string;
  address: string;
  distanceLabel: string;
  availabilityLabel: string;
  statusLabel: string;
  statusTone: "success" | "warning" | "danger";
};

export function toSiteCardView(site: SiteSummaryDto): SiteCardView {
  const distanceLabel = site.distance_m === undefined
    ? "距离未知"
    : site.distance_m < 1000
      ? `${site.distance_m}m`
      : `${(site.distance_m / 1000).toFixed(1)}km`;
  const availableCells = site.availability.reduce((sum, item) => sum + item.available_count, 0);
  return {
    id: site.id,
    name: site.name,
    address: site.address,
    distanceLabel,
    availabilityLabel: `可用${availableCells}格`,
    statusLabel: availableCells > 0 ? "有空柜" : "暂无空柜",
    statusTone: availableCells > 0 ? "success" : "warning"
  };
}
```

- [ ] **Step 5: Write and implement the location store**

Create `frontend/apps/miniapp/src/stores/location-store.test.ts`:

```ts
import { beforeEach, describe, expect, it } from "vitest";
import { useLocationStore } from "./location-store";

describe("location store", () => {
  beforeEach(() => useLocationStore.getState().reset());
  it("stores only selected city and coordinates", () => {
    useLocationStore.getState().selectCity({ code: "310100", name: "上海市" });
    useLocationStore.getState().setCoordinates(31.2304, 121.4737);
    expect(useLocationStore.getState()).toMatchObject({
      cityCode: "310100",
      cityName: "上海市",
      latitude: 31.2304,
      longitude: 121.4737
    });
  });
});
```

Create `frontend/apps/miniapp/src/stores/location-store.ts`:

```ts
import { create } from "zustand";

type LocationState = {
  cityCode: string;
  cityName: string;
  latitude: number | undefined;
  longitude: number | undefined;
  selectCity: (city: { code: string; name: string }) => void;
  setCoordinates: (latitude: number, longitude: number) => void;
  reset: () => void;
};

const initial = { cityCode: "310100", cityName: "上海市", latitude: undefined, longitude: undefined };

export const useLocationStore = create<LocationState>((set) => ({
  ...initial,
  selectCity: (city) => set({ cityCode: city.code, cityName: city.name }),
  setCoordinates: (latitude, longitude) => set({ latitude, longitude }),
  reset: () => set(initial)
}));
```

- [ ] **Step 6: Run tests and commit**

Run:

```powershell
Set-Location frontend
npm test -w @spl/domain-ui
npm test -w @spl/miniapp
npm run typecheck
```

Expected: mapping and store tests PASS.

```powershell
git add frontend/package-lock.json frontend/packages/domain-ui frontend/apps/miniapp/package.json frontend/apps/miniapp/src/stores
git commit -m "feat: add site view models and location state"
```

### Task 5: Implement home, city, nearby-site, and site-detail pages

**Files:**
- Create: `frontend/apps/miniapp/src/features/sites/site-card.tsx`
- Create: `frontend/apps/miniapp/src/features/sites/site-card.scss`
- Create: `frontend/apps/miniapp/src/features/sites/use-sites.ts`
- Create: `frontend/apps/miniapp/src/pages/home/index.tsx`
- Create: `frontend/apps/miniapp/src/pages/home/index.scss`
- Create: `frontend/apps/miniapp/src/pages/location/index.tsx`
- Create: `frontend/apps/miniapp/src/pages/sites/index.tsx`
- Create: `frontend/apps/miniapp/src/pages/site-detail/index.tsx`
- Test: `frontend/apps/miniapp/src/pages/site-pages.test.tsx`

- [ ] **Step 1: Write failing page-flow tests**

Create `frontend/apps/miniapp/src/pages/site-pages.test.tsx`:

```tsx
import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import HomePage from "./home";
import SitesPage from "./sites";

vi.mock("@tarojs/taro", () => ({
  navigateTo: vi.fn(),
  getLocation: vi.fn()
}));

vi.mock("../features/sites/use-sites", () => ({
  useSites: () => ({
    status: "success",
    sites: [{
      id: "1",
      name: "万象城智能寄存点",
      address: "世纪大道88号B1层",
      distanceLabel: "320m",
      availabilityLabel: "可用24格",
      statusLabel: "有空柜",
      statusTone: "success"
    }],
    retry: vi.fn()
  })
}));

describe("site discovery pages", () => {
  it("keeps current-order space truthful when order API is not implemented", () => {
    render(<HomePage />);
    expect(screen.getByText("暂无进行中的订单")).toBeTruthy();
    expect(screen.getByText("万象城智能寄存点")).toBeTruthy();
  });

  it("renders the nearby-site result count", () => {
    render(<SitesPage />);
    expect(screen.getByText("共找到1个站点")).toBeTruthy();
  });
});
```

- [ ] **Step 2: Run the test and verify it fails**

Run: `npm test -w @spl/miniapp -- site-pages.test.tsx`

Expected: FAIL because the pages and site hook do not exist.

- [ ] **Step 3: Implement the reusable site card**

Create `frontend/apps/miniapp/src/features/sites/site-card.tsx`:

```tsx
import { Text, View } from "@tarojs/components";
import type { SiteCardView } from "@spl/domain-ui/site";
import { StatusPill } from "../../components/status-pill";
import "./site-card.scss";

export function SiteCard(props: { site: SiteCardView; onSelect: (id: string) => void }) {
  return (
    <View className="site-card" onClick={() => props.onSelect(props.site.id)}>
      <View className="site-card__header">
        <Text className="site-card__name">{props.site.name}</Text>
        <StatusPill tone={props.site.statusTone} label={props.site.statusLabel} />
      </View>
      <Text className="site-card__meta">
        {props.site.distanceLabel} · {props.site.availabilityLabel}
      </Text>
      <Text className="site-card__address">{props.site.address}</Text>
    </View>
  );
}
```

- [ ] **Step 4: Implement the site loading hook**

Create `frontend/apps/miniapp/src/features/sites/use-sites.ts`:

```ts
import { useCallback, useEffect, useState } from "react";
import { AppError, createTaroRequest } from "@spl/api-client/http";
import { createSiteClient } from "@spl/api-client/site-client";
import { toSiteCardView, type SiteCardView } from "@spl/domain-ui/site";
import { useLocationStore } from "../../stores/location-store";

const client = createSiteClient({ request: createTaroRequest(TARO_APP_API_BASE_URL) });

export function useSites() {
  const location = useLocationStore();
  const [state, setState] = useState<
    | { status: "loading"; sites: SiteCardView[] }
    | { status: "success"; sites: SiteCardView[] }
    | { status: "error"; sites: SiteCardView[]; errorCode: string; traceId?: string }
  >({ status: "loading", sites: [] });

  const load = useCallback(async () => {
    setState({ status: "loading", sites: [] });
    try {
      const result = await client.listSites({
        cityCode: location.cityCode,
        latitude: location.latitude,
        longitude: location.longitude,
        radiusM: 5000
      });
      setState({ status: "success", sites: result.sites.map(toSiteCardView) });
    } catch (error) {
      const appError = error instanceof AppError ? error : new AppError("UNKNOWN", "请求失败");
      setState({
        status: "error",
        sites: [],
        errorCode: appError.code,
        ...(appError.traceId ? { traceId: appError.traceId } : {})
      });
    }
  }, [location.cityCode, location.latitude, location.longitude]);

  useEffect(() => { void load(); }, [load]);
  return { ...state, retry: load };
}
```

- [ ] **Step 5: Implement page composition**

Create `frontend/apps/miniapp/src/pages/home/index.tsx`:

```tsx
import { Input, Text, View } from "@tarojs/components";
import Taro from "@tarojs/taro";
import { SiteCard } from "../../features/sites/site-card";
import { useSites } from "../../features/sites/use-sites";
import { useLocationStore } from "../../stores/location-store";
import "./index.scss";

export default function HomePage() {
  const cityName = useLocationStore((state) => state.cityName);
  const sites = useSites();
  return (
    <View className="page home-page">
      <Text className="home-page__city" onClick={() => Taro.navigateTo({ url: "/pages/location/index" })}>
        {cityName} · 定位成功
      </Text>
      <Input className="home-page__search" placeholder="搜索商场、地铁站或地址" />
      <View className="current-order current-order--empty">
        <Text className="current-order__label">当前订单</Text>
        <Text className="current-order__title">暂无进行中的订单</Text>
        <Text className="current-order__hint">选择附近网点开始寄存</Text>
      </View>
      <View className="section-heading"><Text>附近寄存点</Text><Text>查看全部</Text></View>
      {sites.sites.slice(0, 2).map((site) => (
        <SiteCard key={site.id} site={site} onSelect={(id) => Taro.navigateTo({ url: `/pages/site-detail/index?id=${id}` })} />
      ))}
    </View>
  );
}
```

Create `frontend/apps/miniapp/src/pages/sites/index.tsx`:

```tsx
import { Text, View } from "@tarojs/components";
import Taro from "@tarojs/taro";
import { SiteCard } from "../../features/sites/site-card";
import { useSites } from "../../features/sites/use-sites";

export default function SitesPage() {
  const result = useSites();
  return (
    <View className="page">
      <Text className="page-title">附近寄存点</Text>
      <Text className="page-subtitle">共找到{result.sites.length}个站点</Text>
      {result.sites.map((site) => (
        <SiteCard key={site.id} site={site} onSelect={(id) => Taro.navigateTo({ url: `/pages/site-detail/index?id=${id}` })} />
      ))}
    </View>
  );
}
```

Create `frontend/apps/miniapp/src/pages/location/index.tsx`:

```tsx
import { Text, View } from "@tarojs/components";
import Taro from "@tarojs/taro";
import { useEffect, useState } from "react";
import { createTaroRequest } from "@spl/api-client/http";
import { createSiteClient, type CityDto } from "@spl/api-client/site-client";
import { useLocationStore } from "../../stores/location-store";

const client = createSiteClient({ request: createTaroRequest(TARO_APP_API_BASE_URL) });

export default function LocationPage() {
  const [cities, setCities] = useState<CityDto[]>([]);
  const selectCity = useLocationStore((state) => state.selectCity);
  useEffect(() => { void client.listCities().then((result) => setCities(result.cities)); }, []);
  return (
    <View className="page">
      <Text className="page-title">选择城市</Text>
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
        </View>
      ))}
    </View>
  );
}
```

Create `frontend/apps/miniapp/src/pages/site-detail/index.tsx`:

```tsx
import { Text, View } from "@tarojs/components";
import Taro, { getCurrentInstance } from "@tarojs/taro";
import { useEffect, useMemo, useState } from "react";
import { createTaroRequest } from "@spl/api-client/http";
import { createSiteClient, type CellDto, type SiteDetailDto } from "@spl/api-client/site-client";
import { PrimaryButton } from "../../components/primary-button";

const client = createSiteClient({ request: createTaroRequest(TARO_APP_API_BASE_URL) });

export default function SiteDetailPage() {
  const siteId = getCurrentInstance().router?.params.id ?? "";
  const [site, setSite] = useState<SiteDetailDto>();
  const [cells, setCells] = useState<CellDto[]>([]);
  useEffect(() => {
    if (!siteId) return;
    void Promise.all([client.getSite(siteId), client.listCells(siteId)]).then(([siteResult, cellResult]) => {
      setSite(siteResult);
      setCells(cellResult.cells);
    });
  }, [siteId]);
  const counts = useMemo(() => ({
    small: cells.filter((cell) => cell.size === "CELL_SIZE_SMALL" && cell.status === "CELL_STATUS_IDLE").length,
    medium: cells.filter((cell) => cell.size === "CELL_SIZE_MEDIUM" && cell.status === "CELL_STATUS_IDLE").length,
    large: cells.filter((cell) => cell.size === "CELL_SIZE_LARGE" && cell.status === "CELL_STATUS_IDLE").length
  }), [cells]);
  return (
    <View className="page site-detail-page">
      <Text className="page-title">{site?.name ?? "正在加载网点"}</Text>
      <Text className="page-subtitle">{site?.address ?? ""}</Text>
      <View className="locker-counts">
        <Text>小号 {counts.small}</Text>
        <Text>中号 {counts.medium}</Text>
        <Text>大号 {counts.large}</Text>
      </View>
      <PrimaryButton
        label="选择柜格"
        onClick={() => void Taro.showToast({ title: "柜格预约将在下一阶段开放", icon: "none" })}
      />
    </View>
  );
}
```

- [ ] **Step 6: Add the approved page styling**

Create `frontend/apps/miniapp/src/pages/home/index.scss`:

```scss
@use "@spl/design-tokens/src/tokens.scss" as *;
.page { padding: 16px; }
.home-page__city { display: block; font-size: 17px; font-weight: 700; margin-bottom: 14px; }
.home-page__search { box-sizing: border-box; width: 100%; padding: 11px 12px; background: #f1f5f9; border-radius: 12px; }
.current-order { margin-top: 14px; padding: 17px; border-radius: $radius-card; background: $color-primary; color: #fff; }
.current-order__label, .current-order__title, .current-order__hint { display: block; }
.current-order__title { margin-top: 8px; font-size: 20px; font-weight: 700; }
.current-order__hint { margin-top: 6px; opacity: 0.82; }
.section-heading { display: flex; justify-content: space-between; margin: 20px 0 10px; font-weight: 700; }
```

Create `frontend/apps/miniapp/src/features/sites/site-card.scss`:

```scss
@use "@spl/design-tokens/src/tokens.scss" as *;
.site-card { margin-bottom: 12px; padding: 14px; border: 1px solid $color-border; border-radius: $radius-card; background: $color-surface; }
.site-card__header { display: flex; justify-content: space-between; gap: 12px; }
.site-card__name { font-size: 16px; font-weight: 700; }
.site-card__meta, .site-card__address { display: block; margin-top: 8px; color: $color-text-secondary; }
.status-pill { padding: 4px 7px; border-radius: 7px; font-size: 11px; }
.status-pill--success { color: $color-success; background: #e9f8ef; }
.status-pill--warning { color: $color-warning; background: #fff4df; }
.status-pill--danger { color: $color-danger; background: #fee2e2; }
```

- [ ] **Step 7: Run page tests, type checks, and build**

Run:

```powershell
Set-Location frontend
npm test -w @spl/miniapp
npm run typecheck
npm run build:miniapp
```

Expected: page tests PASS; TypeScript exits 0; Taro build exits 0.

- [ ] **Step 8: Commit**

```powershell
git add frontend/apps/miniapp/src/features frontend/apps/miniapp/src/pages
git commit -m "feat: add mini app site discovery pages"
```

### Task 6: Add loading, empty, network-failure, and offline state surfaces

**Files:**
- Create: `frontend/apps/miniapp/src/components/state-panel.tsx`
- Create: `frontend/apps/miniapp/src/components/state-panel.scss`
- Test: `frontend/apps/miniapp/src/components/state-panel.test.tsx`
- Modify: `frontend/apps/miniapp/src/pages/home/index.tsx`
- Modify: `frontend/apps/miniapp/src/pages/sites/index.tsx`
- Modify: `frontend/apps/miniapp/src/pages/site-detail/index.tsx`

- [ ] **Step 1: Write failing state tests**

Create `frontend/apps/miniapp/src/components/state-panel.test.tsx`:

```tsx
import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { StatePanel } from "./state-panel";

describe("StatePanel", () => {
  it("explains cause, impact, and next action for network failure", () => {
    const retry = vi.fn();
    render(<StatePanel kind="network" traceId="trace-123" onRetry={retry} />);
    expect(screen.getByText("网络连接失败")).toBeTruthy();
    expect(screen.getByText("尚未更改当前订单或柜格")).toBeTruthy();
    expect(screen.getByText("重试")).toBeTruthy();
    expect(screen.getByText("参考编号：trace-123")).toBeTruthy();
    fireEvent.click(screen.getByText("重试"));
    expect(retry).toHaveBeenCalledOnce();
  });
});
```

- [ ] **Step 2: Run the test and verify it fails**

Run: `npm test -w @spl/miniapp -- state-panel.test.tsx`

Expected: FAIL because `StatePanel` does not exist.

- [ ] **Step 3: Implement explicit recoverable states**

Create `frontend/apps/miniapp/src/components/state-panel.tsx`:

```tsx
import { Button, Text, View } from "@tarojs/components";

const copy = {
  empty: { title: "暂无可用网点", impact: "当前筛选条件下无法开始寄存", action: "调整筛选" },
  network: { title: "网络连接失败", impact: "尚未更改当前订单或柜格", action: "重试" },
  offline: { title: "设备已离线", impact: "当前站点暂时无法开门", action: "查看附近网点" }
} as const;

export function StatePanel(props: {
  kind: keyof typeof copy;
  traceId?: string;
  onRetry: () => void;
}) {
  const value = copy[props.kind];
  return (
    <View className={`state-panel state-panel--${props.kind}`}>
      <Text className="state-panel__title">{value.title}</Text>
      <Text className="state-panel__impact">{value.impact}</Text>
      {props.traceId && <Text className="state-panel__trace">参考编号：{props.traceId}</Text>}
      <Button onClick={props.onRetry}>{value.action}</Button>
    </View>
  );
}
```

- [ ] **Step 4: Integrate state branches into all four pages**

Add these branches immediately before the normal list in `home/index.tsx` and `sites/index.tsx`:

```tsx
if (sites.status === "loading") {
  return <View className="page"><View className="skeleton skeleton--title" /><View className="skeleton skeleton--card" /></View>;
}
if (sites.status === "error") {
  return (
    <View className="page">
      <StatePanel
        kind={sites.errorCode === "DEVICE_OFFLINE" ? "offline" : "network"}
        {...(sites.traceId ? { traceId: sites.traceId } : {})}
        onRetry={() => void sites.retry()}
      />
    </View>
  );
}
if (sites.sites.length === 0) {
  return <View className="page"><StatePanel kind="empty" onRetry={() => void sites.retry()} /></View>;
}
```

Replace the detail page's one-shot effect with this retryable loader and add the two early-return branches:

```tsx
const [loadState, setLoadState] = useState<"loading" | "success" | "network" | "offline">("loading");
const load = useCallback(async () => {
  if (!siteId) return;
  setLoadState("loading");
  try {
    const [siteResult, cellResult] = await Promise.all([
      client.getSite(siteId),
      client.listCells(siteId)
    ]);
    setSite(siteResult);
    setCells(cellResult.cells);
    setLoadState(siteResult.online_device_count === 0 ? "offline" : "success");
  } catch (error) {
    setLoadState(error instanceof AppError && error.code === "DEVICE_OFFLINE" ? "offline" : "network");
  }
}, [siteId]);
useEffect(() => { void load(); }, [load]);

if (loadState === "loading") {
  return <View className="page"><View className="skeleton skeleton--card" /></View>;
}
if (loadState === "network" || loadState === "offline") {
  return (
    <View className="page">
      <StatePanel kind={loadState} onRetry={() => void load()} />
    </View>
  );
}
```

Add `useCallback`, `AppError`, and `StatePanel` to the detail-page imports. The loader reuses the original route `siteId` and does not change the selected city.

Create `frontend/apps/miniapp/src/components/state-panel.scss`:

```scss
@use "@spl/design-tokens/src/tokens.scss" as *;
.state-panel { padding: 24px 16px; text-align: center; background: $color-surface; border-radius: $radius-card; }
.state-panel__title, .state-panel__impact, .state-panel__trace { display: block; }
.state-panel__title { font-size: 18px; font-weight: 700; }
.state-panel__impact { margin-top: 8px; color: $color-text-secondary; }
.state-panel__trace { margin-top: 8px; color: $color-text-secondary; font-size: 12px; }
.state-panel button { margin-top: 16px; color: #fff; background: $color-primary; border-radius: $radius-button; }
.skeleton { background: #e8edf4; border-radius: 12px; }
.skeleton--title { width: 45%; height: 24px; }
.skeleton--card { height: 160px; margin-top: 16px; }
```

- [ ] **Step 5: Run tests and commit**

Run:

```powershell
Set-Location frontend
npm test
npm run typecheck
npm run build:miniapp
```

Expected: all frontend tests PASS; type check and WeChat build exit 0.

```powershell
git add frontend/apps/miniapp/src/components frontend/apps/miniapp/src/pages
git commit -m "feat: add recoverable site discovery states"
```

### Task 7: Add CI, local smoke instructions, and visual acceptance evidence

**Files:**
- Create: `.github/workflows/frontend.yml`
- Create: `docs/operations/miniapp-local-development.md`
- Create: `frontend/apps/miniapp/tests/visual/home-baseline.md`
- Modify: `README.md`

- [ ] **Step 1: Add frontend CI**

Create `.github/workflows/frontend.yml`:

```yaml
name: frontend
on:
  pull_request:
    paths:
      - "frontend/**"
      - ".github/workflows/frontend.yml"
jobs:
  miniapp:
    runs-on: ubuntu-latest
    defaults:
      run:
        working-directory: frontend
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: 22
          cache: npm
          cache-dependency-path: frontend/package-lock.json
      - run: npm ci
      - run: npm run typecheck
      - run: npm test
      - run: npm run build:miniapp
```

- [ ] **Step 2: Document local development without credentials**

Create `docs/operations/miniapp-local-development.md` with these exact commands:

```powershell
docker compose -f deploy/docker-compose.yml up -d mysql redis api worker
Set-Location frontend
npm ci
$env:TARO_APP_API_BASE_URL = "http://127.0.0.1:8000"
npm run dev:weapp -w @spl/miniapp
```

Document importing `frontend/apps/miniapp` into WeChat Developer Tools. State that the committed `touristappid` is for local preview only and that a real AppID must be supplied through local developer-tool configuration, never committed with secrets.

- [ ] **Step 3: Record the visual acceptance checklist**

Create `frontend/apps/miniapp/tests/visual/home-baseline.md`:

```md
# Home visual baseline

- Viewport: 375×812.
- First-viewport order: city, search, current order, nearby sites, site cards.
- Background: `#F7F9FC`; surfaces: pure `#FFFFFF`; primary: `#1769E0`.
- No greeting hero, skyline, map, marketing badge, cream background, or shipping workflow.
- Current-order empty state is shown until plan 2 connects the order API; no fake active order appears in production runtime.
- Site names, distance, availability, status, and long Chinese address remain readable without horizontal overflow.
```

- [ ] **Step 4: Update the repository README**

Add a `Frontend` section linking to `docs/operations/miniapp-local-development.md` and listing `npm run typecheck`, `npm test`, and `npm run build:miniapp` as required checks.

- [ ] **Step 5: Run final verification**

Run:

```powershell
Set-Location frontend
npm ci
npm run typecheck
npm test
npm run build:miniapp
Set-Location ..
git diff --check
git status --short
```

Expected: all commands exit 0; only the intentional plan-1 files are modified.

Open the built mini program in WeChat Developer Tools, capture page 01 at 375×812 and pages 02–04, and compare them against the approved blue-white concept. Check copy, page order, typography hierarchy, pure-white surfaces, primary blue, spacing, site-card anatomy, and error states. Fix every material mismatch before declaring the plan complete.

- [ ] **Step 6: Commit**

```powershell
git add .github/workflows/frontend.yml docs/operations/miniapp-local-development.md frontend/apps/miniapp/tests/visual/home-baseline.md README.md
git commit -m "ci: verify mini app site discovery"
```

## Plan Completion Gate

Plan 1 is complete only when pages 01–04 are interactive, load real local Kratos site data, preserve the approved visual hierarchy, handle loading/empty/network/offline states, pass CI, and contain no fake active-order data in production runtime.

After plan 1 passes, write plan 2 for pages 05–12 and the required public reservation, order, payment, and device-orchestration APIs. Write plan 3 for page 13, notification delivery, secure scan resolution, door-safety monitoring, the remaining state matrix, and final cross-device visual QA.

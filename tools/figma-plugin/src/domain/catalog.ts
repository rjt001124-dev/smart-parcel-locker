export type Stage = "foundations" | "components" | "mini-app" | "admin-web";

export interface PageCatalogItem {
  key: string;
  name: string;
}

export interface ScreenCatalogItem {
  key: string;
  name: string;
  width: number;
  height: number;
}

export const PAGE_KEYS: readonly PageCatalogItem[] = [
  { key: "page/design-system", name: "00 Design System" },
  { key: "page/mini-app", name: "01 Mini App" },
  { key: "page/admin-web", name: "02 Admin Web" }
];

export const COMPONENT_KEYS = [
  "button",
  "icon-button",
  "input",
  "search",
  "select",
  "status-pill",
  "alert",
  "result",
  "empty-state",
  "error-state",
  "skeleton",
  "card",
  "site-card",
  "locker-size-card",
  "order-card",
  "modal",
  "bottom-sheet",
  "confirm-dialog",
  "miniapp-top-bar",
  "miniapp-bottom-tab-bar",
  "admin-sidebar",
  "admin-header",
  "filter-bar",
  "data-table",
  "pagination"
] as const;

export const MINI_APP_SCREENS: readonly ScreenCatalogItem[] = [
  { key: "screen/miniapp/home", name: "01 首页", width: 375, height: 812 },
  { key: "screen/miniapp/location", name: "02 城市定位", width: 375, height: 812 },
  { key: "screen/miniapp/nearby-sites", name: "03 附近网点", width: 375, height: 812 },
  { key: "screen/miniapp/site-detail", name: "04 网点详情", width: 375, height: 812 },
  { key: "screen/miniapp/locker-duration", name: "05 柜格与时长", width: 375, height: 812 },
  { key: "screen/miniapp/order-confirm", name: "06 订单确认", width: 375, height: 812 },
  { key: "screen/miniapp/payment", name: "07 支付", width: 375, height: 812 },
  { key: "screen/miniapp/store-door", name: "08 存件开门", width: 375, height: 812 },
  { key: "screen/miniapp/orders", name: "09 订单列表", width: 375, height: 812 },
  { key: "screen/miniapp/order-detail", name: "10 订单详情", width: 375, height: 812 },
  { key: "screen/miniapp/overdue-payment", name: "11 超时补缴", width: 375, height: 812 },
  { key: "screen/miniapp/pickup", name: "12 取件", width: 375, height: 812 },
  { key: "screen/miniapp/profile", name: "13 个人中心", width: 375, height: 812 }
];

export const ADMIN_SCREENS: readonly ScreenCatalogItem[] = [
  { key: "screen/admin/login", name: "01 登录与权限", width: 1440, height: 900 },
  { key: "screen/admin/dashboard", name: "02 Dashboard", width: 1440, height: 900 },
  { key: "screen/admin/sites", name: "03 网点管理", width: 1440, height: 900 },
  { key: "screen/admin/devices", name: "04 柜机管理", width: 1440, height: 900 },
  { key: "screen/admin/orders", name: "05 订单管理", width: 1440, height: 900 },
  { key: "screen/admin/payments", name: "06 支付管理", width: 1440, height: 900 },
  { key: "screen/admin/notifications", name: "07 通知管理", width: 1440, height: 900 },
  { key: "screen/admin/permissions", name: "08 权限管理", width: 1440, height: 900 },
  { key: "screen/admin/audit", name: "09 审计日志", width: 1440, height: 900 }
];

export const REQUIRED_STATES = [
  "loading",
  "empty",
  "network-failure",
  "payment-processing",
  "payment-failure",
  "payment-success",
  "cell-contention",
  "device-offline",
  "door-open-failure",
  "door-not-closed",
  "overdue-payment",
  "permission-denied"
] as const;

const STAGE_PREREQUISITES: Readonly<Record<Stage, readonly Stage[]>> = {
  foundations: [],
  components: ["foundations"],
  "mini-app": ["foundations", "components"],
  "admin-web": ["foundations", "components"]
};

export function prerequisitesFor(stage: Stage): readonly Stage[] {
  return STAGE_PREREQUISITES[stage];
}

export function assertUniqueKeys(): void {
  const keys = [
    ...PAGE_KEYS.map((item) => item.key),
    ...COMPONENT_KEYS.map((key) => `component/${key}`),
    ...MINI_APP_SCREENS.map((item) => item.key),
    ...ADMIN_SCREENS.map((item) => item.key),
    ...REQUIRED_STATES.map((key) => `state/${key}`)
  ];
  const duplicates = keys.filter((key, index) => keys.indexOf(key) !== index);
  if (duplicates.length > 0) {
    throw new Error(`Duplicate stable keys: ${[...new Set(duplicates)].join(", ")}`);
  }
}

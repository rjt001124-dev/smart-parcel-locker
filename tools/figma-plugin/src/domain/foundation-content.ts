export const FOUNDATION_SECTIONS = ["设计原则", "颜色", "排版", "间距", "圆角", "阴影", "图标", "状态"] as const;
export type FoundationSection = (typeof FOUNDATION_SECTIONS)[number];
export type FoundationItemKind = "text" | "swatch" | "spacing" | "radius" | "shadow" | "icon" | "status";
export interface FoundationItem { kind: FoundationItemKind; label: string; value?: number; color?: string; description?: string; }

const CONTENT: Record<FoundationSection, readonly FoundationItem[]> = {
  设计原则: [
    { kind: "text", label: "可信", description: "资金、订单与设备状态清晰可追溯" },
    { kind: "text", label: "高效", description: "主操作突出，减少用户决策成本" },
    { kind: "text", label: "可恢复", description: "失败状态始终给出下一步操作" }
  ],
  颜色: [
    { kind: "swatch", label: "品牌蓝", color: "#1677FF" }, { kind: "swatch", label: "成功绿", color: "#12B76A" },
    { kind: "swatch", label: "警告橙", color: "#F79009" }, { kind: "swatch", label: "危险红", color: "#F04438" },
    { kind: "swatch", label: "中性灰", color: "#667085" }
  ],
  排版: [
    { kind: "text", label: "Display 32", value: 32 }, { kind: "text", label: "Heading 24", value: 24 },
    { kind: "text", label: "Body 16", value: 16 }, { kind: "text", label: "Caption 12", value: 12 }
  ],
  间距: [4, 8, 12, 16, 24, 32].map((value) => ({ kind: "spacing" as const, label: `${value}px`, value })),
  圆角: [4, 8, 12, 24, 999].map((value) => ({ kind: "radius" as const, label: value === 999 ? "Full" : `${value}px`, value })),
  阴影: [
    { kind: "shadow", label: "卡片", value: 8 }, { kind: "shadow", label: "悬浮", value: 24 }, { kind: "shadow", label: "弹窗", value: 40 }
  ],
  图标: [
    { kind: "icon", label: "网点" }, { kind: "icon", label: "柜机" }, { kind: "icon", label: "订单" }, { kind: "icon", label: "支付" }, { kind: "icon", label: "通知" }
  ],
  状态: [
    { kind: "status", label: "成功", color: "#12B76A" }, { kind: "status", label: "处理中", color: "#1677FF" },
    { kind: "status", label: "警告", color: "#F79009" }, { kind: "status", label: "失败", color: "#F04438" },
    { kind: "status", label: "离线", color: "#64748B" }
  ]
};

export function foundationItems(section: FoundationSection): readonly FoundationItem[] { return CONTENT[section]; }

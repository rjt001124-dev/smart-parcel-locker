import { OWNERSHIP } from "../domain/ownership";
import type { ResourceResult } from "../domain/run-report";
import { normalizeColor, sectionVisualSpec, toVariableScopes } from "./adapter-helpers";
import type { FigmaPort, FontDescriptor, ResourceSpec } from "./port";
import type { Stage } from "../domain/catalog";
import { FOUNDATION_SECTIONS, foundationItems, type FoundationSection } from "../domain/foundation-content";

const NS = OWNERSHIP.namespace;

function mark(resource: PluginDataMixin, key: string): void {
  resource.setSharedPluginData(NS, "owner", OWNERSHIP.owner);
  resource.setSharedPluginData(NS, "key", key);
  resource.setSharedPluginData(NS, "schema_version", OWNERSHIP.schemaVersion);
}

function result(key: string, outcome: ResourceResult["outcome"], id?: string, message?: string): ResourceResult {
  return { key, outcome, nodeId: id, message };
}

function owned(resource: PluginDataMixin, key: string): boolean {
  return resource.getSharedPluginData(NS, "owner") === OWNERSHIP.owner && resource.getSharedPluginData(NS, "key") === key;
}

export class RealFigmaAdapter implements FigmaPort {
  async getFileName(): Promise<string> { return figma.root.name; }
  async listAvailableFonts(): Promise<readonly FontDescriptor[]> { return (await figma.listAvailableFontsAsync()).map(({ fontName }) => fontName); }
  async hasStageMarker(stage: Stage): Promise<boolean> { return figma.root.getSharedPluginData(NS, `stage/${stage}`) === "complete"; }
  async setStageMarker(stage: Stage): Promise<void> { figma.root.setSharedPluginData(NS, `stage/${stage}`, "complete"); }

  async upsertPage(spec: ResourceSpec): Promise<ResourceResult> {
    const byKey = figma.root.children.find((page) => owned(page, spec.key));
    if (byKey) { byKey.name = spec.name; return result(spec.key, "updated", byKey.id); }
    if (spec.key === "page/design-system" && figma.root.children.length === 1 && figma.root.children[0].name === "Page 1") {
      const initialPage = figma.root.children[0];
      initialPage.name = spec.name;
      mark(initialPage, spec.key);
      return result(spec.key, "updated", initialPage.id, "Reused the initial free-plan page");
    }
    const sameName = figma.root.children.find((page) => page.name === spec.name);
    if (sameName) return result(spec.key, "conflict", sameName.id, `Unowned page already exists: ${spec.name}`);
    const page = figma.createPage(); page.name = spec.name; mark(page, spec.key);
    return result(spec.key, "created", page.id);
  }

  async upsertVariableCollection(spec: ResourceSpec): Promise<ResourceResult> {
    const collections = await figma.variables.getLocalVariableCollectionsAsync();
    const byKey = collections.find((item) => owned(item, spec.key));
    if (byKey) { byKey.name = spec.name; return result(spec.key, "updated", byKey.id); }
    const sameName = collections.find((item) => item.name === spec.name);
    if (sameName) return result(spec.key, "conflict", sameName.id, `Unowned variable collection exists: ${spec.name}`);
    const collection = figma.variables.createVariableCollection(spec.name); mark(collection, spec.key);
    return result(spec.key, "created", collection.id);
  }

  async upsertVariable(spec: ResourceSpec): Promise<ResourceResult> {
    const variables = await figma.variables.getLocalVariablesAsync();
    let variable = variables.find((item) => owned(item, spec.key));
    let outcome: ResourceResult["outcome"] = "updated";
    if (!variable) {
      const sameName = variables.find((item) => item.name === spec.name);
      if (sameName) return result(spec.key, "conflict", sameName.id, `Unowned variable exists: ${spec.name}`);
      const collections = await figma.variables.getLocalVariableCollectionsAsync();
      const collection = collections.find((item) => owned(item, String(spec.collectionKey)));
      if (!collection) return result(spec.key, "error", undefined, `Missing collection: ${String(spec.collectionKey)}`);
      variable = figma.variables.createVariable(spec.name, collection, spec.type as VariableResolvedDataType);
      mark(variable, spec.key); outcome = "created";
    }
    variable.name = spec.name;
    variable.scopes = toVariableScopes((spec.scopes as readonly string[] | undefined) ?? []);
    const collection = await figma.variables.getVariableCollectionByIdAsync(variable.variableCollectionId);
    if (!collection) return result(spec.key, "error", variable.id, "Variable collection unavailable");
    let value = spec.value as VariableValue | undefined;
    if (spec.type === "COLOR" && value && typeof value === "object" && "r" in value) value = normalizeColor(value as RGB);
    if (spec.alias) {
      const target = variables.find((item) => item.name === spec.alias);
      if (!target) return result(spec.key, "error", variable.id, `Missing alias target: ${String(spec.alias)}`);
      value = figma.variables.createVariableAlias(target);
    }
    if (value !== undefined) variable.setValueForMode(collection.defaultModeId, value);
    if (spec.codeSyntax) variable.setVariableCodeSyntax("WEB", String(spec.codeSyntax));
    return result(spec.key, outcome, variable.id);
  }

  async upsertTextStyle(spec: ResourceSpec): Promise<ResourceResult> {
    const styles = await figma.getLocalTextStylesAsync();
    let style = styles.find((item) => owned(item, spec.key));
    let outcome: ResourceResult["outcome"] = "updated";
    if (!style) {
      const sameName = styles.find((item) => item.name === spec.name);
      if (sameName) return result(spec.key, "conflict", sameName.id, `Unowned text style exists: ${spec.name}`);
      style = figma.createTextStyle(); mark(style, spec.key); outcome = "created";
    }
    const fontName = { family: String(spec.family), style: String(spec.style) };
    await figma.loadFontAsync(fontName); style.name = spec.name; style.fontName = fontName;
    style.fontSize = Number(spec.size); style.lineHeight = { unit: "PIXELS", value: Number(spec.lineHeight) };
    return result(spec.key, outcome, style.id);
  }

  async upsertEffectStyle(spec: ResourceSpec): Promise<ResourceResult> {
    const styles = await figma.getLocalEffectStylesAsync();
    let style = styles.find((item) => owned(item, spec.key));
    let outcome: ResourceResult["outcome"] = "updated";
    if (!style) { const sameName = styles.find((item) => item.name === spec.name); if (sameName) return result(spec.key, "conflict", sameName.id); style = figma.createEffectStyle(); mark(style, spec.key); outcome = "created"; }
    style.name = spec.name;
    style.effects = [{ type: "DROP_SHADOW", color: { r: 0.06, g: 0.09, b: 0.16, a: Number(spec.opacity) }, offset: { x: Number(spec.x), y: Number(spec.y) }, radius: Number(spec.blur), spread: Number(spec.spread ?? 0), visible: true, blendMode: "NORMAL" }];
    return result(spec.key, outcome, style.id);
  }

  async upsertComponentFamily(spec: ResourceSpec): Promise<ResourceResult> {
    const page = await this.pageFor(String(spec.pageKey)); if (!page) return result(spec.key, "error", undefined, "Component page missing");
    await figma.setCurrentPageAsync(page);
    let component = page.findAllWithCriteria({ types: ["COMPONENT"] }).find((item) => owned(item, spec.key));
    let outcome: ResourceResult["outcome"] = "updated";
    if (!component) { const sameName = page.findAllWithCriteria({ types: ["COMPONENT"] }).find((item) => item.name === spec.name); if (sameName) return result(spec.key, "conflict", sameName.id); component = figma.createComponent(); mark(component, spec.key); page.appendChild(component); outcome = "created"; }
    component.name = spec.name;
    await this.renderComponent(component, spec);
    this.positionNode(page, component, spec);
    return result(spec.key, outcome, component.id);
  }

  async upsertScreen(spec: ResourceSpec): Promise<ResourceResult> {
    const page = await this.pageFor(String(spec.pageKey)); if (!page) return result(spec.key, "error", undefined, "Screen page missing");
    await figma.setCurrentPageAsync(page);
    let frame = page.findAllWithCriteria({ types: ["FRAME"] }).find((item) => owned(item, spec.key));
    let outcome: ResourceResult["outcome"] = "updated";
    if (!frame) { const sameName = page.findAllWithCriteria({ types: ["FRAME"] }).find((item) => item.name === spec.name); if (sameName) return result(spec.key, "conflict", sameName.id); frame = figma.createFrame(); mark(frame, spec.key); page.appendChild(frame); outcome = "created"; }
    frame.name = spec.name; frame.resize(Number(spec.width), Number(spec.height)); frame.fills = [{ type: "SOLID", color: { r: 0.97, g: 0.98, b: 1 } }]; frame.clipsContent = true;
    const sections = (spec.sections as readonly string[] | undefined) ?? (spec.states as readonly string[] | undefined) ?? [];
    if (spec.navigationTheme === "dark") await this.renderAdminScreen(frame, spec.name, sections);
    else {
      frame.layoutMode = "VERTICAL"; frame.primaryAxisSizingMode = "FIXED"; frame.counterAxisSizingMode = "FIXED";
      frame.paddingTop = 24; frame.paddingBottom = 24; frame.paddingLeft = 24; frame.paddingRight = 24; frame.itemSpacing = 16;
      await this.ensureLabel(frame, spec.name); await this.ensureSections(frame, sections); await this.ensurePrimaryAction(frame, String(spec.key).includes("state-matrix") ? "查看状态处理规范" : "继续");
    }
    this.positionNode(page, frame, spec);
    return result(spec.key, outcome, frame.id);
  }

  async focusPage(key: string): Promise<void> { const page = await this.pageFor(key); if (page) await figma.setCurrentPageAsync(page); }

  private async pageFor(key: string): Promise<PageNode | undefined> { return figma.root.children.find((page) => owned(page, key)); }
  private async ensureLabel(parent: ChildrenMixin, text: string): Promise<void> {
    await figma.loadFontAsync({ family: "Inter", style: "Semi Bold" });
    let label = parent.children.find((child): child is TextNode => child.type === "TEXT" && child.name === "Generated Label");
    if (!label) { label = figma.createText(); label.name = "Generated Label"; parent.appendChild(label); }
    label.fontName = { family: "Inter", style: "Semi Bold" }; label.fontSize = 16; label.characters = text;
    if (!("layoutMode" in parent) || parent.layoutMode === "NONE") { label.x = 24; label.y = 24; }
  }
  private async renderComponent(component: ComponentNode, spec: ResourceSpec): Promise<void> {
    await figma.loadFontAsync({ family: "Inter", style: "Regular" });
    await figma.loadFontAsync({ family: "Inter", style: "Medium" });
    await figma.loadFontAsync({ family: "Inter", style: "Semi Bold" });
    for (const child of [...component.children]) child.remove();
    const kind = String(spec.visualKind ?? "generic"), width = Number(spec.width ?? 280), height = Number(spec.height ?? 112);
    component.layoutMode = kind === "button" || kind === "icon-button" ? "HORIZONTAL" : "VERTICAL";
    component.primaryAxisSizingMode = "FIXED"; component.counterAxisSizingMode = "FIXED"; component.resize(width, height);
    component.paddingTop = 16; component.paddingBottom = 16; component.paddingLeft = 16; component.paddingRight = 16; component.itemSpacing = 10; component.cornerRadius = 12;
    component.fills = [{ type: "SOLID", color: kind === "button" ? { r: 0.09, g: 0.47, b: 1 } : kind === "admin-sidebar" ? { r: 0.06, g: 0.1, b: 0.2 } : { r: 1, g: 1, b: 1 } }];
    component.strokes = kind === "button" ? [] : [{ type: "SOLID", color: { r: 0.82, g: 0.86, b: 0.92 } }]; component.strokeWeight = kind === "button" ? 0 : 1;
    if (kind === "button") { component.primaryAxisAlignItems = "CENTER"; component.counterAxisAlignItems = "CENTER"; await this.addText(component, "立即寄存", 15, "Semi Bold", { r: 1, g: 1, b: 1 }); return; }
    if (kind === "icon-button") { component.primaryAxisAlignItems = "CENTER"; component.counterAxisAlignItems = "CENTER"; const icon = figma.createEllipse(); icon.resize(18, 18); icon.fills = [{ type: "SOLID", color: { r: 0.09, g: 0.47, b: 1 } }]; component.appendChild(icon); return; }
    if (["input", "search", "select"].includes(kind)) {
      await this.addText(component, kind === "search" ? "搜索网点" : kind === "select" ? "选择城市" : "手机号码", 13, "Medium", { r: 0.2, g: 0.25, b: 0.33 });
      const field = await this.addSurface(component, 44, { r: 0.97, g: 0.98, b: 1 }, 8); await this.addText(field, kind === "search" ? "输入网点名称或地址" : kind === "select" ? "请选择" : "请输入手机号", 13, "Regular", { r: 0.4, g: 0.44, b: 0.52 }); return;
    }
    if (["site-card", "locker-size-card", "order-card", "card"].includes(kind)) {
      await this.addText(component, kind === "site-card" ? "万象城智能寄存点" : kind === "locker-size-card" ? "中号柜格" : kind === "order-card" ? "寄存订单 SPL20260717001" : "信息卡片", 17, "Semi Bold", { r: 0.06, g: 0.16, b: 0.3 });
      await this.addText(component, kind === "site-card" ? "距你 320m · 营业至 22:00" : kind === "locker-size-card" ? "40 × 45 × 50cm · 剩余 8 格" : kind === "order-card" ? "万象城站点 · A-08 柜" : "用于承载关键业务信息", 13, "Regular", { r: 0.4, g: 0.44, b: 0.52 });
      const row = await this.addSurface(component, 44, { r: 0.9, g: 0.95, b: 1 }, 8); await this.addText(row, kind === "site-card" ? "可用 24 格   立即选择 →" : kind === "locker-size-card" ? "¥3/小时   选择柜格 →" : "查看详情 →", 13, "Medium", { r: 0.05, g: 0.35, b: 0.8 }); return;
    }
    if (kind === "data-table") {
      await this.addText(component, "订单数据表", 17, "Semi Bold", { r: 0.06, g: 0.16, b: 0.3 });
      for (const [index, line] of ["订单号        用户        状态        金额", "SPL001      张先生      寄存中      ¥18.00", "SPL002      李女士      待取件      ¥12.00", "SPL003      王先生      已完成      ¥24.00"].entries()) {
        const row = await this.addSurface(component, 42, index === 0 ? { r: 0.9, g: 0.95, b: 1 } : { r: 1, g: 1, b: 1 }, 4); await this.addText(row, line, 13, index === 0 ? "Medium" : "Regular", { r: 0.15, g: 0.2, b: 0.28 });
      } return;
    }
    if (kind === "admin-sidebar") {
      await this.addText(component, "智能快递柜", 18, "Semi Bold", { r: 1, g: 1, b: 1 });
      for (const item of ["概览", "网点管理", "柜机管理", "订单管理", "支付管理", "审计日志"]) await this.addText(component, item, 14, item === "概览" ? "Medium" : "Regular", item === "概览" ? { r: 0.35, g: 0.65, b: 1 } : { r: 0.75, g: 0.8, b: 0.9 }); return;
    }
    if (["status-pill", "alert", "result", "empty-state", "error-state", "skeleton"].includes(kind)) {
      const content: Record<string, [string, string, RGB]> = {
        "status-pill": ["寄存中", "订单状态正常", { r: 0.09, g: 0.47, b: 1 }], alert: ["设备暂时离线", "请选择其他网点或稍后重试", { r: 0.96, g: 0.56, b: 0.04 }],
        result: ["支付成功", "正在为你分配柜格", { r: 0.07, g: 0.72, b: 0.42 }], "empty-state": ["暂无订单", "完成寄存后可在这里查看", { r: 0.4, g: 0.44, b: 0.52 }],
        "error-state": ["开门失败", "请重试或联系现场客服", { r: 0.94, g: 0.27, b: 0.22 }], skeleton: ["加载中", "正在获取最新数据", { r: 0.55, g: 0.62, b: 0.72 }]
      };
      const [title, detail, color] = content[kind]; const marker = figma.createEllipse(); marker.resize(24, 24); marker.fills = [{ type: "SOLID", color }]; component.appendChild(marker); await this.addText(component, title, 15, "Semi Bold", { r: 0.06, g: 0.16, b: 0.3 }); await this.addText(component, detail, 12, "Regular", { r: 0.4, g: 0.44, b: 0.52 }); return;
    }
    if (["modal", "bottom-sheet", "confirm-dialog"].includes(kind)) {
      await this.addText(component, kind === "confirm-dialog" ? "确认远程开门？" : kind === "bottom-sheet" ? "选择寄存时长" : "订单信息", 18, "Semi Bold", { r: 0.06, g: 0.16, b: 0.3 });
      await this.addText(component, kind === "confirm-dialog" ? "该操作将写入审计日志，请填写操作原因。" : "请确认信息后继续操作。", 13, "Regular", { r: 0.4, g: 0.44, b: 0.52 });
      const action = await this.addSurface(component, 48, { r: 0.09, g: 0.47, b: 1 }, 8); action.primaryAxisAlignItems = "CENTER"; action.counterAxisAlignItems = "CENTER"; await this.addText(action, "确认", 14, "Semi Bold", { r: 1, g: 1, b: 1 }); return;
    }
    if (["miniapp-top-bar", "miniapp-bottom-tab-bar"].includes(kind)) {
      component.layoutMode = "HORIZONTAL"; component.counterAxisAlignItems = "CENTER"; component.primaryAxisAlignItems = "SPACE_BETWEEN";
      for (const item of kind === "miniapp-top-bar" ? ["‹", "智能寄存", "•••"] : ["首页", "订单", "我的"]) await this.addText(component, item, 14, item === "首页" || item === "智能寄存" ? "Semi Bold" : "Regular", { r: 0.06, g: 0.16, b: 0.3 }); return;
    }
    if (["admin-header", "filter-bar", "pagination"].includes(kind)) {
      component.layoutMode = "HORIZONTAL"; component.counterAxisAlignItems = "CENTER"; component.primaryAxisAlignItems = "SPACE_BETWEEN";
      const items = kind === "admin-header" ? ["运营概览", "消息 8", "管理员"] : kind === "filter-bar" ? ["全部城市 ▾", "全部状态 ▾", "搜索", "查询"] : ["‹ 上一页", "1", "2", "3", "下一页 ›"];
      for (const item of items) await this.addText(component, item, 13, item === "查询" || item === "1" ? "Medium" : "Regular", { r: item === "查询" || item === "1" ? 0.09 : 0.2, g: item === "查询" || item === "1" ? 0.47 : 0.25, b: item === "查询" || item === "1" ? 1 : 0.33 }); return;
    }
    await this.addText(component, spec.name, 16, "Semi Bold", { r: 0.06, g: 0.16, b: 0.3 });
    await this.addText(component, "Default · Loading · Disabled", 12, "Regular", { r: 0.4, g: 0.44, b: 0.52 });
  }
  private async addText(parent: ChildrenMixin, characters: string, size: number, style: "Regular" | "Medium" | "Semi Bold", color: RGB): Promise<TextNode> {
    const text = figma.createText(); text.fontName = { family: "Inter", style }; text.fontSize = size; text.characters = characters; text.fills = [{ type: "SOLID", color }]; parent.appendChild(text); return text;
  }
  private async addSurface(parent: ChildrenMixin, height: number, color: RGB, radius: number): Promise<FrameNode> {
    const frame = figma.createFrame(); frame.layoutMode = "HORIZONTAL"; frame.primaryAxisSizingMode = "FIXED"; frame.counterAxisSizingMode = "FIXED"; frame.resize(100, height); frame.paddingLeft = 12; frame.paddingRight = 12; frame.paddingTop = 10; frame.paddingBottom = 10; frame.fills = [{ type: "SOLID", color }]; frame.cornerRadius = radius; parent.appendChild(frame);
    if ("layoutMode" in parent && parent.layoutMode !== "NONE") frame.layoutSizingHorizontal = "FILL"; return frame;
  }
  private async ensureSections(parent: FrameNode, sections: readonly string[]): Promise<void> {
    for (const child of [...parent.children]) if (child.name.startsWith("Generated Section/")) child.remove();
    await figma.loadFontAsync({ family: "Inter", style: "Regular" });
    for (const [index, sectionName] of sections.entries()) {
      const section = figma.createFrame(); section.name = `Generated Section/${sectionName}`;
      const visual = sectionVisualSpec(parent.width, index);
      const sectionHeight = parent.width > 500 && parent.height <= 900 ? 120 : visual.height;
      section.layoutMode = "VERTICAL"; section.primaryAxisSizingMode = "FIXED"; section.counterAxisSizingMode = "FIXED";
      section.paddingTop = 20; section.paddingBottom = 20; section.paddingLeft = 20; section.paddingRight = 20; section.itemSpacing = 12;
      section.fills = [{ type: "SOLID", color: visual.fill }]; section.strokes = [{ type: "SOLID", color: visual.stroke }]; section.strokeWeight = visual.strokeWeight; section.cornerRadius = 12;
      parent.appendChild(section); section.resize(Math.max(240, parent.width - 48), sectionHeight); section.layoutSizingHorizontal = "FILL"; section.layoutSizingVertical = "FIXED";
      const [title, detail] = sectionName.split("｜");
      const text = figma.createText(); text.fontName = { family: "Inter", style: "Medium" }; text.fontSize = 18; text.characters = title; text.fills = [{ type: "SOLID", color: { r: 0.06, g: 0.16, b: 0.3 } }]; section.appendChild(text);
      if (detail) { const description = figma.createText(); description.fontName = { family: "Inter", style: "Regular" }; description.fontSize = 13; description.characters = detail; description.fills = [{ type: "SOLID", color: { r: 0.4, g: 0.44, b: 0.52 } }]; section.appendChild(description); }
      if ((FOUNDATION_SECTIONS as readonly string[]).includes(sectionName)) await this.addFoundationExamples(section, sectionName as FoundationSection);
    }
  }
  private async addFoundationExamples(section: FrameNode, sectionName: FoundationSection): Promise<void> {
    await figma.loadFontAsync({ family: "Inter", style: "Medium" });
    const row = figma.createFrame(); row.name = `Generated Examples/${sectionName}`; row.layoutMode = "HORIZONTAL"; row.itemSpacing = 12; row.fills = [];
    section.appendChild(row); row.layoutSizingHorizontal = "FILL"; row.layoutSizingVertical = "HUG";
    for (const item of foundationItems(sectionName)) {
      const card = figma.createFrame(); card.name = `Example/${item.label}`; card.layoutMode = "VERTICAL"; card.primaryAxisSizingMode = "FIXED"; card.counterAxisSizingMode = "FIXED";
      card.resize(sectionName === "设计原则" ? 260 : 150, 74); card.paddingTop = 12; card.paddingBottom = 12; card.paddingLeft = 12; card.paddingRight = 12; card.itemSpacing = 6; card.cornerRadius = item.kind === "radius" ? Math.min(Number(item.value), 36) : 8;
      card.fills = [{ type: "SOLID", color: item.color ? this.hexColor(item.color) : { r: 0.95, g: 0.97, b: 1 } }];
      if (item.kind === "shadow") card.effects = [{ type: "DROP_SHADOW", color: { r: 0.06, g: 0.09, b: 0.16, a: 0.16 }, offset: { x: 0, y: 6 }, radius: Number(item.value), spread: 0, visible: true, blendMode: "NORMAL" }];
      row.appendChild(card);
      const label = figma.createText(); label.fontName = { family: "Inter", style: "Medium" }; label.fontSize = item.kind === "text" && item.value ? Math.min(item.value, 24) : 14; label.characters = item.label;
      label.fills = [{ type: "SOLID", color: item.color ? { r: 1, g: 1, b: 1 } : { r: 0.06, g: 0.16, b: 0.3 } }]; card.appendChild(label);
      if (item.description) { const detail = figma.createText(); detail.fontName = { family: "Inter", style: "Regular" }; detail.fontSize = 11; detail.characters = item.description; detail.fills = [{ type: "SOLID", color: { r: 0.4, g: 0.44, b: 0.52 } }]; card.appendChild(detail); }
      if (item.kind === "spacing") { const bar = figma.createRectangle(); bar.resize(Math.max(8, Number(item.value) * 3), 8); bar.cornerRadius = 4; bar.fills = [{ type: "SOLID", color: { r: 0.09, g: 0.47, b: 1 } }]; card.appendChild(bar); }
    }
  }
  private async ensurePrimaryAction(parent: FrameNode, label: string): Promise<void> {
    for (const child of [...parent.children]) if (child.name === "Generated Primary Action") child.remove();
    const button = figma.createFrame(); button.name = "Generated Primary Action"; button.layoutMode = "HORIZONTAL"; button.primaryAxisAlignItems = "CENTER"; button.counterAxisAlignItems = "CENTER"; button.resize(Math.max(240, parent.width - 48), 48); button.cornerRadius = 10; button.fills = [{ type: "SOLID", color: { r: 0.09, g: 0.47, b: 1 } }]; parent.appendChild(button); button.layoutSizingHorizontal = "FILL"; button.layoutSizingVertical = "FIXED";
    await this.addText(button, label, 15, "Semi Bold", { r: 1, g: 1, b: 1 });
  }
  private async renderAdminScreen(frame: FrameNode, title: string, sections: readonly string[]): Promise<void> {
    for (const child of [...frame.children]) child.remove();
    frame.layoutMode = "HORIZONTAL"; frame.primaryAxisSizingMode = "FIXED"; frame.counterAxisSizingMode = "FIXED"; frame.paddingTop = 0; frame.paddingBottom = 0; frame.paddingLeft = 0; frame.paddingRight = 0; frame.itemSpacing = 0;
    const sidebar = figma.createFrame(); sidebar.name = "Generated Admin Sidebar"; sidebar.layoutMode = "VERTICAL"; sidebar.resize(220, frame.height); sidebar.paddingTop = 28; sidebar.paddingBottom = 28; sidebar.paddingLeft = 20; sidebar.paddingRight = 20; sidebar.itemSpacing = 18; sidebar.fills = [{ type: "SOLID", color: { r: 0.06, g: 0.1, b: 0.2 } }]; frame.appendChild(sidebar); sidebar.layoutSizingVertical = "FILL";
    await this.addText(sidebar, "智能快递柜", 18, "Semi Bold", { r: 1, g: 1, b: 1 });
    for (const item of ["概览", "网点管理", "柜机管理", "订单管理", "支付管理", "通知管理", "权限管理", "审计日志"]) await this.addText(sidebar, item, 14, item === "概览" ? "Medium" : "Regular", item === "概览" ? { r: 0.35, g: 0.65, b: 1 } : { r: 0.72, g: 0.78, b: 0.88 });
    const main = figma.createFrame(); main.name = "Generated Admin Content"; main.layoutMode = "VERTICAL"; main.primaryAxisSizingMode = "FIXED"; main.counterAxisSizingMode = "FIXED"; main.resize(frame.width - 220, frame.height); main.paddingTop = 24; main.paddingBottom = 24; main.paddingLeft = 24; main.paddingRight = 24; main.itemSpacing = 16; main.fills = [{ type: "SOLID", color: { r: 0.97, g: 0.98, b: 1 } }]; frame.appendChild(main); main.layoutSizingHorizontal = "FILL"; main.layoutSizingVertical = "FILL";
    await this.ensureLabel(main, title); await this.ensureSections(main, sections);
  }
  private hexColor(hex: string): RGB {
    const value = hex.replace("#", "");
    return { r: Number.parseInt(value.slice(0, 2), 16) / 255, g: Number.parseInt(value.slice(2, 4), 16) / 255, b: Number.parseInt(value.slice(4, 6), 16) / 255 };
  }
  private positionNode(page: PageNode, node: SceneNode, spec: ResourceSpec): void {
    if (typeof spec.x === "number" && typeof spec.y === "number") { node.x = spec.x; node.y = spec.y; return; }
    const others = page.children.filter((item) => item.id !== node.id && "x" in item) as SceneNode[];
    const right = others.reduce((max, item) => Math.max(max, item.x + item.width), 0);
    if (node.x === 0 && node.y === 0) { node.x = right + 80; node.y = 80; }
  }
}

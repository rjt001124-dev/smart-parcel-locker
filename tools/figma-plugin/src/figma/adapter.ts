import { OWNERSHIP } from "../domain/ownership";
import type { ResourceResult } from "../domain/run-report";
import { normalizeColor, resourceDecision, toVariableScopes } from "./adapter-helpers";
import type { FigmaPort, FontDescriptor, ResourceSpec } from "./port";
import type { Stage } from "../domain/catalog";

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
    component.name = spec.name; component.resize(240, 72); component.fills = [{ type: "SOLID", color: { r: 1, g: 1, b: 1 } }]; component.cornerRadius = 8;
    await this.ensureLabel(component, spec.name);
    this.placeOnPage(page, component);
    return result(spec.key, outcome, component.id);
  }

  async upsertScreen(spec: ResourceSpec): Promise<ResourceResult> {
    const page = await this.pageFor(String(spec.pageKey)); if (!page) return result(spec.key, "error", undefined, "Screen page missing");
    await figma.setCurrentPageAsync(page);
    let frame = page.findAllWithCriteria({ types: ["FRAME"] }).find((item) => owned(item, spec.key));
    let outcome: ResourceResult["outcome"] = "updated";
    if (!frame) { const sameName = page.findAllWithCriteria({ types: ["FRAME"] }).find((item) => item.name === spec.name); if (sameName) return result(spec.key, "conflict", sameName.id); frame = figma.createFrame(); mark(frame, spec.key); page.appendChild(frame); outcome = "created"; }
    frame.name = spec.name; frame.resize(Number(spec.width), Number(spec.height)); frame.fills = [{ type: "SOLID", color: { r: 0.97, g: 0.98, b: 1 } }]; frame.clipsContent = true;
    await this.ensureLabel(frame, spec.name); this.placeOnPage(page, frame);
    return result(spec.key, outcome, frame.id);
  }

  async focusPage(key: string): Promise<void> { const page = await this.pageFor(key); if (page) await figma.setCurrentPageAsync(page); }

  private async pageFor(key: string): Promise<PageNode | undefined> { return figma.root.children.find((page) => owned(page, key)); }
  private async ensureLabel(parent: ChildrenMixin, text: string): Promise<void> {
    await figma.loadFontAsync({ family: "Inter", style: "Semi Bold" });
    let label = parent.children.find((child): child is TextNode => child.type === "TEXT" && child.name === "Generated Label");
    if (!label) { label = figma.createText(); label.name = "Generated Label"; parent.appendChild(label); }
    label.fontName = { family: "Inter", style: "Semi Bold" }; label.fontSize = 16; label.characters = text; label.x = 24; label.y = 24;
  }
  private placeOnPage(page: PageNode, node: SceneNode): void {
    const others = page.children.filter((item) => item.id !== node.id && "x" in item) as SceneNode[];
    const right = others.reduce((max, item) => Math.max(max, item.x + item.width), 0);
    if (node.x === 0 && node.y === 0) { node.x = right + 80; node.y = 80; }
  }
}

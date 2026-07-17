import type { Stage } from "../src/domain/catalog";
import type { ResourceResult } from "../src/domain/run-report";
import type { FigmaPort, FontDescriptor, ResourceSpec } from "../src/figma/port";

export class FakeFigmaPort implements FigmaPort {
  private readonly markers = new Set<Stage>();
  private readonly resources = new Map<string, ResourceSpec>();
  private fonts: readonly FontDescriptor[] = [
    { family: "Inter", style: "Regular" },
    { family: "Inter", style: "Medium" },
    { family: "Inter", style: "Semi Bold" },
    { family: "Inter", style: "Bold" }
  ];

  async getFileName(): Promise<string> {
    return "Smart Parcel Locker / 智能快递柜";
  }

  async listAvailableFonts(): Promise<readonly FontDescriptor[]> {
    return this.fonts;
  }

  setFonts(fonts: readonly FontDescriptor[]): void { this.fonts = fonts; }

  async hasStageMarker(stage: Stage): Promise<boolean> {
    return this.markers.has(stage);
  }

  async setStageMarker(stage: Stage): Promise<void> {
    this.markers.add(stage);
  }

  private async upsert(spec: ResourceSpec): Promise<ResourceResult> {
    const exists = this.resources.has(spec.key);
    this.resources.set(spec.key, spec);
    return { key: spec.key, outcome: exists ? "updated" : "created", nodeId: `fake:${spec.key}` };
  }

  upsertPage(spec: ResourceSpec): Promise<ResourceResult> { return this.upsert(spec); }
  upsertVariableCollection(spec: ResourceSpec): Promise<ResourceResult> { return this.upsert(spec); }
  upsertVariable(spec: ResourceSpec): Promise<ResourceResult> { return this.upsert(spec); }
  upsertTextStyle(spec: ResourceSpec): Promise<ResourceResult> { return this.upsert(spec); }
  upsertEffectStyle(spec: ResourceSpec): Promise<ResourceResult> { return this.upsert(spec); }
  upsertComponentFamily(spec: ResourceSpec): Promise<ResourceResult> { return this.upsert(spec); }
  upsertScreen(spec: ResourceSpec): Promise<ResourceResult> { return this.upsert(spec); }
  async focusPage(_key: string): Promise<void> {}

  resourceCount(): number { return this.resources.size; }
  componentKeys(): string[] { return [...this.resources.keys()].filter((key) => key.startsWith("component/")); }
}

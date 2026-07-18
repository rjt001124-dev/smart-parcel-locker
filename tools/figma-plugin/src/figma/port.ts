import type { Stage } from "../domain/catalog";
import type { ResourceResult } from "../domain/run-report";

export interface ResourceSpec {
  key: string;
  name: string;
  [property: string]: unknown;
}

export interface FontDescriptor {
  family: string;
  style: string;
}

export interface FigmaPort {
  getFileName(): Promise<string>;
  listAvailableFonts(): Promise<readonly FontDescriptor[]>;
  hasStageMarker(stage: Stage): Promise<boolean>;
  setStageMarker(stage: Stage): Promise<void>;
  upsertPage(spec: ResourceSpec): Promise<ResourceResult>;
  upsertVariableCollection(spec: ResourceSpec): Promise<ResourceResult>;
  upsertVariable(spec: ResourceSpec): Promise<ResourceResult>;
  upsertTextStyle(spec: ResourceSpec): Promise<ResourceResult>;
  upsertEffectStyle(spec: ResourceSpec): Promise<ResourceResult>;
  upsertComponentFamily(spec: ResourceSpec): Promise<ResourceResult>;
  upsertScreen(spec: ResourceSpec): Promise<ResourceResult>;
  focusPage(key: string): Promise<void>;
}

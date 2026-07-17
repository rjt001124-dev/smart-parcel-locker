export interface RGBValue {
  r: number;
  g: number;
  b: number;
}

export type ColorScope = "FRAME_FILL" | "SHAPE_FILL" | "TEXT_FILL" | "STROKE_COLOR";

export interface ColorToken {
  name: string;
  value?: RGBValue;
  alias?: string;
  scopes: readonly ColorScope[];
}

export interface DimensionToken {
  name: string;
  value: number;
  scopes: readonly ("GAP" | "WIDTH_HEIGHT" | "CORNER_RADIUS" | "STROKE_FLOAT")[];
}

function rgb(hex: string): RGBValue {
  const value = hex.replace("#", "");
  return {
    r: Number.parseInt(value.slice(0, 2), 16) / 255,
    g: Number.parseInt(value.slice(2, 4), 16) / 255,
    b: Number.parseInt(value.slice(4, 6), 16) / 255
  };
}

const fillScopes = ["FRAME_FILL", "SHAPE_FILL"] as const;
const textScope = ["TEXT_FILL"] as const;
const strokeScope = ["STROKE_COLOR"] as const;

export const COLOR_TOKENS: readonly ColorToken[] = [
  { name: "primitive/blue/50", value: rgb("E6F4FF"), scopes: fillScopes },
  { name: "primitive/blue/100", value: rgb("BAE0FF"), scopes: fillScopes },
  { name: "primitive/blue/500", value: rgb("1677FF"), scopes: fillScopes },
  { name: "primitive/blue/600", value: rgb("0958D9"), scopes: fillScopes },
  { name: "primitive/gray/0", value: rgb("FFFFFF"), scopes: fillScopes },
  { name: "primitive/gray/50", value: rgb("F7F9FC"), scopes: fillScopes },
  { name: "primitive/gray/100", value: rgb("F0F2F5"), scopes: fillScopes },
  { name: "primitive/gray/300", value: rgb("D0D5DD"), scopes: fillScopes },
  { name: "primitive/gray/500", value: rgb("667085"), scopes: fillScopes },
  { name: "primitive/gray/700", value: rgb("344054"), scopes: fillScopes },
  { name: "primitive/gray/900", value: rgb("101828"), scopes: fillScopes },
  { name: "primitive/green/500", value: rgb("12B76A"), scopes: fillScopes },
  { name: "primitive/orange/500", value: rgb("F79009"), scopes: fillScopes },
  { name: "primitive/red/500", value: rgb("F04438"), scopes: fillScopes },
  { name: "primitive/slate/500", value: rgb("64748B"), scopes: fillScopes },
  { name: "color/bg/canvas", alias: "primitive/gray/50", scopes: fillScopes },
  { name: "color/bg/surface", alias: "primitive/gray/0", scopes: fillScopes },
  { name: "color/bg/primary", alias: "primitive/blue/500", scopes: fillScopes },
  { name: "color/bg/navigation", alias: "primitive/gray/900", scopes: fillScopes },
  { name: "color/action/primary", alias: "primitive/blue/500", scopes: fillScopes },
  { name: "color/action/primary-hover", alias: "primitive/blue/600", scopes: fillScopes },
  { name: "color/action/disabled", alias: "primitive/gray/300", scopes: fillScopes },
  { name: "color/text/primary", alias: "primitive/gray/900", scopes: textScope },
  { name: "color/text/secondary", alias: "primitive/gray/500", scopes: textScope },
  { name: "color/text/inverse", alias: "primitive/gray/0", scopes: textScope },
  { name: "color/border/default", alias: "primitive/gray/300", scopes: strokeScope },
  { name: "color/border/focus", alias: "primitive/blue/500", scopes: strokeScope },
  { name: "color/status/success", alias: "primitive/green/500", scopes: fillScopes },
  { name: "color/status/warning", alias: "primitive/orange/500", scopes: fillScopes },
  { name: "color/status/danger", alias: "primitive/red/500", scopes: fillScopes },
  { name: "color/status/offline", alias: "primitive/slate/500", scopes: fillScopes },
  { name: "color/status/processing", alias: "primitive/blue/500", scopes: fillScopes }
];

export const DIMENSION_TOKENS: readonly DimensionToken[] = [
  ...[0, 4, 8, 12, 16, 20, 24, 32, 40, 48].map((value) => ({
    name: `space/${value}`,
    value,
    scopes: ["GAP"] as const
  })),
  { name: "radius/0", value: 0, scopes: ["CORNER_RADIUS"] },
  { name: "radius/4", value: 4, scopes: ["CORNER_RADIUS"] },
  { name: "radius/8", value: 8, scopes: ["CORNER_RADIUS"] },
  { name: "radius/12", value: 12, scopes: ["CORNER_RADIUS"] },
  { name: "radius/16", value: 16, scopes: ["CORNER_RADIUS"] },
  { name: "radius/24", value: 24, scopes: ["CORNER_RADIUS"] },
  { name: "radius/full", value: 999, scopes: ["CORNER_RADIUS"] },
  { name: "size/control/small", value: 32, scopes: ["WIDTH_HEIGHT"] },
  { name: "size/control/medium", value: 40, scopes: ["WIDTH_HEIGHT"] },
  { name: "size/control/large", value: 48, scopes: ["WIDTH_HEIGHT"] },
  { name: "stroke/default", value: 1, scopes: ["STROKE_FLOAT"] }
];

export function tokenCssSyntax(name: string): string {
  return `var(--${name.toLowerCase().replaceAll("/", "-").replaceAll(" ", "-")})`;
}

export function validateTokens(): string[] {
  const errors: string[] = [];
  const allNames = [...COLOR_TOKENS, ...DIMENSION_TOKENS].map((token) => token.name);
  const colorNames = new Set(COLOR_TOKENS.map((token) => token.name));

  for (const name of allNames) {
    if (!name.includes("/") || /\s/.test(name)) errors.push(`invalid token name: ${name}`);
    if (allNames.filter((candidate) => candidate === name).length > 1) errors.push(`duplicate token: ${name}`);
  }

  for (const token of COLOR_TOKENS) {
    if ((token.value === undefined) === (token.alias === undefined)) errors.push(`color must define value or alias: ${token.name}`);
    if (token.value && Object.values(token.value).some((channel) => channel < 0 || channel > 1)) errors.push(`invalid color channel: ${token.name}`);
    if (token.alias && !colorNames.has(token.alias)) errors.push(`missing alias target: ${token.name}`);
    if (token.scopes.length === 0) errors.push(`missing color scope: ${token.name}`);
  }

  for (const token of DIMENSION_TOKENS) {
    if (token.value < 0) errors.push(`negative dimension: ${token.name}`);
    if (token.scopes.length === 0) errors.push(`missing dimension scope: ${token.name}`);
  }

  return [...new Set(errors)];
}

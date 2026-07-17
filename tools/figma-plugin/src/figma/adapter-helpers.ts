import { OWNERSHIP, type MutationDecision } from "../domain/ownership";

const VALID_SCOPES = new Set<VariableScope>([
  "TEXT_CONTENT", "CORNER_RADIUS", "WIDTH_HEIGHT", "GAP", "ALL_FILLS", "FRAME_FILL", "SHAPE_FILL",
  "TEXT_FILL", "STROKE_COLOR", "STROKE_FLOAT", "EFFECT_FLOAT", "EFFECT_COLOR", "OPACITY",
  "FONT_FAMILY", "FONT_STYLE", "FONT_WEIGHT", "FONT_SIZE", "LINE_HEIGHT", "LETTER_SPACING"
]);

export function normalizeColor(value: { r: number; g: number; b: number }): RGB {
  const divisor = Math.max(value.r, value.g, value.b) > 1 ? 255 : 1;
  return { r: value.r / divisor, g: value.g / divisor, b: value.b / divisor };
}

export function toVariableScopes(scopes: readonly string[]): VariableScope[] {
  return scopes.filter((scope): scope is VariableScope => VALID_SCOPES.has(scope as VariableScope));
}

export function resourceDecision(existing: { owner: string; key: string; nameMatches: boolean } | null, key: string): MutationDecision {
  if (!existing) return "create";
  if (existing.owner === OWNERSHIP.owner && existing.key === key) return "update";
  if (existing.nameMatches) return "conflict";
  return "skip";
}

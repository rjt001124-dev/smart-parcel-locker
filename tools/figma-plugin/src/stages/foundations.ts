import { PAGE_KEYS } from "../domain/catalog";
import { createRunReport, recordResult, type RunReport } from "../domain/run-report";
import { COLOR_TOKENS, DIMENSION_TOKENS, tokenCssSyntax } from "../domain/tokens";
import type { FigmaPort, ResourceSpec } from "../figma/port";
import { FOUNDATION_SECTIONS } from "../domain/foundation-content";

const REQUIRED_FONTS = [
  { family: "Inter", style: "Regular" },
  { family: "Inter", style: "Medium" },
  { family: "Inter", style: "Semi Bold" },
  { family: "Inter", style: "Bold" }
] as const;

const TEXT_STYLES: readonly ResourceSpec[] = [
  { key: "text/display", name: "Display", family: "Inter", style: "Bold", size: 32, lineHeight: 40 },
  { key: "text/heading-1", name: "Heading/1", family: "Inter", style: "Semi Bold", size: 24, lineHeight: 32 },
  { key: "text/heading-2", name: "Heading/2", family: "Inter", style: "Semi Bold", size: 20, lineHeight: 28 },
  { key: "text/body", name: "Body", family: "Inter", style: "Regular", size: 14, lineHeight: 22 },
  { key: "text/label", name: "Label", family: "Inter", style: "Medium", size: 14, lineHeight: 20 },
  { key: "text/caption", name: "Caption", family: "Inter", style: "Regular", size: 12, lineHeight: 18 },
  { key: "text/number", name: "Number", family: "Inter", style: "Semi Bold", size: 28, lineHeight: 36 }
];

const EFFECT_STYLES: readonly ResourceSpec[] = [
  { key: "effect/card", name: "Shadow/Card", x: 0, y: 2, blur: 8, opacity: 0.08 },
  { key: "effect/floating", name: "Shadow/Floating", x: 0, y: 8, blur: 24, opacity: 0.12 },
  { key: "effect/modal", name: "Shadow/Modal", x: 0, y: 16, blur: 40, opacity: 0.16 },
  { key: "effect/focus", name: "Focus/Primary", x: 0, y: 0, blur: 0, spread: 3, opacity: 0.2 }
];

export async function runFoundations(port: FigmaPort): Promise<RunReport> {
  const available = await port.listAvailableFonts();
  const missing = REQUIRED_FONTS.filter((required) => !available.some((font) => font.family === required.family && font.style === required.style));
  if (missing.length > 0) {
    throw new Error(`Missing required fonts: ${missing.map((font) => `${font.family} ${font.style}`).join(", ")}`);
  }

  let report = createRunReport("foundations");
  const page = PAGE_KEYS[0];
  report = recordResult(report, await port.upsertPage({ key: page.key, name: page.name }));

  for (const collection of [
    { key: "collection/primitive-color", name: "Primitives / Color", modes: ["Light"] },
    { key: "collection/semantic-color", name: "Semantic / Color", modes: ["Light"] },
    { key: "collection/dimension", name: "Dimension", modes: ["Default"] }
  ]) report = recordResult(report, await port.upsertVariableCollection(collection));

  for (const token of COLOR_TOKENS) {
    const collectionKey = token.name.startsWith("primitive/") ? "collection/primitive-color" : "collection/semantic-color";
    report = recordResult(report, await port.upsertVariable({
      key: `variable/${token.name}`, name: token.name, collectionKey, type: "COLOR", value: token.value, alias: token.alias,
      scopes: token.scopes, codeSyntax: tokenCssSyntax(token.name)
    }));
  }
  for (const token of DIMENSION_TOKENS) report = recordResult(report, await port.upsertVariable({
    key: `variable/${token.name}`, name: token.name, collectionKey: "collection/dimension", type: "FLOAT", value: token.value,
    scopes: token.scopes, codeSyntax: tokenCssSyntax(token.name)
  }));
  for (const style of TEXT_STYLES) report = recordResult(report, await port.upsertTextStyle(style));
  for (const style of EFFECT_STYLES) report = recordResult(report, await port.upsertEffectStyle(style));

  report = recordResult(report, await port.upsertScreen({
    key: "screen/foundations/documentation", name: "Foundations / Documentation", pageKey: page.key,
    width: 1440, height: 1800, x: 80, y: 80,
    sections: FOUNDATION_SECTIONS
  }));

  if (report.status === "success") await port.setStageMarker("foundations");
  return report;
}

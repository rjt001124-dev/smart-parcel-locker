import { readdirSync, readFileSync, statSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

// src root = parent of this test/ dir
const SRC_DIR = join(__dirname, "..");

function walk(dir: string): string[] {
  const out: string[] = [];
  for (const entry of readdirSync(dir)) {
    const full = join(dir, entry);
    if (statSync(full).isDirectory()) out.push(...walk(full));
    else if (full.endsWith(".scss")) out.push(full);
  }
  return out;
}

const ALL_SCSS = walk(SRC_DIR);

// Pure-white / pure-black are acceptable (used for text on colored buttons, etc.)
const HEX_WHITELIST = new Set(["#fff", "#ffffff", "#000", "#000000"]);

function stripComments(code: string): string {
  return code
    .replace(/\/\*[\s\S]*?\*\//g, "")
    .replace(/\/\/.*$/gm, "");
}

describe("style conformance", () => {
  it("every scss file uses border-radius <= 8px and never $radius-card", () => {
    const violations: string[] = [];
    for (const file of ALL_SCSS) {
      const code = stripComments(readFileSync(file, "utf8"));
      const re = /border-radius\s*:\s*([^;]+);/g;
      let m: RegExpExecArray | null;
      while ((m = re.exec(code)) !== null) {
        const value = m[1]?.trim();
        if (!value) continue;
        if (value.includes("$radius-card")) {
          violations.push(`${file}: uses forbidden $radius-card (16px)`);
          continue;
        }
        const px = value.match(/(\d+(?:\.\d+)?)px/);
        if (px && parseFloat(px[1] ?? "0") > 8) {
          violations.push(`${file}: border-radius ${value} exceeds 8px`);
        }
      }
    }
    expect(violations, violations.join("\n")).toEqual([]);
  });

  it("no hardcoded hex colors (must use design tokens)", () => {
    const violations: string[] = [];
    for (const file of ALL_SCSS) {
      const code = stripComments(readFileSync(file, "utf8"));
      const re = /#([0-9a-fA-F]{3,8})\b/g;
      let m: RegExpExecArray | null;
      while ((m = re.exec(code)) !== null) {
        const hexPart = m[1];
        if (!hexPart) continue;
        const hex = `#${hexPart}`.toLowerCase();
        if (HEX_WHITELIST.has(hex)) continue;
        violations.push(`${file}: hardcoded color ${hex}`);
      }
    }
    expect(violations, violations.join("\n")).toEqual([]);
  });

  it("no nested card / panel containers", () => {
    const violations: string[] = [];
    for (const file of ALL_SCSS) {
      const code = stripComments(readFileSync(file, "utf8"));
      if (hasNestedCardRoot(code)) {
        violations.push(`${file}: nested card/panel container detected`);
      }
    }
    expect(violations, violations.join("\n")).toEqual([]);
  });
});

// A "card/panel root" is a simple selector whose class base is `card` or `panel`
// and is NOT a BEM element (no `__`). Nesting one card-root inside another is forbidden.
function isCardRootSelector(selector: string): boolean {
  const simples = selector.split(/[\s,>+~]+/).filter(Boolean);
  return simples.some((s) => {
    const m = s.match(/\.([a-zA-Z0-9_-]+)/);
    if (!m) return false;
    const cls = m[1];
    if (!cls) return false;
    if (cls.includes("__")) return false;
    return /(^|-)(card|panel)(-|$)/i.test(cls);
  });
}

function hasNestedCardRoot(code: string): boolean {
  const stack: string[] = []; // selectors of open blocks
  let buf = "";
  for (let i = 0; i < code.length; i++) {
    const c = code[i];
    if (c === "{") {
      const sel = buf.trim();
      const isCard = isCardRootSelector(sel);
      const ancestorCard = stack.some((s) => isCardRootSelector(s));
      if (isCard && ancestorCard) return true;
      stack.push(sel);
      buf = "";
    } else if (c === "}") {
      stack.pop();
      buf = "";
    } else if (c === ";") {
      buf = "";
    } else {
      buf += c;
    }
  }
  return false;
}

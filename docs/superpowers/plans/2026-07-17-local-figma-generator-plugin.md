# Local Figma Generator Plugin Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a repository-owned Figma development plugin that generates the smart parcel locker Foundations, component library, Mini App screens, and Admin Web screens without consuming Figma MCP quota.

**Architecture:** A TypeScript plugin uses a small `FigmaPort` boundary so catalogs, stage orchestration, ownership decisions, dependency checks, and reports are unit tested outside Figma. The real adapter writes variables, styles, Auto Layout nodes, components, and screens through the Figma Plugin API. Four idempotent stages are triggered from a minimal plugin UI and only mutate resources carrying the project ownership namespace.

**Tech Stack:** TypeScript 5, Figma Plugin API, `@figma/plugin-typings`, esbuild, Vitest, Node.js 20+

---

## File map

- `tools/figma-plugin/manifest.json` — Figma development-plugin manifest.
- `tools/figma-plugin/package.json` — build and test commands with locked dependencies.
- `tools/figma-plugin/tsconfig.json` — strict TypeScript configuration.
- `tools/figma-plugin/esbuild.mjs` — bundles plugin main thread and copies the UI.
- `tools/figma-plugin/src/code.ts` — plugin entry point and UI message router.
- `tools/figma-plugin/src/ui.html` — confirmation, four stage buttons, progress and report UI.
- `tools/figma-plugin/src/domain/catalog.ts` — stable pages, components, screens and state keys.
- `tools/figma-plugin/src/domain/tokens.ts` — primitive and semantic token definitions.
- `tools/figma-plugin/src/domain/ownership.ts` — ownership namespace and mutation decisions.
- `tools/figma-plugin/src/domain/run-report.ts` — structured stage outcome aggregation.
- `tools/figma-plugin/src/figma/port.ts` — testable interface used by stages.
- `tools/figma-plugin/src/figma/adapter.ts` — real Figma Plugin API implementation.
- `tools/figma-plugin/src/figma/node-builders.ts` — Auto Layout, text and shape helpers.
- `tools/figma-plugin/src/stages/foundations.ts` — variables, styles and documentation page.
- `tools/figma-plugin/src/stages/components.ts` — component page and component families.
- `tools/figma-plugin/src/stages/mini-app.ts` — 375 × 812 Mini App screens and states.
- `tools/figma-plugin/src/stages/admin-web.ts` — 1440 × 900 Admin screens and risk dialogs.
- `tools/figma-plugin/tests/*.test.ts` — pure-domain and fake-port stage tests.
- `tools/figma-plugin/README.md` — build, import, run, rerun and troubleshooting instructions.
- `docs/design/figma-state-ledger.json` — plugin delivery status and manual QA evidence.
- `docs/design/figma-qa.md` — repeatable Figma visual/structural checklist.

### Task 1: Scaffold a buildable Figma plugin

**Files:**
- Create: `tools/figma-plugin/package.json`
- Create: `tools/figma-plugin/tsconfig.json`
- Create: `tools/figma-plugin/esbuild.mjs`
- Create: `tools/figma-plugin/manifest.json`
- Create: `tools/figma-plugin/src/code.ts`
- Create: `tools/figma-plugin/src/ui.html`
- Create: `tools/figma-plugin/tests/scaffold.test.ts`
- Create: `tools/figma-plugin/.gitignore`

- [ ] **Step 1: Write the failing scaffold test**

```ts
import { readFileSync } from "node:fs";
import { describe, expect, it } from "vitest";

describe("plugin scaffold", () => {
  it("declares the generated main and UI files", () => {
    const manifest = JSON.parse(readFileSync("manifest.json", "utf8"));
    expect(manifest.name).toBe("Smart Parcel Locker Generator");
    expect(manifest.main).toBe("dist/code.js");
    expect(manifest.ui).toBe("dist/ui.html");
    expect(manifest.editorType).toEqual(["figma"]);
  });
});
```

- [ ] **Step 2: Run the test and verify RED**

Run from `tools/figma-plugin`:

```powershell
npm test -- --run tests/scaffold.test.ts
```

Expected: FAIL because `manifest.json` and the package do not exist.

- [ ] **Step 3: Add the minimal package and manifest**

`package.json`:

```json
{
  "name": "smart-parcel-locker-figma-plugin",
  "private": true,
  "version": "0.1.0",
  "type": "module",
  "scripts": {
    "build": "node esbuild.mjs",
    "test": "vitest",
    "typecheck": "tsc --noEmit"
  },
  "devDependencies": {
    "@figma/plugin-typings": "^1.117.0",
    "@types/node": "^24.0.0",
    "esbuild": "^0.25.0",
    "typescript": "^5.8.0",
    "vitest": "^3.2.0"
  }
}
```

`manifest.json`:

```json
{
  "name": "Smart Parcel Locker Generator",
  "id": "smart-parcel-locker-local-generator",
  "api": "1.0.0",
  "main": "dist/code.js",
  "ui": "dist/ui.html",
  "editorType": ["figma"],
  "documentAccess": "dynamic-page",
  "networkAccess": { "allowedDomains": ["none"] }
}
```

`tsconfig.json`:

```json
{
  "compilerOptions": {
    "target": "ES2022",
    "module": "ESNext",
    "moduleResolution": "Bundler",
    "strict": true,
    "types": ["@figma/plugin-typings", "node"],
    "skipLibCheck": true,
    "noEmit": true
  },
  "include": ["src/**/*.ts", "tests/**/*.ts"]
}
```

`esbuild.mjs`:

```js
import { build } from "esbuild";
import { copyFile, mkdir } from "node:fs/promises";

await mkdir("dist", { recursive: true });
await build({ entryPoints: ["src/code.ts"], bundle: true, outfile: "dist/code.js", target: "es2022" });
await copyFile("src/ui.html", "dist/ui.html");
```

Use a minimal `code.ts` that opens the UI and a minimal `ui.html` containing the plugin title. Ignore `node_modules/` and `dist/` in `.gitignore`.

- [ ] **Step 4: Install, test, type-check and build**

```powershell
npm install
npm test -- --run tests/scaffold.test.ts
npm run typecheck
npm run build
```

Expected: one passing test, clean type-check, and `dist/code.js` plus `dist/ui.html` created.

- [ ] **Step 5: Commit the scaffold**

```powershell
git add tools/figma-plugin
git commit -m "build: scaffold local Figma plugin"
```

### Task 2: Define stable catalogs and stage dependencies

**Files:**
- Create: `tools/figma-plugin/src/domain/catalog.ts`
- Create: `tools/figma-plugin/tests/catalog.test.ts`

- [ ] **Step 1: Write failing catalog tests**

```ts
import { describe, expect, it } from "vitest";
import { ADMIN_SCREENS, COMPONENT_KEYS, MINI_APP_SCREENS, PAGE_KEYS, REQUIRED_STATES, assertUniqueKeys, prerequisitesFor } from "../src/domain/catalog";

describe("design catalog", () => {
  it("has unique stable keys", () => {
    expect(() => assertUniqueKeys()).not.toThrow();
  });

  it("covers required pages and screen sizes", () => {
    expect(PAGE_KEYS.map((item) => item.name)).toEqual(["00 Foundations", "01 Components", "02 Mini App", "03 Admin Web"]);
    expect(MINI_APP_SCREENS.every((item) => item.width === 375 && item.height === 812)).toBe(true);
    expect(ADMIN_SCREENS.every((item) => item.width === 1440 && item.height === 900)).toBe(true);
  });

  it("covers required failure and processing states", () => {
    expect(REQUIRED_STATES).toEqual(expect.arrayContaining(["loading", "empty", "network-failure", "payment-processing", "payment-failure", "payment-success", "cell-contention", "device-offline", "door-open-failure", "door-not-closed", "overdue-payment", "permission-denied"]));
  });

  it("requires stages in order", () => {
    expect(prerequisitesFor("components")).toEqual(["foundations"]);
    expect(prerequisitesFor("mini-app")).toEqual(["foundations", "components"]);
    expect(prerequisitesFor("admin-web")).toEqual(["foundations", "components"]);
  });

  it("includes the agreed core component families", () => {
    expect(COMPONENT_KEYS).toEqual(expect.arrayContaining(["button", "input", "site-card", "locker-size-card", "order-card", "data-table", "admin-sidebar"]));
  });
});
```

- [ ] **Step 2: Run and verify RED**

```powershell
npm test -- --run tests/catalog.test.ts
```

Expected: FAIL because `catalog.ts` does not exist.

- [ ] **Step 3: Implement immutable catalog constants**

Define exact stable keys for the four pages, 13 Mini App screens, 9 Admin screens, agreed component families, and 12 required states. Implement `assertUniqueKeys()` by flattening every key and throwing on a duplicate. Implement `prerequisitesFor()` with a typed stage map.

- [ ] **Step 4: Run tests and commit**

```powershell
npm test -- --run tests/catalog.test.ts
git add tools/figma-plugin/src/domain/catalog.ts tools/figma-plugin/tests/catalog.test.ts
git commit -m "feat: define Figma generation catalog"
```

Expected: five passing catalog tests.

### Task 3: Define validated design tokens

**Files:**
- Create: `tools/figma-plugin/src/domain/tokens.ts`
- Create: `tools/figma-plugin/tests/tokens.test.ts`

- [ ] **Step 1: Write failing token tests**

```ts
import { describe, expect, it } from "vitest";
import { COLOR_TOKENS, DIMENSION_TOKENS, tokenCssSyntax, validateTokens } from "../src/domain/tokens";

describe("design tokens", () => {
  it("uses unique slash-delimited names", () => expect(validateTokens()).toEqual([]));
  it("contains project semantic states", () => {
    const names = COLOR_TOKENS.map((token) => token.name);
    expect(names).toEqual(expect.arrayContaining(["color/action/primary", "color/status/success", "color/status/warning", "color/status/danger", "color/status/offline", "color/status/processing"]));
  });
  it("uses a 4px spacing scale", () => {
    expect(DIMENSION_TOKENS.filter((token) => token.name.startsWith("space/")).every((token) => Number(token.value) % 4 === 0)).toBe(true);
  });
  it("derives stable web syntax", () => expect(tokenCssSyntax("color/bg/primary")).toBe("var(--color-bg-primary)"));
});
```

- [ ] **Step 2: Run and verify RED**

```powershell
npm test -- --run tests/tokens.test.ts
```

Expected: FAIL because the token module is absent.

- [ ] **Step 3: Implement token definitions and validation**

Use blue `#1677FF` as the primary action primitive, neutral gray surfaces, green success, orange warning, red danger, and blue-gray offline. Represent colors as `{ r, g, b }` values normalized to 0–1. Add `space/0` through `space/48`, `radius/0` through `radius/full`, common control heights and one-pixel stroke. `validateTokens()` must report duplicate names, invalid channels, invalid scopes, missing aliases, and non-positive nonzero dimensions.

- [ ] **Step 4: Run tests and commit**

```powershell
npm test -- --run tests/tokens.test.ts
git add tools/figma-plugin/src/domain/tokens.ts tools/figma-plugin/tests/tokens.test.ts
git commit -m "feat: define smart locker design tokens"
```

### Task 4: Implement ownership decisions and run reports

**Files:**
- Create: `tools/figma-plugin/src/domain/ownership.ts`
- Create: `tools/figma-plugin/src/domain/run-report.ts`
- Create: `tools/figma-plugin/tests/ownership.test.ts`
- Create: `tools/figma-plugin/tests/run-report.test.ts`

- [ ] **Step 1: Write failing ownership and report tests**

```ts
import { describe, expect, it } from "vitest";
import { decideMutation, OWNERSHIP } from "../src/domain/ownership";
import { createRunReport, recordResult } from "../src/domain/run-report";

describe("ownership", () => {
  it("updates a matching owned resource", () => expect(decideMutation({ nameMatches: true, owner: OWNERSHIP.owner, keyMatches: true })).toBe("update"));
  it("blocks an unowned name conflict", () => expect(decideMutation({ nameMatches: true, owner: "", keyMatches: false })).toBe("conflict"));
  it("creates when no resource exists", () => expect(decideMutation(null)).toBe("create"));
});

describe("run report", () => {
  it("aggregates partial failures without hiding successes", () => {
    let report = createRunReport("foundations");
    report = recordResult(report, { key: "page/foundations", outcome: "created" });
    report = recordResult(report, { key: "style/body", outcome: "error", message: "font unavailable" });
    expect(report.status).toBe("partial-failure");
    expect(report.counts).toMatchObject({ created: 1, error: 1 });
  });
});
```

- [ ] **Step 2: Run and verify RED**

```powershell
npm test -- --run tests/ownership.test.ts tests/run-report.test.ts
```

- [ ] **Step 3: Implement minimal pure domain logic**

Use namespace `smart-parcel-locker`, owner `local-figma-generator`, and schema version `1`. Mutation outcomes are `create | update | skip | conflict`; report outcomes add `error`. Derive stage status from recorded results.

- [ ] **Step 4: Run tests and commit**

```powershell
npm test -- --run tests/ownership.test.ts tests/run-report.test.ts
git add tools/figma-plugin/src/domain tools/figma-plugin/tests/ownership.test.ts tools/figma-plugin/tests/run-report.test.ts
git commit -m "feat: protect plugin-owned Figma resources"
```

### Task 5: Create the Figma port and fake adapter

**Files:**
- Create: `tools/figma-plugin/src/figma/port.ts`
- Create: `tools/figma-plugin/tests/fake-figma-port.ts`
- Create: `tools/figma-plugin/tests/stage-dependencies.test.ts`
- Create: `tools/figma-plugin/src/stages/dependencies.ts`

- [ ] **Step 1: Write a failing dependency test using the fake port**

```ts
import { describe, expect, it } from "vitest";
import { FakeFigmaPort } from "./fake-figma-port";
import { assertStageReady } from "../src/stages/dependencies";

describe("stage dependencies", () => {
  it("rejects components before foundations", async () => {
    const port = new FakeFigmaPort();
    await expect(assertStageReady(port, "components")).rejects.toThrow("Run Foundations first");
  });
});
```

- [ ] **Step 2: Run and verify RED**

```powershell
npm test -- --run tests/stage-dependencies.test.ts
```

- [ ] **Step 3: Define the port and fake**

The port must expose focused methods rather than raw Figma nodes: `getFileName`, `listAvailableFonts`, `hasStageMarker`, `setStageMarker`, `upsertPage`, `upsertVariableCollection`, `upsertVariable`, `upsertTextStyle`, `upsertEffectStyle`, `upsertComponentFamily`, `upsertScreen`, and `focusPage`. Implement the fake with Maps keyed by stable resource key.

- [ ] **Step 4: Implement and test dependency checks**

`assertStageReady()` uses `prerequisitesFor()` and `hasStageMarker()`, returning precise messages such as `Run Foundations first` and `Run Components first`.

- [ ] **Step 5: Run tests and commit**

```powershell
npm test -- --run tests/stage-dependencies.test.ts
git add tools/figma-plugin/src/figma/port.ts tools/figma-plugin/src/stages/dependencies.ts tools/figma-plugin/tests
git commit -m "feat: add testable Figma generation port"
```

### Task 6: Implement and test the Foundations stage

**Files:**
- Create: `tools/figma-plugin/src/stages/foundations.ts`
- Create: `tools/figma-plugin/tests/foundations.test.ts`
- Create: `tools/figma-plugin/src/figma/node-builders.ts`

- [ ] **Step 1: Write the failing fake-port stage test**

```ts
import { describe, expect, it } from "vitest";
import { runFoundations } from "../src/stages/foundations";
import { FakeFigmaPort } from "./fake-figma-port";

it("creates foundations once and updates on rerun", async () => {
  const port = new FakeFigmaPort();
  expect((await runFoundations(port)).counts.created).toBeGreaterThan(0);
  const sizeAfterFirstRun = port.resourceCount();
  expect((await runFoundations(port)).counts.updated).toBeGreaterThan(0);
  expect(port.resourceCount()).toBe(sizeAfterFirstRun);
  expect(await port.hasStageMarker("foundations")).toBe(true);
});
```

- [ ] **Step 2: Run and verify RED**

```powershell
npm test -- --run tests/foundations.test.ts
```

- [ ] **Step 3: Implement stage orchestration**

Validate at least Inter Regular, Medium, Semi Bold and Bold styles before writes. Upsert the page, collections, tokens, text styles, effect styles and one documentation screen. Record every port result and write the stage marker only when no error or conflict exists.

- [ ] **Step 4: Add pure node-builder specifications**

Define `AutoLayoutSpec`, `TextSpec`, `ShapeSpec`, `ComponentFamilySpec`, and `ScreenSpec` types used by the real and fake adapters. Every related-child container must use Auto Layout specifications.

- [ ] **Step 5: Run focused and full tests, then commit**

```powershell
npm test -- --run tests/foundations.test.ts
npm test -- --run
git add tools/figma-plugin/src/stages/foundations.ts tools/figma-plugin/src/figma/node-builders.ts tools/figma-plugin/tests/foundations.test.ts
git commit -m "feat: generate Figma foundations"
```

### Task 7: Implement and test the Components stage

**Files:**
- Create: `tools/figma-plugin/src/stages/components.ts`
- Create: `tools/figma-plugin/tests/components.test.ts`

- [ ] **Step 1: Write failing stage tests**

```ts
it("requires foundations and creates every catalog component once", async () => {
  const port = new FakeFigmaPort();
  await expect(runComponents(port)).rejects.toThrow("Run Foundations first");
  await runFoundations(port);
  const report = await runComponents(port);
  expect(report.status).toBe("success");
  expect(port.componentKeys()).toEqual(expect.arrayContaining(COMPONENT_KEYS));
  const count = port.resourceCount();
  await runComponents(port);
  expect(port.resourceCount()).toBe(count);
});
```

- [ ] **Step 2: Run RED, implement, and verify GREEN**

```powershell
npm test -- --run tests/components.test.ts
```

Implement Button, Icon Button, Input, Search, Select, status feedback, cards, dialogs, navigation and admin data component specifications. Component variants must bind semantic token names and use stable keys.

```powershell
npm test -- --run tests/components.test.ts
npm test -- --run
```

- [ ] **Step 3: Commit**

```powershell
git add tools/figma-plugin/src/stages/components.ts tools/figma-plugin/tests/components.test.ts
git commit -m "feat: generate Figma component library"
```

### Task 8: Implement and test Mini App screens

**Files:**
- Create: `tools/figma-plugin/src/stages/mini-app.ts`
- Create: `tools/figma-plugin/tests/mini-app.test.ts`

- [ ] **Step 1: Write failing screen coverage tests**

Verify dependency rejection, all 13 catalog screens, `375 × 812` dimensions, component-instance references, and the complete required-state matrix.

- [ ] **Step 2: Run RED**

```powershell
npm test -- --run tests/mini-app.test.ts
```

- [ ] **Step 3: Implement screen specifications**

Each screen uses a vertical Auto Layout root, status/top bar, scrollable content specification, and bottom action/navigation region where applicable. Use Chinese user-facing copy and action-oriented error recovery. Build the normal flow in review order and a separate `Mini App / State Matrix` screen.

- [ ] **Step 4: Run GREEN and commit**

```powershell
npm test -- --run tests/mini-app.test.ts
npm test -- --run
git add tools/figma-plugin/src/stages/mini-app.ts tools/figma-plugin/tests/mini-app.test.ts
git commit -m "feat: generate mini app Figma screens"
```

### Task 9: Implement and test Admin Web screens

**Files:**
- Create: `tools/figma-plugin/src/stages/admin-web.ts`
- Create: `tools/figma-plugin/tests/admin-web.test.ts`

- [ ] **Step 1: Write failing screen and safety tests**

Verify all 9 screens, `1440 × 900` dimensions, dark sidebar, light content surface, table/filter components, permission-denied state, and confirmation specifications for remote open, restart, isolate and rollback.

- [ ] **Step 2: Run RED, implement and verify GREEN**

```powershell
npm test -- --run tests/admin-web.test.ts
```

Implement the screen specifications and high-risk confirmation dialog content, including affected device, reason input, risk warning and audit notice.

```powershell
npm test -- --run tests/admin-web.test.ts
npm test -- --run
```

- [ ] **Step 3: Commit**

```powershell
git add tools/figma-plugin/src/stages/admin-web.ts tools/figma-plugin/tests/admin-web.test.ts
git commit -m "feat: generate admin web Figma screens"
```

### Task 10: Implement the real Figma adapter

**Files:**
- Create: `tools/figma-plugin/src/figma/adapter.ts`
- Create: `tools/figma-plugin/tests/adapter-decisions.test.ts`

- [ ] **Step 1: Write failing adapter-decision tests**

Extract and test pure functions for locating owned nodes, resolving name conflicts, normalizing colors, mapping variable scopes, and converting node specs to Figma property payloads. Do not mock the entire Figma global.

- [ ] **Step 2: Run RED**

```powershell
npm test -- --run tests/adapter-decisions.test.ts
```

- [ ] **Step 3: Implement the adapter**

The adapter must:

- use `getSharedPluginData`/`setSharedPluginData` with the approved namespace;
- use `await figma.setCurrentPageAsync(page)` for page changes;
- load verified fonts before text mutations;
- create variables with explicit scopes and WEB code syntax;
- bind semantic variables to fills, strokes, gaps, radii and text where supported;
- use Auto Layout for related children;
- create components and combine component variants;
- update or replace only owned subtrees;
- never delete unknown nodes;
- return stable keys and node IDs in each result.

- [ ] **Step 4: Type-check, test and build**

```powershell
npm run typecheck
npm test -- --run
npm run build
```

Expected: clean type-check, all tests pass, build exits 0.

- [ ] **Step 5: Commit**

```powershell
git add tools/figma-plugin/src/figma tools/figma-plugin/tests/adapter-decisions.test.ts
git commit -m "feat: write generated designs through Figma API"
```

### Task 11: Implement the plugin UI and message router

**Files:**
- Modify: `tools/figma-plugin/src/code.ts`
- Modify: `tools/figma-plugin/src/ui.html`
- Create: `tools/figma-plugin/tests/messages.test.ts`
- Create: `tools/figma-plugin/src/domain/messages.ts`

- [ ] **Step 1: Write failing message validation tests**

Test accepted messages (`run-stage`, `close-plugin`), rejected unknown stages, required file confirmation, and serializable run-report responses.

- [ ] **Step 2: Run RED**

```powershell
npm test -- --run tests/messages.test.ts
```

- [ ] **Step 3: Implement the UI**

The UI shows the current file name, confirmation checkbox, ordered stage buttons, status badges, counts, copyable error details and a close button. Disable all run buttons until confirmation, disable the active button during a run, and preserve no sensitive information.

- [ ] **Step 4: Implement the router**

Open a `420 × 620` UI, validate messages, create the real adapter, dispatch exactly one stage, and post progress/result messages. Catch errors at the stage boundary and return a failed run report rather than an unhandled rejection.

- [ ] **Step 5: Run tests, type-check, build and commit**

```powershell
npm test -- --run tests/messages.test.ts
npm test -- --run
npm run typecheck
npm run build
git add tools/figma-plugin/src tools/figma-plugin/tests/messages.test.ts
git commit -m "feat: add staged Figma generator UI"
```

### Task 12: Document installation and perform manual Figma QA

**Files:**
- Create: `tools/figma-plugin/README.md`
- Create: `docs/design/figma-qa.md`
- Modify: `docs/design/figma-state-ledger.json`
- Modify: `README.md`

- [ ] **Step 1: Write exact installation guidance**

Document prerequisites, `npm install`, `npm run build`, Figma Desktop import path, manifest location, target file link, four-stage order, rerun behavior, conflict behavior, font failures and how to copy reports.

- [ ] **Step 2: Run the complete automated suite**

```powershell
Set-Location tools/figma-plugin
npm test -- --run
npm run typecheck
npm run build
Set-Location ../..
python scripts/test_agents_rules.py
python scripts/test_validate_skills.py
python scripts/validate_skills.py
git diff --check
```

Expected: all plugin tests, TypeScript checks, plugin build and repository validators exit 0.

- [ ] **Step 3: Import and run in Figma Desktop**

Open `https://www.figma.com/design/ui9lT54QlghpCFiYxiB6WT`, import `tools/figma-plugin/manifest.json`, confirm the target file, and run Foundations → Components → Mini App → Admin Web.

Expected: every stage returns success with created/updated counts and no conflicts.

- [ ] **Step 4: Verify idempotency**

Run all four stages a second time.

Expected: page and stable-key counts do not increase; reports show updates rather than duplicate creation.

- [ ] **Step 5: Complete visual and structural QA**

Record in `docs/design/figma-qa.md`:

- four page names and order;
- variable collections, styles and component families;
- all Mini App and Admin screen sizes;
- required state matrix coverage;
- no overlaps, clipped text, wrong fonts or unknown-node deletion;
- screenshot references supplied by the user when MCP screenshot access remains unavailable.

Update `figma-state-ledger.json` from `blocked_mcp_rate_limit` to `local_plugin_generated` only after the Figma run succeeds. Set `qa.approved` to `true` only after user visual approval.

- [ ] **Step 6: Commit documentation and QA evidence**

```powershell
git add tools/figma-plugin/README.md docs/design README.md
git commit -m "docs: explain local Figma generation workflow"
```

### Task 13: Final verification and branch publication

**Files:**
- Verify only; modify a file only to fix a demonstrated failure.

- [ ] **Step 1: Run fresh full verification**

```powershell
Set-Location tools/figma-plugin
npm test -- --run
npm run typecheck
npm run build
Set-Location ../..
go mod verify
go vet ./...
go test ./...
python scripts/test_agents_rules.py
python scripts/test_validate_skills.py
python scripts/validate_skills.py
git diff --check
git status --short --branch
```

Expected: zero test failures, clean type-check/build, Go and Python verification pass, no whitespace errors, and a clean working tree.

- [ ] **Step 2: Push the implementation branch**

```powershell
git push origin codex/milestone-2-figma-baseline
```

- [ ] **Step 3: Report the manual action boundary**

If the plugin has not yet been run inside Figma Desktop, report the code as verified but the Figma design as awaiting the user's import/run. Do not claim the design baseline is complete until the four stages and visual QA have succeeded.

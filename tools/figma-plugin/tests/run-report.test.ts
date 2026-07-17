import { describe, expect, it } from "vitest";
import { createRunReport, recordResult } from "../src/domain/run-report";

describe("run report", () => {
  it("aggregates partial failures without hiding successes", () => {
    let report = createRunReport("foundations");
    report = recordResult(report, { key: "page/foundations", outcome: "created" });
    report = recordResult(report, { key: "style/body", outcome: "error", message: "font unavailable" });

    expect(report.status).toBe("partial-failure");
    expect(report.counts).toMatchObject({ created: 1, error: 1 });
  });
});

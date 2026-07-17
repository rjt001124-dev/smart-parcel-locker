import type { Stage } from "./catalog";

export type ResourceOutcome = "created" | "updated" | "skipped" | "conflict" | "error";
export type RunStatus = "success" | "partial-failure" | "failed";

export interface ResourceResult {
  key: string;
  outcome: ResourceOutcome;
  nodeId?: string;
  message?: string;
}

export interface RunReport {
  stage: Stage;
  status: RunStatus;
  counts: Record<ResourceOutcome, number>;
  results: readonly ResourceResult[];
}

export function createRunReport(stage: Stage): RunReport {
  return {
    stage,
    status: "success",
    counts: { created: 0, updated: 0, skipped: 0, conflict: 0, error: 0 },
    results: []
  };
}

export function recordResult(report: RunReport, result: ResourceResult): RunReport {
  const counts = { ...report.counts, [result.outcome]: report.counts[result.outcome] + 1 };
  const completed = counts.created + counts.updated + counts.skipped;
  const failures = counts.conflict + counts.error;
  const status: RunStatus = failures === 0 ? "success" : completed === 0 ? "failed" : "partial-failure";
  return { ...report, status, counts, results: [...report.results, result] };
}

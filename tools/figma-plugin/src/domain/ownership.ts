export const OWNERSHIP = {
  namespace: "smart-parcel-locker",
  owner: "local-figma-generator",
  schemaVersion: "1"
} as const;

export type MutationDecision = "create" | "update" | "skip" | "conflict";

export interface ExistingResource {
  nameMatches: boolean;
  owner: string;
  keyMatches: boolean;
}

export function decideMutation(existing: ExistingResource | null): MutationDecision {
  if (!existing) return "create";
  if (existing.owner === OWNERSHIP.owner && existing.keyMatches) return "update";
  if (existing.nameMatches) return "conflict";
  return "skip";
}

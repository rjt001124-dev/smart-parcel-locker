import { prerequisitesFor, type Stage } from "../domain/catalog";
import type { FigmaPort } from "../figma/port";

const STAGE_LABELS: Readonly<Record<Stage, string>> = {
  foundations: "Foundations",
  components: "Components",
  "mini-app": "Mini App",
  "admin-web": "Admin Web"
};

export async function assertStageReady(port: Pick<FigmaPort, "hasStageMarker">, stage: Stage): Promise<void> {
  for (const prerequisite of prerequisitesFor(stage)) {
    if (!(await port.hasStageMarker(prerequisite))) {
      throw new Error(`Run ${STAGE_LABELS[prerequisite]} first`);
    }
  }
}

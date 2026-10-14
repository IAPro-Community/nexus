import type { MissionTeamSelection, PackageRun, WorkPlan } from "../../types";
import { asArray } from "../../lib/safeArray";
import type { FlowRunState } from "./flowRunModel";

/** Stateless view projection helpers shared by FlowRunSurface and its tests. */
export const terminalMissionStates = new Set([
  "COMPLETED_VERIFIED",
  "FAILED",
  "FAILED_NO_PROGRESS",
  "FAILED_BUDGET_EXCEEDED",
  "FAILED_VERIFICATION",
  "CANCELED_BY_USER",
]);

export const toneFor = (
  state: FlowRunState,
): "default" | "brand" | "success" | "warning" | "danger" => {
  if (state === "COMPLETED") return "success";
  if (state === "FAILED" || state === "CANCELED") return "danger";
  if (state === "VERIFYING" || state === "BLOCKED") return "warning";
  if (state === "RUNNING" || state === "READY") return "brand";
  return "default";
};

export const providerIssuePattern =
  /provider|quota|budget|credential|auth|rate.?limit|resource|unauthorized|forbidden|insufficient|payment/i;

export const readMissionTeamSelection = (
  plan: WorkPlan | null,
): MissionTeamSelection | null => {
  const raw = plan?.structured_facts?.["nexus.mission_team.selection"];
  if (!raw) return null;
  try {
    const parsed = JSON.parse(raw) as MissionTeamSelection;
    return parsed.direct ? null : parsed;
  } catch {
    return null;
  }
};

export const readSelectedResource = (pkg: PackageRun | undefined) => {
  if (!pkg) return "";
  try {
    const routing = JSON.parse(pkg.routing_decision || "{}") as {
      selected_provider?: string;
      selected_profile?: string;
      selected_model?: string;
    };
    const provider = routing.selected_provider || pkg.provider;
    const profile = routing.selected_profile || pkg.profile;
    const model = routing.selected_model;
    return [provider, profile, model].filter(Boolean).join(" / ");
  } catch {
    return [pkg.provider, pkg.profile].filter(Boolean).join(" / ");
  }
};

// readExecutionSummary projects the unified execution summary from existing
// models only: skills pinned on the package, reasoning/model from the routing
// decision, isolation from the package workspace, and the routing mode with
// its typed reason. Nothing is invented when the data is absent.
export const readExecutionSummary = (pkg: PackageRun | undefined) => {
  if (!pkg) return null;
  let routing: {
    selected_reasoning?: string;
    selected_model?: string;
    fallback?: boolean;
    reason?: string;
  } = {};
  try {
    routing = JSON.parse(pkg.routing_decision || "{}") as typeof routing;
  } catch {
    routing = {};
  }
  const skills = [
    ...asArray<string>(pkg.skill_ids),
    ...asArray<string>(pkg.maestro_skills),
  ];
  return {
    skillsText: skills.join(", "),
    reasoning: routing.selected_reasoning || routing.selected_model || "",
    isolation: pkg.workspace || "",
    fallback: routing.fallback === true,
    reason: routing.reason || "",
  };
};

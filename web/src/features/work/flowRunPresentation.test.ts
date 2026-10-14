import { describe, expect, it } from "vitest";
import type { PackageRun, WorkPlan } from "../../types";
import {
  providerIssuePattern,
  readExecutionSummary,
  readMissionTeamSelection,
  readSelectedResource,
  terminalMissionStates,
  toneFor,
} from "./flowRunPresentation";

describe("flow run presentation projections", () => {
  it("keeps terminal mission outcomes and status tones stable", () => {
    expect(terminalMissionStates.has("COMPLETED_VERIFIED")).toBe(true);
    expect(terminalMissionStates.has("FAILED_VERIFICATION")).toBe(true);
    expect(terminalMissionStates.has("RUNNING")).toBe(false);
    expect(toneFor("COMPLETED")).toBe("success");
    expect(toneFor("FAILED")).toBe("danger");
    expect(toneFor("VERIFYING")).toBe("warning");
    expect(toneFor("RUNNING")).toBe("brand");
  });

  it("does not invent team selection for invalid or direct-only plans", () => {
    expect(readMissionTeamSelection(null)).toBeNull();
    const plan = (facts: string) =>
      ({
        structured_facts: { "nexus.mission_team.selection": facts },
      }) as unknown as WorkPlan;
    expect(readMissionTeamSelection(plan("not json"))).toBeNull();
    expect(readMissionTeamSelection(plan('{"direct":true}'))).toBeNull();
    expect(readMissionTeamSelection(plan('{"direct":false}'))).toEqual({
      direct: false,
    });
  });

  it("keeps provider display stable for valid and malformed routing decisions", () => {
    const pkg = (routing_decision: string) =>
      ({
        provider: "codex",
        profile: "primary",
        routing_decision,
      }) as unknown as PackageRun;
    expect(readSelectedResource(pkg("{}"))).toBe("codex / primary");
    expect(readSelectedResource(pkg("invalid"))).toBe("codex / primary");
    expect(
      readSelectedResource(
        pkg(
          '{"selected_provider":"opencode","selected_profile":"lead","selected_model":"model-x"}',
        ),
      ),
    ).toBe("opencode / lead / model-x");
    expect(readSelectedResource(undefined)).toBe("");
  });

  it("only reports actual selected execution data", () => {
    const pkg = {
      routing_decision:
        '{"selected_reasoning":"high","fallback":true,"reason":"quota"}',
      skill_ids: ["planning"],
      maestro_skills: ["review"],
      workspace: "/tmp/mission",
    } as unknown as PackageRun;
    expect(readExecutionSummary(pkg)).toEqual({
      skillsText: "planning, review",
      reasoning: "high",
      isolation: "/tmp/mission",
      fallback: true,
      reason: "quota",
    });
    expect(readExecutionSummary(undefined)).toBeNull();
  });

  it("recognizes provider resource errors but not arbitrary work status", () => {
    expect(providerIssuePattern.test("quota exceeded")).toBe(true);
    expect(providerIssuePattern.test("provider unauthorized")).toBe(true);
    expect(providerIssuePattern.test("review complete")).toBe(false);
  });
});

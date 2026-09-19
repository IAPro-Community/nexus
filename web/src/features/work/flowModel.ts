import type { WorkPackage, WorkPlan } from '../../types';

export const FLOW_POLICY_FACT = 'nexus.flow_policy';
export type FlowPolicy = 'GUIDED' | 'AUTONOMOUS';
export type FlowAssignmentStrategy = 'EXISTING' | 'CREATE' | 'AUTO';

export interface FlowPhaseMeta {
  id: string;
  title: string;
  description?: string;
  order: number;
}

export interface FlowStepModel {
  id: string;
  phaseId: string;
  order: number;
  title: string;
  goal: string;
  priority: WorkPackage['priority'];
  status: WorkPackage['status'];
  dependencies: string[];
  parallelGroup?: string;
  role: string;
  taskRequirements?: string;
  assignmentStrategy: FlowAssignmentStrategy;
  agentId?: string;
  resourcePolicy?: string;
  provider?: string;
  profile?: string;
  skillIds: string[];
  maestroGates: string[];
  maestroSkills: string[];
  relevantPaths: string[];
  acceptanceCriteria: string[];
  verificationRequirements: string[];
  sharedArtifacts: string[];
  compiledPrompt?: string;
}

export interface FlowDraftModel {
  id: string;
  projectId: string;
  missionId?: string;
  title: string;
  description: string;
  status: WorkPlan['status'];
  revision: number;
  policy: FlowPolicy;
  policyStored: boolean;
  phases: FlowPhaseMeta[];
  steps: FlowStepModel[];
  structuredFacts?: Record<string, string>;
}

const clone = <T>(items: T[] | undefined): T[] => (items ? [...items] : []);

export function packageAssignment(pkg: WorkPackage): FlowAssignmentStrategy {
  if (
    pkg.assignment_strategy === 'EXISTING' ||
    pkg.assignment_strategy === 'CREATE' ||
    pkg.assignment_strategy === 'AUTO'
  )
    return pkg.assignment_strategy;
  return pkg.agent_allocation ? 'EXISTING' : 'AUTO';
}

export function flowFromWorkPlan(plan: WorkPlan): FlowDraftModel {
  const rawPolicy = plan.structured_facts?.[FLOW_POLICY_FACT];
  const steps: FlowStepModel[] = [];
  const phases = Array.isArray(plan?.phases) ? plan.phases : [];
  phases.forEach((phase) => {
    const packages = Array.isArray(phase?.packages) ? phase.packages : [];
    packages.forEach((pkg, order) => {
      steps.push({
        id: pkg.id,
        phaseId: phase.id,
        order,
        title: pkg.title,
        goal: pkg.goal,
        priority: pkg.priority,
        status: pkg.status,
        dependencies: clone(pkg.dependencies),
        parallelGroup: pkg.parallel_group,
        role: pkg.role,
        taskRequirements: pkg.task_requirements,
        assignmentStrategy: packageAssignment(pkg),
        agentId: pkg.agent_allocation,
        resourcePolicy: pkg.resource_policy,
        provider: pkg.provider,
        profile: pkg.profile,
        skillIds: clone(pkg.skill_ids || pkg.maestro_skills),
        maestroGates: clone(pkg.maestro_gates),
        maestroSkills: clone(pkg.maestro_skills),
        relevantPaths: clone(pkg.relevant_paths),
        acceptanceCriteria: clone(pkg.acceptance_criteria),
        verificationRequirements: clone(pkg.verification_requirements),
        sharedArtifacts: clone(pkg.shared_artifacts),
        compiledPrompt: pkg.compiled_prompt,
      });
    });
  });
  return {
    id: plan.id,
    projectId: plan.project_id,
    missionId: plan.mission_id,
    title: plan.title,
    description: plan.description,
    status: plan.status,
    revision: plan.current_revision,
    policy: rawPolicy === 'AUTONOMOUS' ? 'AUTONOMOUS' : 'GUIDED',
    policyStored: rawPolicy === 'GUIDED' || rawPolicy === 'AUTONOMOUS',
    phases: phases.map((phase) => ({
      id: phase.id,
      title: phase.title,
      description: phase.description,
      order: phase.order,
    })),
    steps: normalizeFlowSteps(steps),
    structuredFacts: plan.structured_facts ? { ...plan.structured_facts } : undefined,
  };
}

/** Stable graph identity so React Flow does not rebuild nodes on object churn. */
export function flowGraphSignature(flow: FlowDraftModel): string {
  return (flow.steps || [])
    .map(
      (step) =>
        `${step.id}:${step.title}:${step.status}:${step.phaseId}:${(step.dependencies || []).join(',')}`,
    )
    .join('|');
}

export function workPlanFromFlow(flow: FlowDraftModel, baseline: WorkPlan): WorkPlan {
  const facts = flow.structuredFacts ? { ...flow.structuredFacts } : {};
  if (flow.policyStored || flow.policy !== 'GUIDED') facts[FLOW_POLICY_FACT] = flow.policy;
  else delete facts[FLOW_POLICY_FACT];
  const flowPhases = Array.isArray(flow?.phases) ? flow.phases : [];
  const flowSteps = Array.isArray(flow?.steps) ? flow.steps : [];
  const phaseMap = new Map(
    flowPhases.map((phase) => [phase.id, { ...phase, packages: [] as WorkPackage[] }]),
  );
  [...flowSteps]
    .sort(
      (a, b) =>
        phaseOrder(flow, a.phaseId) - phaseOrder(flow, b.phaseId) ||
        a.order - b.order ||
        a.id.localeCompare(b.id),
    )
    .forEach((step) => {
      let phase = phaseMap.get(step.phaseId);
      if (!phase) {
        phase = { id: step.phaseId, title: 'Flow', order: phaseMap.size + 1, packages: [] };
        phaseMap.set(step.phaseId, phase);
      }
      phase.packages.push({
        id: step.id,
        title: step.title,
        goal: step.goal,
        priority: step.priority,
        status: step.status,
        dependencies: clone(step.dependencies),
        parallel_group: step.parallelGroup,
        role: step.role,
        task_requirements: step.taskRequirements,
        agent_allocation: step.agentId,
        assignment_strategy: step.assignmentStrategy,
        resource_policy: step.resourcePolicy,
        provider: step.provider,
        profile: step.profile,
        skill_ids: clone(step.skillIds),
        maestro_gates: clone(step.maestroGates),
        maestro_skills: clone(step.maestroSkills),
        relevant_paths: clone(step.relevantPaths),
        acceptance_criteria: clone(step.acceptanceCriteria),
        verification_requirements: clone(step.verificationRequirements),
        shared_artifacts: clone(step.sharedArtifacts),
        compiled_prompt: step.compiledPrompt,
      });
    });
  return {
    ...baseline,
    id: flow.id,
    project_id: flow.projectId,
    mission_id: flow.missionId,
    title: flow.title,
    description: flow.description,
    status: flow.status,
    current_revision: flow.revision,
    phases: flowPhases.map((meta) => {
      const phase = phaseMap.get(meta.id)!;
      return {
        id: meta.id,
        title: meta.title,
        description: meta.description,
        order: meta.order,
        packages: phase.packages,
      };
    }),
    structured_facts: Object.keys(facts).length ? facts : undefined,
  };
}

function isSchemaPlaceholder(value: string | undefined): boolean {
  const trimmed = (value || '').trim();
  if (!trimmed) return true;
  const lower = trimmed.toLowerCase();
  if (
    lower === '...' ||
    lower === '…' ||
    lower === 'title' ||
    lower === 'goal' ||
    lower === 'package title' ||
    lower === 'measurable criterion' ||
    lower === 'specific measurable objective'
  )
    return true;
  if (trimmed.startsWith('<') && trimmed.endsWith('>')) return true;
  return trimmed.replace(/[.…]/g, '') === '';
}

function uniqueIds(values: string[]): string[] {
  const seen = new Set<string>();
  const out: string[] = [];
  for (const value of values) {
    if (!value || seen.has(value)) continue;
    seen.add(value);
    out.push(value);
  }
  return out;
}

function normalizeFlowSteps(steps: FlowStepModel[]): FlowStepModel[] {
  const byId = new Set(steps.map((step) => step.id));
  const byTitle = new Map<string, string>();
  for (const step of steps) {
    const key = (step.title || '').trim().toLowerCase();
    if (key && !isSchemaPlaceholder(key) && !byTitle.has(key)) byTitle.set(key, step.id);
  }
  return steps.map((step, index) => {
    const title = isSchemaPlaceholder(step.title)
      ? !isSchemaPlaceholder(step.goal)
        ? step.goal.trim()
        : `Work step ${index + 1}`
      : step.title;
    const goal = isSchemaPlaceholder(step.goal) ? title : step.goal;
    const dependencies = uniqueIds(
      (step.dependencies || []).flatMap((dep) => {
        const token = (dep || '').trim();
        if (!token || isSchemaPlaceholder(token) || token === step.id) return [];
        if (byId.has(token)) return [token];
        const mapped = byTitle.get(token.toLowerCase());
        return mapped && mapped !== step.id ? [mapped] : [];
      }),
    );
    const acceptanceCriteria = (step.acceptanceCriteria || []).filter(
      (item) => !isSchemaPlaceholder(item),
    );
    return { ...step, title, goal, dependencies, acceptanceCriteria };
  });
}

function phaseOrder(flow: FlowDraftModel, phaseId: string): number {
  return (
    (flow.phases || []).find((phase) => phase.id === phaseId)?.order ?? Number.MAX_SAFE_INTEGER
  );
}
function compareStep(flow: FlowDraftModel, a: FlowStepModel, b: FlowStepModel): number {
  return (
    phaseOrder(flow, a.phaseId) - phaseOrder(flow, b.phaseId) ||
    a.order - b.order ||
    a.id.localeCompare(b.id)
  );
}

function dependencyMaps(flow: FlowDraftModel) {
  const steps = Array.isArray(flow?.steps) ? flow.steps : [];
  const byId = new Map<string, FlowStepModel>();
  for (const step of steps) {
    if (!step.id) throw new Error('Flow Step id is required.');
    if (byId.has(step.id)) throw new Error(`duplicate Flow Step ${step.id}.`);
    byId.set(step.id, step);
  }
  const indegree = new Map(steps.map((step) => [step.id, 0]));
  const dependents = new Map<string, string[]>();
  for (const step of steps)
    for (const dep of step.dependencies || []) {
      if (dep === step.id) throw new Error(`${step.title}: self dependency.`);
      if (!byId.has(dep)) throw new Error(`${step.title}: unknown dependency ${dep}.`);
      indegree.set(step.id, (indegree.get(step.id) || 0) + 1);
      dependents.set(dep, [...(dependents.get(dep) || []), step.id]);
    }
  return { byId, indegree, dependents };
}

export function topologicalOrder(flow: FlowDraftModel): string[] {
  const steps = Array.isArray(flow?.steps) ? flow.steps : [];
  const { byId, indegree, dependents } = dependencyMaps(flow);
  const ready = steps
    .filter((step) => (indegree.get(step.id) || 0) === 0)
    .sort((a, b) => compareStep(flow, a, b));
  const out: string[] = [];
  while (ready.length) {
    const step = ready.shift()!;
    out.push(step.id);
    const children = (dependents.get(step.id) || [])
      .map((id) => byId.get(id)!)
      .sort((a, b) => compareStep(flow, a, b));
    for (const child of children) {
      indegree.set(child.id, (indegree.get(child.id) || 0) - 1);
      if (indegree.get(child.id) === 0) {
        ready.push(child);
        ready.sort((a, b) => compareStep(flow, a, b));
      }
    }
  }
  if (out.length !== steps.length) throw new Error('Flow contains a dependency cycle.');
  return out;
}

export function executionWaves(flow: FlowDraftModel): string[][] {
  const steps = Array.isArray(flow?.steps) ? flow.steps : [];
  const { byId, indegree, dependents } = dependencyMaps(flow);
  const remaining = new Set(steps.map((step) => step.id));
  const waves: string[][] = [];
  while (remaining.size) {
    const ready = [...remaining]
      .filter((id) => (indegree.get(id) || 0) === 0)
      .map((id) => byId.get(id)!)
      .sort((a, b) => compareStep(flow, a, b));
    if (!ready.length) throw new Error('Flow contains a dependency cycle.');
    const wave = ready.map((step) => step.id);
    waves.push(wave);
    wave.forEach((id) => remaining.delete(id));
    wave.forEach((id) =>
      (dependents.get(id) || []).forEach((child) =>
        indegree.set(child, (indegree.get(child) || 0) - 1),
      ),
    );
  }
  return waves;
}

export function validateFlow(flow: FlowDraftModel): string[] {
  const errors: string[] = [];
  const phases = Array.isArray(flow?.phases) ? flow.phases : [];
  const steps = Array.isArray(flow?.steps) ? flow.steps : [];
  const phaseIds = new Set(phases.map((phase) => phase.id));
  try {
    topologicalOrder(flow);
  } catch (error) {
    errors.push(error instanceof Error ? error.message : String(error));
  }
  for (const step of steps) {
    if (!phaseIds.has(step.phaseId)) errors.push(`${step.title}: unknown phase ${step.phaseId}.`);
    if (step.assignmentStrategy === 'EXISTING' && !step.agentId)
      errors.push(`${step.title}: EXISTING assignment requires an Agent.`);
    if (
      (step.assignmentStrategy === 'CREATE' || step.assignmentStrategy === 'AUTO') &&
      step.agentId
    )
      errors.push(`${step.title}: ${step.assignmentStrategy} assignment cannot fix an AgentID.`);
    if (step.assignmentStrategy === 'CREATE' && !step.role.trim())
      errors.push(`${step.title}: CREATE assignment requires a role.`);
    if (step.resourcePolicy === 'MANUAL' && !step.provider && !step.profile)
      errors.push(`${step.title}: MANUAL resource policy requires provider/profile.`);
  }
  return [...new Set(errors)];
}

export function updateStepLocalized(
  flow: FlowDraftModel,
  stepId: string,
  patch: Partial<FlowStepModel>,
): FlowDraftModel {
  return {
    ...flow,
    steps: flow.steps.map((step) => (step.id === stepId ? { ...step, ...patch } : step)),
  };
}
export function removeStepLocalized(flow: FlowDraftModel, stepId: string): FlowDraftModel {
  return {
    ...flow,
    steps: flow.steps
      .filter((step) => step.id !== stepId)
      .map((step) => ({
        ...step,
        dependencies: (step.dependencies || []).filter((dep) => dep !== stepId),
      })),
  };
}
export function splitStepLocalized(
  flow: FlowDraftModel,
  stepId: string,
  ids: [string, string],
): FlowDraftModel {
  const target = flow.steps.find((step) => step.id === stepId);
  if (!target) return flow;
  const [firstId, secondId] = ids;
  const first = {
    ...target,
    id: firstId,
    title: `${target.title} · 1`,
    dependencies: clone(target.dependencies),
  };
  const second = {
    ...target,
    id: secondId,
    title: `${target.title} · 2`,
    order: target.order + 1,
    dependencies: [firstId],
  };
  return {
    ...flow,
    steps: flow.steps.flatMap((step) =>
      step.id === stepId
        ? [first, second]
        : [
            {
              ...step,
              dependencies: (step.dependencies || []).map((dep) =>
                dep === stepId ? secondId : dep,
              ),
            },
          ],
    ),
  };
}

export function expandStepLocalized(flow: FlowDraftModel, stepId: string): FlowDraftModel {
  const step = flow.steps.find((candidate) => candidate.id === stepId);
  if (!step) return flow;
  const evidence = `Capture verification evidence for: ${step.title}.`;
  const currentAcceptance = Array.isArray(step.acceptanceCriteria) ? step.acceptanceCriteria : [];
  const acceptanceCriteria = currentAcceptance.includes(evidence)
    ? clone(currentAcceptance)
    : [...currentAcceptance, evidence];
  const taskRequirements =
    step.taskRequirements?.trim() ||
    `Work only on the approved Flow Step: ${step.goal || step.title}`;
  return updateStepLocalized(flow, stepId, { acceptanceCriteria, taskRequirements });
}

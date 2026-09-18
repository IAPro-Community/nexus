# Frontend/UX red-team — production go/no-go

- Audited HEAD: `e53f8352e4c3647632ebac7877cf0a6a3bd62647`
- Scope: onboarding, project/provider selection, Attention Center, Mission progress, error recovery, i18n in PlanBuilder/FlowRun, and `AGENTS.md` compliance.
- Method: source inspection plus the synthetic rendered captures whose manifest identifies the audited SHA. Cosmetic debt was not treated as a release blocker.

## Web usability verdict: PARTIAL

## Direct answer

For the happy path, **no**: a developer can add a repository using the OS browser/scanner, select a project, open a terminal, and start a direct session when an authenticated account is already available.

For first-time setup and blocked-Mission recovery, **yes, currently some internal architecture must be understood**. The user has to infer the distinction between direct provider accounts and “Nexus Intelligence”, know where “Usage” lives and how an external coding CLI is authenticated, and understand that a blocked Flow Run may need to be resolved in the Attention Center rather than on the run page. Those gaps prevent a clean production-certification PASS.

## Findings

### HIGH — UX-01: first AI session has a provider-setup dead end

When no eligible provider account exists, the direct-session dialog shows “Nenhuma conta de IA utilizável” and says to install/authenticate a CLI, but provides no action to open Usage, provider setup, diagnostics, or instructions. The primary Start action remains disabled (`web/src/features/work/DirectSessionLauncher.tsx:154-161,254-261`). The rendered `direct-session.png` capture confirms this dead-end state.

The Intelligence selector repeats the same pattern: “Autentique um CLI em Usage” without navigation (`web/src/features/settings/IntelligenceProviderCombo.tsx:49-52`). Settings separately exposes OFF/CLI/API and low-level fields such as base URL, environment-variable name and secret-file path (`web/src/features/settings/SettingsSurface.tsx:564-672`). A new user must already know:

- that direct sessions and Intelligence are different provider paths;
- what “Usage” means and where it is;
- which external CLI to install and how Nexus discovers its authentication/profile.

This can prevent the core AI workflow, not merely make it less polished.

### HIGH — UX-02: onboarding exists but is not first-run onboarding

The Welcome modal and product tour contain useful overview/quick-start content, but neither is automatically opened. `tourKey` is only written when the tour closes and is never read to decide whether onboarding should start (`web/src/app/NexusWorkspaceApp.tsx:86-87,1274-1281`). Users reach Welcome/Tour only through Help, a command, or an explicit route.

The no-project Project Hub successfully explains repository creation and offers OS browse/scan actions (`web/src/features/projects/ProjectHub.tsx:69-131`), but after creation the user lands in the workspace without a guided bridge between Project, Agent, direct session, Composer, Flow Run, Attention Center, provider account, and Intelligence. This makes the product taxonomy—not software development itself—a prerequisite for normal use.

### HIGH — UX-03: a blocked Flow Run does not present the required decision

The Attention Center is the strongest recovery surface: it sanitizes external text, shows the question/context/impact, and renders explicit intervention options (`web/src/features/work/AttentionCenter.tsx:31-77,202-243`).

However, the Flow Run page—the destination used to inspect Mission progress—only says “Human decision required” and shows the paused reason. It does not render intervention options or link back to the Attention Center (`web/src/features/work/FlowRunSurface.tsx:174-183`). For `BLOCKED_NEEDS_USER`, its main recovery control is “Resume / Return”, which invokes `returnToMission` rather than resolving the required decision (`web/src/features/work/FlowRunSurface.tsx:333-347`). The user must know the internal split between execution state and intervention resolution. This is a material recovery-path defect.

### MEDIUM — UX-04: Mission progress exposes implementation vocabulary instead of user meaning

Progress is visible and refreshes automatically, but the run surface exposes raw concepts such as execution snapshot IDs, iteration budget, policy enums, assigned-agent IDs, package dependency IDs, Capsules, Work Receipts and raw state values (`web/src/features/work/FlowRunSurface.tsx:157-208,210-289`). Error messages are also displayed directly from backend exceptions (`web/src/features/work/FlowRunSurface.tsx:73-86,115-127`).

An experienced Nexus developer can infer these values; a regular developer cannot always answer “what is happening, what should I do, and is my work safe?” without understanding the Mission/WorkPlan runtime model.

### MEDIUM — UX-05: errors are visible but recovery is inconsistent

Positive: async failures are generally surfaced, the app-level boundary offers Retry, and session expiry offers Reload (`web/src/components/ErrorBoundary.tsx:33-47`; `web/src/app/NexusWorkspaceApp.tsx:129-150`).

Residual problems:

- many project/plan operations only log failures to the console, including plan loading, revisions, package edits, scheduling, cancellation and takeover (`web/src/features/work/PlanBuilderSurface.tsx:148-176,442-548,804-869`);
- project OS-launch failures are console-only (`web/src/features/projects/ProjectManagerSurface.tsx:207-211`);
- provider and plan errors expose raw backend text and tell users to configure Settings without a direct recovery action (`web/src/features/work/PlanBuilderSurface.tsx:1127-1131`);
- Attention Center’s error state has no visible Retry action, although background polling retries every 10 seconds (`web/src/features/work/AttentionCenter.tsx:89-109,148-157`).

These do not block every happy path, but they make failures require console knowledge or architectural guessing.

### MEDIUM — UX-06: localization breaks in the most complex workflows

`FlowRunSurface` does not use i18n at all; all headings, states, actions, empty/error copy and evidence labels are hardcoded English (`web/src/features/work/FlowRunSurface.tsx:145-365`). `PlanBuilderSurface` uses `useTranslation` minimally but contains extensive hardcoded Portuguese/English copy in generation, clarification, preflight, policy, scheduling, errors and actions (`web/src/features/work/PlanBuilderSurface.tsx:302-412,872-923,927-1237,1240-2109`).

The result is a mixed-language critical workflow even though English, Portuguese and Spanish resources exist. This violates `AGENTS.md`’s zero-hardcoded-visible-string rule and can materially reduce comprehension; it is not classified HIGH because the controls remain operable.

### MEDIUM — UX-07: project selection works, but some alternatives are pointer-only

The canonical project path is understandable: browse/scan, explicit project cards, active state, and Switch/Open actions are present. The primary actions are buttons.

Still, project cards/table rows, path-copy regions and starter-template cards attach `onClick` to `div`/`tr`-based surfaces (`web/src/features/projects/ProjectManagerSurface.tsx:348-458,461-546,572-640`; `web/src/features/projects/ProjectHub.tsx:134-179`). Starter templates therefore have no equivalent keyboard control. This violates semantic/native-first requirements, but does not block repository creation or project switching because accessible primary buttons remain available.

## `AGENTS.md` compliance assessment

The audited frontend does not comply with the repository’s prescribed contract:

- **i18n:** hardcoded visible strings throughout PlanBuilder, FlowRun, provider/Intelligence setup, templates and parts of Settings.
- **SCSS Modules / inline styles:** `PlanBuilderSurface.tsx` contains extensive static inline presentation despite having a colocated module; `ProvidersView.tsx` also uses static inline styles (`web/src/components/ProvidersView.tsx:13-143`).
- **Semantic HTML / keyboard:** clickable `div`, `Card` (implemented as a `div`) and table-row interactions are used for project/template selection; PlanBuilder’s compatibility reorder path is drag-only.
- **TypeScript:** explicit `any` remains in PlanBuilder compiled prompt state, project casts/catches, and agent configuration (`web/src/features/work/PlanBuilderSurface.tsx:87`; `web/src/features/projects/ProjectManagerSurface.tsx:162,325`; `web/src/features/agents/NewAgentModal.tsx:261`).
- **Component limits:** `PlanBuilderSurface.tsx` is 2,110 lines, `ProjectManagerSurface.tsx` is 769 lines, and `NewAgentModal.tsx` is 431 lines, beyond the documented decomposition thresholds.

These violations increase regression and accessibility risk. They are not independently promoted to HIGH where they only represent maintainability or cosmetic debt; the HIGH findings above are based on concrete user-path impact.

## What already works

- Project creation offers native OS browsing, an in-app directory browser, scanning, manual path entry, and clear busy/error states.
- Direct-session provider choices, when present, communicate health, authentication and quota state and use a keyboard-operable radio-group pattern.
- Attention Center decisions are actionable in place and use sanitized external text.
- Flow history has useful empty states that route users back to Composer/drafts.
- Existing certification evidence reports responsive, accessibility and visual smoke gates passing; this review does not dispute those automated checks, but they do not cover the comprehension/recovery gaps above.

## Certification recommendation

Do not grant Web usability PASS yet. A PASS requires, at minimum:

1. an actionable first-provider setup/recovery path from every empty/error state;
2. automatic or explicit first-run onboarding that explains the product model without assuming Nexus terminology;
3. intervention choices, or a direct Attention Center action, on blocked Flow Runs;
4. localization of PlanBuilder and FlowRun critical copy.


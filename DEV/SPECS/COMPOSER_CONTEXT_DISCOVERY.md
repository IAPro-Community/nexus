# Composer Context Discovery — Nexus 1.0

Status: implementation in `feat/nexus-1.0-closure`; same-SHA certification pending.

## 1. Purpose

Make Composer context selection explicit, bounded and inspectable before a prompt
becomes a Flow. The design borrows the useful context-first discipline observed in
Freebuff without importing its runtime, programmable agents or executable-skill model.

## 2. Ownership

No new orchestration engine is introduced.

| Concern | Canonical owner |
| --- | --- |
| Conversation / LivingBrief | Composer |
| Static repository facts | Project Intelligence / ContextSnapshot |
| Reusable preferences | IntelligenceMemory |
| Skill discovery | Nexus Skill Catalog |
| Agent selection | MatchAgents |
| Provider/profile/model | ResourceScheduler |
| Execution | MissionRunner |
| Historical execution proof | EvidenceBundle / Mission stores |
| Method/gates when available | Maestro |
| Context selection explanation | ComposerContextManifest |

## 3. Backend flow

```text
Create/continue Composer session
        |
        +-- Context Readiness / Git identity
        +-- Memory Recall
        +-- Project Intelligence
        +-- explicit referenced files
        +-- Skill Catalog ranking
        +-- recent Mission references
        |
        v
ComposerContextManifest
        |
        +-- KEEP ------------> intelligence provider
        +-- REFERENCE_ONLY --> UI / later explicit resolution
        +-- DROP ------------> UI / audit only
        |
        v
LivingBrief -> PromptArtifact(context manifest frozen) -> Flow
```

The provider never receives raw Mission transcripts from this phase. Mission entries
are pointers with run id, plan id, state and timestamp. Skills are metadata candidates;
their instructions are loaded only through the existing explicit skill selection path.

The immutable PromptArtifact stores the **full audit envelope** (brief + full manifest).
Its executable prompt is compiled from an execution brief in which referenced files
not classified as KEEP are removed. When that artifact becomes a Flow, structured
execution facts carry only the KEEP projection plus the manifest fingerprint. DROP and
REFERENCE_ONLY remain retrievable through the source artifact, never copied into the
Mission prompt.

## 4. Context budget

The 1.0 budget uses a deterministic conservative token estimate. It is a context-size
guard, not a billing measurement.

Priority:
1. explicit referenced files;
2. current project identity;
3. PROJECT memory;
4. INSTALL memory;
5. high/medium/low-confidence project facts;
6. skill candidates and prior Missions remain references.

When the KEEP budget is exhausted the item becomes DROP with an explicit reason.
Nothing is silently promoted, truncated into a success claim, or serialized as zero.

## 5. Frontend impact

The Composer Context tab exposes:
- estimated context budget used/max;
- KEEP / REFERENCE / DROP counts;
- retained items with source, locator and reason;
- Mission/Skill references separately from model context;
- excluded items and the reason for exclusion;
- discovery warnings;
- existing manual context-file controls and Project Intelligence inspector.

Skill recommendations are source-agnostic. The UI shows source, version, score,
reason and supports local search. The UI does not rank independently; it mirrors the
backend proposal order.

## 6. Flow impact

Context Discovery happens before brief extraction/clarification. It does not create
Agents or Missions and cannot execute project scripts.

Flow creation still consumes the finalized PromptArtifact. The PromptArtifact freezes:
- LivingBrief;
- selected skill IDs;
- ComposerContextManifest.

Execution Preview and Mission remain unchanged authorities for actual Agent/resource
selection and execution. Historical Mission context in Composer cannot bypass Preview,
GO, admission, budget or autonomy policy.

## 7. Security invariants

- No arbitrary scripts from discovered Skills are executed.
- No credentials, .env contents or raw private terminal transcripts are added.
- User-attached file excerpts remain bounded and redacted by the existing context-file path.
- Mission history is reference-only.
- Current user input overrides memory.
- Missing context remains missing/unknown; it is never fabricated as evidence.

## 8. Acceptance

- session manifest survives restart;
- every included/excluded item has provenance and reason;
- model input contains KEEP only;
- context budget never exceeds its declared limit;
- recent Mission records cannot leak payload/transcript data;
- relevant skills are ranked deterministically and bounded;
- Context tab renders manifest state;
- final PromptArtifact contains the manifest used;
- Composer vague/full-prompt E2E paths continue through Flow and Execution Preview.

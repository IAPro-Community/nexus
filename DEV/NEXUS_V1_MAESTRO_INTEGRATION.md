# Nexus V1 — Maestro Integration

## Checkpoint corrente — 2026-11-08

No Maestro `release/1.1.0@9c9c9eb`, os contratos read-only existem: `capabilities --json` anuncia `contracts.advice=1.0.0` e `contracts.verification=1.0.0`; `advise --json` delega seleção ao `SkillRouterV3`; `verify --json` avalia resultados explicitamente marcados `PASS:`, referências de evidência e sinais de risco sem executar comandos nem abrir referências. `verify --strict` continua sendo o verificador legado de instalação. Suíte completa PASS (1.369/0 failures/4 skipped), `npm run validate` e validação pública também PASS.

Nexus `release/v1.0.0@b5cfe715` agora exige a versão anunciada antes de chamar advice/verification, aceita advice válido sem match como resultado vazio não degradado, recusa intent secret-like antes de iniciar o CLI e expõe `POST /api/v1/maestro/verify` autenticado. `go test`/`go vet` dos pacotes alterados passaram, e o teste cruzado Go→CLI passou em `f2f29cd0` antes da normalização mecânica de formatação. `make format-check` passou em `b5cfe715`; aguarda-se o gate completo e repetir o teste cruzado neste SHA. Não houve migration de memória, homologação do ownership de todos os flows, provider real ou GO da release; M9 global permanece **NO_GO**.

Charter §5, §52-58, §103, §134-136. Maestro is a **supported dependency** of Nexus,
never the reverse. Nexus consumes a machine-readable contract; it does not copy
Maestro knowledge.

## Dependency contract

- `Orquestrador-Maestro` = 100% COMMUNITY, owns methodology/skills/process/risk/gates/verification.
- Nexus installer/doctor must: detect Maestro → read version/capabilities → verify
  compatibility → install/update when explicitly safe (§5).
- If Maestro is unavailable at runtime → **`MAESTRO_DEGRADED`**: agents, terminals,
  stop, recovery and runtime observation keep working; Nexus must not invent
  recommendations or simulate Maestro Assist (§5, §136).

## Maestro modes (project-scoped, §52)

| Mode | Behavior |
|---|---|
| OFF | Installed but no active recommendations |
| ASSIST | Default. Recommends skills/process/gates/verification; no mandatory DAG |
| ORCHESTRATE | Optional/BETA; activates Mission concepts |

## Machine-readable contract

Maestro is the **owner of the schemas** (§54). Versioned contract v1 must expose at
least:

```
MaestroVersion/Capabilities
AdviceRequest { project metadata, language/framework, git summary, changed paths,
                goal, risk signals, process state }   // NO secrets, §55
AdviceResponse { classification, risk, process, requiredSkills, recommendedSkills,
                 optionalSkills, gates, explanations }  // priorities: REQUIRED|RECOMMENDED|OPTIONAL + WHY
SkillRecommendation · ProcessRecommendation · RiskAssessment · VerificationRequirement
```

- Local transport preference: `stdin JSON → orquestrador-maestro → stdout JSON`,
  no mandatory daemon (§53).
- Nexus sends **safe context only**: never `.env`, credentials, OAuth, auth files or
  raw terminal transcripts (§55).
- UI: Project Maestro page + Recommendation Cards with `[Apply]`/`[Explain]`;
  compliance (Design/Plan/TDD/Review/Verification) shown only with real evidence (§57-58).

## Status

- Maestro CLI contract: **not yet implemented** (Maestro repo work, `feat/nexus-contracts-v1`).
- Nexus side: `maestro_mode` column + default `ASSIST` already in the Project model;
  recommendation UI placeholder ready; bridge + doctor detection are Gate 6.
- **Honest degraded fallback (implemented):** When Maestro binary is absent or
  `capabilities --json` fails, `MaestroClient` returns `Available: false` with
  explicit error. `GetAdvice` returns `Mode: MaestroOff` with empty recommendation
  lists and explanation — no hardcoded skills, gates, or processes are fabricated.

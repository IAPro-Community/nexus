# Smart Account Selection & Scheduling

**🇬🇧 English** | [🇧🇷 Português](account-selection.pt-BR.md) | [🇪🇸 Español](account-selection.es.md)

When you run `ai <provider>` (for example, `ai codex` or `ai claude`) without explicitly specifying a profile, `ai-cli` automatically selects the healthiest, highest-capacity account.

For Nexus, `nexus <provider>:balance` and `nexus <provider>:auto` rotate new launches round-robin across eligible profiles. The cursor is persistent per provider and serialized across separate Nexus processes. Automatic selection requires recent `LIVE` or trustworthy `CACHED` quota evidence. Accounts that are disabled, unauthenticated, cooling down, exhausted, `UNKNOWN`, `ERROR`, `ESTIMATED`, or `UNSUPPORTED` are skipped. An explicitly bound/default or named profile follows its explicit-selection policy without being ranked as measured capacity.

---

## 1. Scheduling Principles

- **Prioritize Capacity**: Choose the account with the highest available remaining quota.
- **Respect Rate Limits**: Automatically exclude accounts currently in cooldown.
- **Respect Workspace Bindings**: Prefer accounts bound to your active project workspace.
- **Fail closed on unverified quota**: `UNKNOWN`, `ERROR`, `ESTIMATED`, and `UNSUPPORTED` are not evidence for automatic account selection.
- **Never invent capacity**: opaque or estimated quota may be shown with its evidence label, but is not ranked or selected automatically.
- **Zero Infinite Loops**: Track attempted profiles in each execution cycle so errors fail fast if all accounts are exhausted.

---

## 2. The Scoring Formula

```text
Score(Profile P) =
    AvailabilityScore (100 if authenticated)
  + QuotaScore (+0 to +100 based on remaining %)
  + ProjectBindingBonus (+35 if bound to active workspace)
  + ConfiguredDefaultBonus (+10 if set as default)
  + PriorityWeight (+/- user configured priority)
  - RateLimitPenalty (-1000 if in active cooldown)
```

---

## 3. Explaining Decisions

You can inspect the exact reasoning of the scheduler at any time:

```bash
ai explain codex
```

Example Output:

```text
=== Smart Account Selection: CODEX ===

Selected: personal
Reason:   authenticated, 85% capacity remaining, configured default

Evaluation of all candidate profiles:
  • personal         score: 195.0 (authenticated, 85% capacity remaining, configured default)
  • work             score: 120.0 (authenticated, 20% capacity remaining)
  ✗ client           rejected: rate limited until 19:45 (HTTP 429 quota exceeded)
```

---

## 4. Workspace Bindings

Bind your project workspace to a specific profile:

```bash
cd ~/projects/client-repo
ai bind codex:client-account
```

Automatic switching between accounts is disabled by default. Enable it explicitly with `nexus config fallback enable`; turn it off with `nexus config fallback disable`. The opt-in permits a pre-dispatch switch when an account is busy, subject to identity and recent-quota checks. After a provider process has started, Nexus does not replay a quota/rate-limited operation because output and external effects may be ambiguous. A configured opt-in does not make unknown or stale quota eligible.

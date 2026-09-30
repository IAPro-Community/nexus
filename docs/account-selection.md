# Smart Account Selection & Scheduling

**🇬🇧 English** | [🇧🇷 Português](account-selection.pt-BR.md) | [🇪🇸 Español](account-selection.es.md)

When you run `ai <provider>` (for example, `ai codex` or `ai claude`) without explicitly specifying a profile, `ai-cli` automatically selects the healthiest, highest-capacity account.

---

## 1. Scheduling Principles

- **Prioritize Capacity**: Choose the account with the highest available remaining quota.
- **Respect Rate Limits**: Automatically exclude accounts currently in cooldown.
- **Respect Workspace Bindings**: Prefer accounts bound to your active project workspace.
- **Fail closed on `UNKNOWN`**: missing or untrusted evidence is not automatic capacity.
- **LRU for opaque capacity**: when a provider explicitly reports quota as `UNSUPPORTED`, healthy authenticated accounts remain routable and rotate deterministically without a fabricated percentage.
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

If the bound profile hits a rate limit and automatic fallback is enabled, `ai-cli` will temporarily fail over to your next-best account while notifying you.

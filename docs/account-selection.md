# Smart Account Selection & Scheduling

**🇬🇧 English** | [🇧🇷 Português](account-selection.pt-BR.md) | [🇪🇸 Español](account-selection.es.md)

When you run `ai <provider>` (for example, `ai codex` or `ai claude`) without explicitly specifying a profile, `ai-cli` automatically selects the healthiest, highest-capacity account.

---

## 1. Scheduling Principles

- **Prioritize Capacity**: Choose the account with the highest available remaining quota.
- **Respect Rate Limits**: Automatically exclude accounts currently in cooldown.
- **Respect Workspace Bindings**: Prefer accounts bound to your active project workspace.
- **LRU for Unknown Capacity**: When exact quota is `UNKNOWN` across healthy accounts, rotate evenly to distribute load.
- **Zero Infinite Loops**: Track attempted profiles in each execution cycle so errors fail fast if all accounts are exhausted.

---

## 2. The Scoring Formula

```text
Score(Profile P) =
    AvailabilityScore (100 if authenticated)
  + CapacityRank (provider/model-pool routing signal; not a displayed percentage)
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


## Quota percentage vs routing capacity

`quota_remaining` is always literal remaining capacity in the 0..1 range.
Multi-pool providers may have a routing-capacity rank that rewards multiple
usable model families; that rank is intentionally separate and must never be
formatted as a quota percentage. Fresh LIVE/CACHED evidence has precedence.
When every usable profile has only measured ESTIMATED evidence, the selector
uses those measurements rather than blind LRU; LRU is reserved for truly
unknown quota.

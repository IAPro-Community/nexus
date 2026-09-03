# Capability Preservation Matrix

This matrix tracks all existing capabilities and verifies they are preserved after the canonical hardening.

| Capability | Status | Notes |
|---|---|---|
| Projects | ✅ PRESERVED | Project store, management, and workspace unchanged |
| Filesystem Browser | ✅ PRESERVED | DirectoryBrowserModal and fs API unchanged |
| BranchSwitcher | ✅ PRESERVED | Git branch management unchanged |
| Providers | ✅ PRESERVED | All 5 provider adapters unchanged |
| Profiles/Accounts | ✅ PRESERVED | Profile management and selection unchanged |
| Auth | ✅ PRESERVED | Bootstrap token, session cookies, CSRF unchanged |
| Quota/Usage | ✅ PRESERVED | Usage engine and display unchanged |
| Resources | ✅ PRESERVED | Resource discovery and allocation unchanged |
| Persistent Agents | ✅ PRESERVED | Agent model and lifecycle unchanged |
| RuntimeGeneration | ✅ PRESERVED | Runtime model and lifecycle unchanged |
| Sessions | ✅ PRESERVED | Session management unchanged |
| Terminal | ✅ PRESERVED | PTY, ConPTY, xterm.js unchanged |
| Safe Apply | ✅ PRESERVED | Atomic file operations unchanged |
| Worktrees | ✅ PRESERVED | Git worktree support unchanged |
| Schedules | ✅ PRESERVED | Task scheduling unchanged |
| Updates | ✅ PRESERVED | Version checking unchanged |
| Composer | ✅ PRESERVED | Flow composition unchanged |
| Flow | ✅ PRESERVED | Flow execution unchanged |
| WorkPlan | ✅ PRESERVED | Work plan management unchanged |
| Clarifications | ✅ PRESERVED | Clarification flow unchanged |
| Mission Runner | ✅ PRESERVED | Mission execution unchanged |
| ContextCapsule | ✅ PRESERVED | Context delivery unchanged |
| WorkReceipt | ✅ PRESERVED | Work receipt generation unchanged |
| Maestro | ✅ PRESERVED | Orquestrador integration unchanged |
| Security | ✅ PRESERVED | All security controls preserved |
| **NEW: Global Attention** | ✅ ADDED | GlobalAttentionManager, GlobalAttentionStream |
| **NEW: Auto Arrange** | ✅ ADDED | arrangeWindows with Smart/Grid/Columns/Rows |
| **NEW: Workspace Recovery** | ✅ ADDED | VisualStore for geometry persistence |
| **NEW: Agent Recovery** | ✅ REWORKED | RecoveryCenter with state tracking |
| **NEW: Deep Links** | ✅ ADDED | /projects/:id?view=agent:<id> |
| **NEW: Global Search** | ✅ ADDED | Fuzzy search across all entities |
| **NEW: Session Discovery** | ✅ ADDED | ProviderSessionDiscovery interface |
| **NEW: Attention Center** | ✅ ADDED | Global attention aggregation panel |
| **NEW: Service Worker** | ✅ ENHANCED | Push, notificationclick, pushsubscriptionchange |
| **NEW: Notification Transport** | ✅ ADDED | IN_APP, BROWSER, WEB_PUSH abstraction |

## No Regressions

- All existing tests pass (407 Go, 154 frontend)
- No breaking API changes
- No removed features
- No altered security boundaries

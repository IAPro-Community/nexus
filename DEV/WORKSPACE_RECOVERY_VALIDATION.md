# Workspace Recovery Validation

## Components Implemented

### 1. Visual Store (`internal/control/workspace/visual_store.go`)
- Persists workspace geometry per project
- ViewGeometry: x, y, width, height, zIndex, minimized, maximized
- WorkspaceVisualState: version, projectId, activeView, presentationMode, focusedStack, views
- Atomic file writes with .tmp + rename
- Default store at ~/.local/share/nexus/workspace-visual.json

### 2. Recovery Center (`internal/control/workspace/recovery.go`)
- AgentRecoveryInfo with recovery state tracking
- Recovery states: RUNNING, RECOVERABLE, LOST, STOPPED
- Resume capability: SUPPORTED, PARTIAL, UNSUPPORTED, UNKNOWN
- Register, UpdateState, MarkRunning, MarkLost, MarkStopped
- RecoverableCount and AllRecoverable queries

### 3. Deep Links (`web/src/workspace/deepLink.ts`)
- Format: /projects/:projectId?view=agent:<id> or ?view=flow-run:<id>
- parseDeepLink and generateDeepLink functions
- applyDeepLink for workspace integration

### 4. Global Search (`web/src/workspace/globalSearch.ts`)
- Fuzzy matching across projects, agents, provider sessions, flow runs
- Relevance scoring (exact > starts-with > contains > fuzzy)
- Search results with action hints (open, focus, adopt)

## Test Coverage

### Backend (15 tests)
- `TestVisualStoreSaveAndGet` - Basic persistence
- `TestVisualStoreDelete` - Deletion
- `TestVisualStoreList` - Listing
- `TestVisualStorePersistence` - Cross-instance persistence
- `TestVisualStoreUpdatedAt` - Timestamp tracking
- `TestRecoveryCenterRegisterAndList` - Agent registration
- `TestRecoveryCenterRecoverableCount` - Counting
- `TestRecoveryCenterMarkRunning` - State transitions
- `TestRecoveryCenterMarkLost` - Lost state
- `TestRecoveryCenterAllRecoverable` - Filtering
- `TestSessionDiscoveryManagerRegisterAndDiscover` - Discovery registration
- `TestSessionDiscoveryManagerDiscoverByProvider` - Provider-specific discovery

### Frontend (5 tests)
- `parseDeepLink` - Project-only links
- `parseDeepLink` - Agent focus links
- `parseDeepLink` - Flow-run focus links
- `parseDeepLink` - Non-project paths return null
- `generateDeepLink` - Link generation

## Recovery Scenarios

### After UI Reload
1. VisualStore restores workspace geometry from disk
2. RecoveryCenter identifies agents with RECOVERABLE state
3. User can Recover, Start Fresh, or Dismiss each agent

### After Nexus Restart
1. VisualStore restores all project workspaces
2. RecoveryCenter marks all previous runtimes as RECOVERABLE/LOST
3. Provider-specific resume attempted if capability is SUPPORTED
4. Project Shells marked as LOST (cannot resume shell processes)

### After Runtime Crash
1. Attention event emitted with StateFailed/StateRecoverable
2. RecoveryCenter updated with crash info
3. User notified via Attention Center

## Verification

All 15 workspace tests pass. All 5 frontend tests pass.

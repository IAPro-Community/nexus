# Global Attention Validation

## Components Implemented

### 1. Attention Types (`internal/nexus/attention/types.go`)
- States: WORKING, WAITING_INPUT, WAITING_APPROVAL, COMPLETED, FAILED, RATE_LIMITED, RECOVERABLE
- Sources: STRUCTURED_PROVIDER_EVENT, PROVIDER_HOOK, TERMINAL_HEURISTIC, PROCESS_EXIT, FLOW_RUNNER, SYSTEM
- AttentionEvent with bounded/redacted messages
- ProjectAttentionSummary with priority scoring

### 2. Attention Stream (`internal/nexus/attention/stream.go`)
- MemoryStream with configurable capacity
- Idempotent Ingest (dedup by project+event ID)
- Acknowledge with timestamp
- Replay with project/agent/runtime/flow filtering
- ProjectSummary with latest-state-per-subject aggregation

### 3. Global Attention Manager (`internal/nexus/attention/manager.go`)
- Singleton pattern with DefaultManager()
- Project-specific and global subscriptions
- Non-blocking event delivery (drop on full channel)
- AllProjectSummaries for cross-project view

### 4. Global Attention Stream (`internal/nexus/attention/global_stream.go`)
- Single WebSocket connection per browser
- Replay + live event delivery
- Deduplication by project ID
- Connection lifecycle management

### 5. Notification Policy (Service Worker)
- IN_APP: visible + focused page
- BROWSER: hidden/minimized page
- WEB_PUSH: no active clients
- Never double-notifies for same attentionEventId

## Test Coverage

- `TestNormalizeEventSanitizesBoundedRedactedMessage` - Message sanitization
- `TestNormalizeEventSanitizesBoundedRedactedReason` - Reason sanitization
- `TestNormalizeEventRejectsInvalidStateAndSource` - Validation
- `TestMemoryStreamIngestIsIdempotentAndReplayIsBounded` - Dedup and replay
- `TestMemoryStreamRetainsSameEventIDAcrossProjects` - Cross-project isolation
- `TestProjectSummaryCountsLatestStatePerLogicalSubject` - Aggregation
- `TestAcknowledgeCompletedEventDropsUnreadCompletedPriority` - Ack behavior
- `TestAcknowledgeTargetsMatchingProjectAndEventID` - Targeted ack
- `TestReplayFiltersByProjectAndLimit` - Filtering
- `TestMemoryStreamOutputsDoNotExposeStoredAcknowledgedAtPointers` - Immutability
- `TestManagerIngestAndSubscribe` - Subscription delivery
- `TestManagerIngestDuplicateIsIdempotent` - Manager dedup
- `TestManagerProjectSummary` - Manager summary
- `TestManagerSubscribeAll` - Global subscription
- `TestManagerUnsubscribe` - Cleanup
- `TestManagerAllProjectSummaries` - Multi-project summary

## Verification

All 16 attention tests pass. All 407 Go tests pass. All 154 frontend tests pass.

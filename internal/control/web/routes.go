package web

import "net/http"

// routeDependencies keeps route composition explicit while allowing Server to
// remain responsible only for listener and middleware lifecycle.
type routeDependencies struct {
	server       *Server
	api          *APIHandler
	nexusHandler *NexusHandler
}

// registerRoutes composes the versioned API from cohesive route groups. The
// handlers remain in their existing packages and contracts; this boundary only
// owns transport registration and middleware placement.
func registerRoutes(mux *http.ServeMux, deps routeDependencies) {
	registerControlRoutes(mux, deps)
	registerNexusRoutes(mux, deps)
	registerMaestroRoutes(mux, deps)
	registerPlanningRoutes(mux, deps)
	registerMissionRoutes(mux, deps)
	registerSystemRoutes(mux, deps)
	registerFilesystemRoutes(mux, deps)
	registerTunnelRoutes(mux, deps)
	registerBrowserRoutes(mux, deps)
}

func registerControlRoutes(mux *http.ServeMux, deps routeDependencies) {
	s, api := deps.server, deps.api
	mux.HandleFunc("/api/v1/health", api.handleHealth)
	mux.HandleFunc("/api/v1/session", api.handleSession)
	mux.HandleFunc("/api/v1/auth/bootstrap", s.handleAuthBootstrap)
	mux.HandleFunc("/api/v1/auth/loopback-reauth", s.handleAuthLoopbackReauth)
	mux.HandleFunc("/api/v1/desktop/bootstrap", s.handleDesktopBootstrap)
	mux.HandleFunc("/api/v1/session/rotate", s.authMiddleware(s.handleSessionRotate))
	mux.HandleFunc("/api/v1/session/logout", s.authMiddleware(s.handleSessionLogout))
	mux.HandleFunc("/api/v1/workspaces", s.authMiddleware(api.handleWorkspaces))
	mux.HandleFunc("/api/v1/runtimes", s.authMiddleware(api.handleRuntimes))
	mux.HandleFunc("/api/v1/runtimes/", s.routeRuntime(deps.nexusHandler))
	mux.HandleFunc("/api/v1/providers", s.authMiddleware(api.handleProviders))
	mux.HandleFunc("/api/v1/profiles", s.authMiddleware(api.handleProfiles))
	mux.HandleFunc("/api/v1/profiles/import-host-sessions", s.authMiddleware(api.handleHostSessionImport))
	mux.HandleFunc("/api/v1/profiles/default", s.authMiddleware(api.handleProfileDefault))
	mux.HandleFunc("/api/v1/profiles/logout", s.authMiddleware(api.handleProfileLogout))
	mux.HandleFunc("/api/v1/events", s.authMiddleware(api.handleEvents))
}

func registerNexusRoutes(mux *http.ServeMux, deps routeDependencies) {
	s, h := deps.server, deps.nexusHandler
	mux.HandleFunc("/api/v1/projects", s.authMiddleware(h.handleProjectsList))
	mux.HandleFunc("/api/v1/projects/summaries", s.authMiddleware(h.handleProjectSummaries))
	mux.HandleFunc("/api/v1/projects/", s.routeProject(h))
	mux.HandleFunc("/api/v1/agent-modes", s.authMiddleware(h.handleAgentModes))
	mux.HandleFunc("/api/v1/execution-roles", s.authMiddleware(h.handleExecutionRoles))
	mux.HandleFunc("/api/v1/agent-projections/", s.authMiddleware(h.handleAgentProjections))
	mux.HandleFunc("/api/v1/agents/", s.routeAgent(h))
	mux.HandleFunc("/api/v1/resources", s.authMiddleware(h.handleResourcesList))
	mux.HandleFunc("/api/v1/resources/refresh", s.authMiddleware(h.handleResourcesRefresh))
	mux.HandleFunc("/api/v1/resources/select", s.authMiddleware(h.handleResourceSelect))
	mux.HandleFunc("/api/v1/resources/recommend", s.authMiddleware(h.handleResourceRecommend))
	mux.HandleFunc("/api/v1/desktop/commands", s.authMiddleware(h.handleDesktopCommand))
}

func registerMaestroRoutes(mux *http.ServeMux, deps routeDependencies) {
	s, h := deps.server, deps.nexusHandler
	mux.HandleFunc("/api/v1/maestro", s.authMiddleware(h.handleMaestroStatus))
	mux.HandleFunc("/api/v1/maestro/advice", s.authMiddleware(h.handleMaestroAdvice))
	mux.HandleFunc("/api/v1/maestro/catalog", s.authMiddleware(h.handleMaestroCatalog))
	mux.HandleFunc("/api/v1/maestro/sync/preview", s.authMiddleware(h.handleMaestroSyncPreview))
	mux.HandleFunc("/api/v1/maestro/sync", s.authMiddleware(h.handleMaestroSync))
	mux.HandleFunc("/api/v1/maestro/update", s.authMiddleware(h.handleMaestroUpdate))
}

func registerPlanningRoutes(mux *http.ServeMux, deps routeDependencies) {
	s, h := deps.server, deps.nexusHandler
	mux.HandleFunc("/api/v1/intelligence", s.authMiddleware(h.handleIntelligence))
	mux.HandleFunc("/api/v1/flow/generate", s.authMiddleware(h.handleFlowGenerate))
	mux.HandleFunc("/api/v1/intelligence/probe", s.authMiddleware(h.handleIntelligenceProbe))
	mux.HandleFunc("/api/v1/intelligence/memories", s.authMiddleware(h.handleIntelligenceMemories))
	mux.HandleFunc("/api/v1/intelligence/memories/recall", s.authMiddleware(h.handleIntelligenceMemoryRecall))
	mux.HandleFunc("/api/v1/intelligence/memories/reflect", s.authMiddleware(h.handleIntelligenceMemoryReflect))
	mux.HandleFunc("/api/v1/intelligence/memories/", s.authMiddleware(h.handleIntelligenceMemoryDelete))
	mux.HandleFunc("/api/v1/clarifications/", s.authMiddleware(h.handleClarification))
	mux.HandleFunc("/api/v1/composer-sessions/", s.authMiddleware(h.handleComposerSession))
	mux.HandleFunc("/api/v1/attachments/", s.authMiddleware(h.handleAttachmentByID))
	mux.HandleFunc("/api/v1/prompt-artifacts/", s.authMiddleware(h.handlePromptArtifact))
	mux.HandleFunc("/api/v1/plans/", s.routePlan(h))
}

func registerMissionRoutes(mux *http.ServeMux, deps routeDependencies) {
	s, h := deps.server, deps.nexusHandler
	mux.HandleFunc("/api/v1/runs", s.authMiddleware(h.handleRunsList))
	mux.HandleFunc("/api/v1/runs/", s.routeRun(h))
	mux.HandleFunc("/api/v1/schedules", s.authMiddleware(h.handleMissionSchedules))
	mux.HandleFunc("/api/v1/attention", s.authMiddleware(h.handleAttentionCenter))
}

func registerSystemRoutes(mux *http.ServeMux, deps routeDependencies) {
	s, h := deps.server, deps.nexusHandler
	mux.HandleFunc("/api/v1/system/info", s.authMiddleware(deps.api.handleSystemInfo))
	mux.HandleFunc("/api/v1/system/doctor", s.authMiddleware(h.handleSystemDoctor))
	mux.HandleFunc("/api/v1/system/updates", s.authMiddleware(h.handleSystemUpdates))
	mux.HandleFunc("/api/v1/telemetry/stats", s.authMiddleware(h.handleTelemetryStats))
	mux.HandleFunc("/api/v1/telemetry/history", s.authMiddleware(h.handleTelemetryHistory))
	mux.HandleFunc("/api/v1/capacity/status", s.authMiddleware(h.handleCapacityStatus))
	mux.HandleFunc("/api/v1/capacity/history", s.authMiddleware(h.handleCapacityHistory))
	mux.HandleFunc("/api/v1/capacity/pace.csv", s.authMiddleware(h.handleCapacityPaceCSV))
	mux.HandleFunc("/api/v1/providers/public-status", s.authMiddleware(h.handlePublicServiceStatus))
	mux.HandleFunc("/api/v1/capacity/preferences", s.authMiddleware(h.handleCapacityPreferences))
	mux.HandleFunc("/api/v1/capacity/ack", s.authMiddleware(h.handleCapacityAck))
	mux.HandleFunc("/api/v1/capacity/snooze", s.authMiddleware(h.handleCapacitySnooze))
	mux.HandleFunc("/api/v1/cost/summary", s.authMiddleware(h.handleCostSummary))
	mux.HandleFunc("/api/v1/cost/observations", s.authMiddleware(h.handleCostObservations))
	mux.HandleFunc("/api/v1/cost/observations.csv", s.authMiddleware(h.handleCostCSV))
	mux.HandleFunc("/api/v1/cost/external-claude", s.authMiddleware(h.handleExternalClaudeUsage))
	mux.HandleFunc("/api/v1/cost/preferences", s.authMiddleware(h.handleCostPreferences))
	mux.HandleFunc("/api/v1/hud", s.authMiddleware(h.handleHUDState))
	mux.HandleFunc("/api/v1/voice/settings", s.authMiddleware(h.handleVoiceSettings))
	mux.HandleFunc("/api/v1/voice/transcribe", s.authMiddleware(h.handleVoiceTranscribe))
}

func registerFilesystemRoutes(mux *http.ServeMux, deps routeDependencies) {
	s, h := deps.server, deps.nexusHandler
	mux.HandleFunc("/api/v1/fs/browse", s.authMiddleware(h.handleFSBrowse))
	mux.HandleFunc("/api/v1/fs/scan", s.authMiddleware(h.handleFSScan))
	mux.HandleFunc("/api/v1/fs/inspect", s.authMiddleware(h.handleFSInspect))
	mux.HandleFunc("/api/v1/fs/mkdir", s.authMiddleware(h.handleFSMkdir))
}

func registerTunnelRoutes(mux *http.ServeMux, deps routeDependencies) {
	s := deps.server
	mux.HandleFunc("/api/v1/tunnel/status", s.authMiddleware(s.handleTunnelStatus))
	mux.HandleFunc("/api/v1/tunnel/start", s.authMiddleware(s.handleTunnelStart))
	mux.HandleFunc("/api/v1/tunnel/stop", s.authMiddleware(s.handleTunnelStop))
	mux.HandleFunc("/api/v1/tunnel/qr", s.authMiddleware(s.handleTunnelQR))
	mux.HandleFunc("/api/v1/tunnel/invite", s.authMiddleware(s.handleTunnelQR))
}

// Browser is a workspace capability, not a second orchestration layer.
func registerBrowserRoutes(mux *http.ServeMux, deps routeDependencies) {
	s := deps.server
	mux.HandleFunc("/api/v1/browser/mcp", s.handleBrowserMCP)
	mux.HandleFunc("/api/v1/browser/sessions", s.authMiddleware(s.handleBrowserSessions))
	mux.HandleFunc("/api/v1/browser/sessions/", s.authMiddleware(s.handleBrowserSession))
}

import type { NexOperationalMood } from './motion/nexBehaviorState';

export type NexV2PresetId = 'compact' | 'standard' | 'large';
export type NexV2TabId = 'chat' | 'resources' | 'attention';

export interface NexV2PlacementState {
  displayId: string;
  anchor: string;
  offsetX: number;
  offsetY: number;
  presetId: NexV2PresetId | string;
  scaleFactor: number;
  alwaysOnTop: boolean;
}

export interface NexV2PresenceState {
  placement: NexV2PlacementState;
  showAvatar: boolean;
  compactBar: boolean;
  doNotDisturb: boolean;
  privacyMode: boolean;
  paused: boolean;
  reducedMotion: boolean;
  hotkey: string;
  respectFullscreen: boolean;
}

export interface NexV2PlatformCapabilities {
  globalPositioning: boolean;
  clickThrough: boolean;
  globalHotkey: boolean;
  globalHotkeyReason?: string;
  secondaryWindow: boolean;
  tray: boolean;
  autoStart: boolean;
  platform: string;
  session?: string;
}

export interface NexV2HudWindow {
  group?: string;
  kind: string;
  label?: string;
  remaining_percent?: number;
  reset_description?: string;
  reset_at?: string;
  status: string;
}

export interface NexV2HudProvider {
  provider: string;
  profile: string;
  display_name?: string;
  health: string;
  availability: string;
  capacity_state: string;
  source?: string;
  freshness: string;
  last_updated?: string;
  rate_limited: boolean;
  limiting_window?: NexV2HudWindow;
  windows: NexV2HudWindow[];
}

export interface NexV2HudAgent {
  id: string;
  project_id: string;
  name: string;
  role?: string;
  status: string;
  provider?: string;
  model?: string;
}

export interface NexV2HudMission {
  run_id: string;
  project_id: string;
  state: string;
  current_stage: string;
  current_task_id?: string;
  current_task_title?: string;
  provider?: string;
  profile?: string;
  completed_packages: number;
  total_packages: number;
  needs_user: boolean;
  intervention_reason?: string;
  updated_at?: string;
}

export interface NexV2HudAlert {
  fingerprint: string;
  category: string;
  severity: string;
  provider?: string;
  profile?: string;
  reason?: string;
  remaining_percent?: number;
  acked: boolean;
  snoozed_until?: string;
  last_observed_at: string;
}

export interface NexV2HudRuntimeAttention {
  runtime_id: string;
  agent_id?: string;
  project_id?: string;
  project_name?: string;
  provider: string;
  reason: string;
  prompt_kind: string;
  fingerprint: string;
  updated_at: string;
}

export interface NexV2HudState {
  generated_at: string;
  project_id?: string;
  providers: NexV2HudProvider[];
  agents: NexV2HudAgent[];
  missions: NexV2HudMission[];
  alerts: NexV2HudAlert[];
  runtime_attention: NexV2HudRuntimeAttention[];
  cost?: {
    evidence: string;
    token_evidence: string;
    known_microusd: number;
    unknown_observation_count: number;
    budget_level: string;
  };
  primary: { kind: string; id?: string };
  system: {
    capacity: string;
    resources: string;
    agents: string;
    missions: string;
    cost: string;
    cache_ttl_millis: number;
  };
}

export interface NexV2HudSnapshotResult {
  offline: boolean;
  enabled: boolean;
  state: NexV2HudState | null;
  error?: string;
}

export interface NexV2ChatSetupCandidate {
  mode: string;
  provider: string;
  profile?: string;
  model?: string;
  ready: boolean;
  read_only: boolean;
  installed: boolean;
  authenticated: boolean;
  quota_ok: boolean;
  reason?: string;
}

export interface NexV2ChatSetupAssistance {
  reason: string;
  can_auto_configure: boolean;
  recommended_mode?: string;
  recommended_provider?: string;
  recommended_profile?: string;
  recommended_action?: string;
  candidates?: NexV2ChatSetupCandidate[];
}

export interface AutoConfigureIntelligencePayload {
  mode: 'AUTO' | 'CLI' | 'OPENAI_COMPATIBLE';
  provider?: string;
  profile?: string;
  base_url?: string;
  model?: string;
  api_key_env?: string;
  api_key_file?: string;
}

export interface NexV2ChatTurn {
  sequence?: number;
  role: 'user' | 'assistant';
  content: string;
  source?: string;
  created_at?: string;
  proposal?: NexV2ActionProposal;
  setup_assistance?: NexV2ChatSetupAssistance;
}

export interface NexV2ActionProposal {
  id?: string;
  kind?: string;
  title?: string;
  description?: string;
  target?: string;
  risk?: string;
  requires_confirmation?: boolean;
}

export interface NexV2ChatResponse {
  answer: string;
  source?: string;
  proposal?: NexV2ActionProposal;
  setup_assistance?: NexV2ChatSetupAssistance;
}

export interface NexV2PersonalMemoryEntry {
  id: string;
  key: string;
  value: string;
  scope: string;
  provenance: string;
  created_at: string;
  updated_at: string;
}

export interface NexV2ModelCatalogEntry {
  provider: string;
  profile: string;
  model_id: string;
  is_default: boolean;
  deprecated: boolean;
  source: string;
  observed_at: string;
}

export interface NexV2ModelCatalogChange {
  fingerprint: string;
  kind: 'ADDED' | 'REMOVED' | 'DEPRECATED' | 'DEFAULT_CHANGED' | 'UNKNOWN_SOURCE' | string;
  provider: string;
  profile: string;
  model_id: string;
  previous_default?: string;
  new_default?: string;
  source: string;
  observed_at: string;
}

export interface NexV2ModelCatalogWatchSnapshot {
  freshness: 'FRESH' | 'STALE' | 'UNKNOWN' | string;
  observed_at?: string;
  entries: NexV2ModelCatalogEntry[];
  changes: NexV2ModelCatalogChange[];
}

export interface NexV2EventEnvelope {
  revision: number;
  event_id: string;
  fingerprint: string;
  kind: string;
  severity: string;
  source: string;
  title: string;
  detail?: string;
  provider?: string;
  profile?: string;
  created_at: string;
}

export interface NexV2EventsPage {
  offline: boolean;
  revision: number;
  replay_truncated: boolean;
  events: NexV2EventEnvelope[];
  model_watch_freshness: string;
}

export interface NexV2PresenceMetricsSnapshot {
  panelOpen: boolean;
  panelOpenColdMs: number | null;
  panelOpenWarmMs: number | null;
  panelToggleCount: number;
  activeTab: NexV2TabId;
  settingsOpen: boolean;
  avatarVisible: boolean;
  compactBar: boolean;
  doNotDisturb: boolean;
  privacyMode: boolean;
  offline: boolean;
  mood: NexOperationalMood;
  chunkLoaded: boolean;
  eventRevision: number;
  modelWatchFreshness: string;
  backoffMs: number;
}

export interface NexV2ActionReceipt {
  receipt_id: string;
  action_id: string;
  risk: string;
  owner: string;
  status: 'PROPOSED' | 'DENIED' | 'EXECUTED' | 'FAILED' | 'UNAVAILABLE' | string;
  summary?: string;
  params?: Record<string, string>;
  created_at: string;
  completed_at?: string;
}

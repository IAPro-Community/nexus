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
  completed_packages: number;
  total_packages: number;
  needs_user: boolean;
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

export interface NexV2ChatTurn {
  role: 'user' | 'assistant';
  content: string;
  source?: string;
}

export interface NexV2ChatResponse {
  answer: string;
  source?: string;
  proposal?: {
    id?: string;
    kind?: string;
    title?: string;
    description?: string;
  };
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
  offline: boolean;
  mood: NexOperationalMood;
  chunkLoaded: boolean;
}

export interface AccountScope {
  provider_id: string;
  profile_id: string;
  account_id: string;
  identity_version: string;
  credential_scope?: string;
}

/* ---- Provider accounts, capacity, cost evidence and capabilities ---- */

export type ResourceChannel = 'SUBSCRIPTION' | 'API';

export interface ResourceChannelState {
  channel: ResourceChannel;
  quota?: {
    provider_id: string;
    profile_id: string;
    status: string;
    source: string;
    confidence?: string;
    fetched_at: string;
    windows?: Array<{
      kind: string;
      group?: string;
      used_percent?: number;
      remaining_percent?: number;
      reset_time?: string;
      reset_description?: string;
    }>;
  };
  consumption?: {
    input_tokens?: number;
    cached_tokens?: number;
    output_tokens?: number;
    reasoning_tokens?: number;
    requests?: number;
  };
  cost?: {
    amount?: number;
    currency?: string;
    budget?: number;
  };
  rate_limit?: {
    requests_remaining?: number;
    tokens_remaining?: number;
    reset_at?: string;
  };
}

export interface RegisteredProfile {
  provider: string;
  name: string;
  created_at?: string;
  account_scope?: AccountScope;
}

export type HostSessionImportStatus =
  | 'IMPORTED'
  | 'ALREADY_AUTHENTICATED'
  | 'NO_SESSION'
  | 'LOGIN_REQUIRED'
  | 'UNSUPPORTED'
  | 'FAILED'
  | 'NOT_INSTALLED';

export interface HostSessionImportResult {
  provider: string;
  profile?: string;
  status: HostSessionImportStatus;
}

export interface ProviderAccount {
  id: string;
  provider: string;
  profile: string;
  display_name: string;
  authenticated: boolean;
  is_default: boolean;
  available: boolean;
  capabilities?: Record<string, string>;
  resource_channels?: ResourceChannelState[];
  quota_remaining: number;
  quota_total: number;
  rate_limited: boolean;
  network_error?: boolean;
  health: string;
  last_checked: string;
  cooldown_until?: string;
  account_scope?: {
    provider_id?: string;
    account_id?: string;
    identity_version?: string;
  };
  /** Provenance of the profile credential, when it was adopted from the host CLI. */
  auth_source?: {
    origin?: string;
    linked_at?: string;
    account_id_hash?: string;
    plan?: string;
  };
  /** Set when the profile credential diverges from the host login account. */
  account_warning?: string;
  avail_reasons?: {
    exhausted_windows?: string[];
    rate_limited?: boolean;
    unknown_quota?: boolean;
    auth_required?: boolean;
    all_ok?: boolean;
  };
  eligibility?: {
    eligible: boolean;
    reason: string;
    confidence: string;
    message?: string;
  };
  quota_projection?: {
    truth_state:
      | 'confirmed'
      | 'stale'
      | 'exhausted'
      | 'rate_limited'
      | 'unknown'
      | 'unauthenticated'
      | 'unsupported';
    available: boolean;
    primary_available: boolean;
    auxiliary_only: boolean;
    active_tier?: 'primary' | 'auxiliary';
    primary_remaining?: number;
    auxiliary_remaining?: number;
    effective_remaining?: number;
    bottleneck_kind?: string;
    compact_summary?: string;
    diagnostic_code?: string;
    diagnostic_message?: string;
    freshness_seconds?: number;
    needs_reset_reprobe?: boolean;
    rank_score: number;
  };
  quota_view?: {
    status?: string;
    source?: string;
    plan?: string;
    account?: string;
    diagnostic_code?: string;
    diagnostic_message?: string;
    fetched_at?: string;
    available?: boolean;
    avail_reasons?: {
      exhausted_windows?: string[];
      rate_limited?: boolean;
      unknown_quota?: boolean;
      auth_required?: boolean;
      all_ok?: boolean;
    };
    model_groups?: Array<{
      key?: string;
      name?: string;
      tier?: 'primary' | 'auxiliary';
      windows?: Array<{
        kind?: string;
        label?: string;
        remaining?: number;
        reset_desc?: string;
        status?: string;
        bar?: string;
      }>;
    }>;
  };
}

export type CapacityMonitorState =
  'UNKNOWN' | 'HEALTHY' | 'WARNING' | 'CRITICAL' | 'EXHAUSTED' | 'RECOVERED';

export interface PublicServiceStatus {
  provider: string;
  indicator: 'none' | 'minor' | 'major' | 'critical' | 'UNKNOWN';
  detail?: string;
  checked_at?: string;
  stale: boolean;
}

export interface CapacityPaceForecast {
  status: 'RISING' | 'RESET_FIRST' | 'STABLE';
  confidence: 'LOW' | 'MEDIUM' | 'HIGH';
  samples: number;
  used_per_hour: number;
  exhausts_at?: string | null;
  reset_at: string;
}

export interface CapacityResourceStatus {
  key: string;
  provider: string;
  profile: string;
  quota_group: string;
  window_kind: string;
  state: CapacityMonitorState | string;
  fingerprint?: string;
  reason?: string;
  transitioned_at: string;
  last_observed_at: string;
  remaining_percent?: number | null;
  reset_description?: string;
  pace?: CapacityPaceForecast;
  acked: boolean;
  snoozed_until?: string | null;
}

export interface CapacityPreferences {
  enabled: boolean;
  background_enabled: boolean;
  warning_threshold: number;
  critical_threshold: number;
  exhausted_threshold: number;
  recovery_threshold: number;
  reminder_interval_seconds: number;
  quiet_hours_enabled: boolean;
  quiet_hours_start_utc: string;
  quiet_hours_end_utc: string;
  exhausted_bypass_quiet_hours: boolean;
  browser_notifications_enabled: boolean;
  native_notifications_enabled: boolean;
  lock_screen_safe_copy: boolean;
  notify_action_required: boolean;
  notify_auth_permission: boolean;
  notify_critical_failure: boolean;
  notify_quota_exhausted: boolean;
  notify_quota_low: boolean;
  notify_completion_recovery: boolean;
  notify_runtime_connection: boolean;
  updated_at: string;
}

export interface CapacityStatus {
  enabled: boolean;
  monitored_resources: number;
  resources: CapacityResourceStatus[];
  lease_owner?: string;
  lease_expires_at?: string;
  preferences: CapacityPreferences;
}

export interface CapacityHistoryEvent {
  id: string;
  fingerprint: string;
  event_type: string;
  provider: string;
  profile: string;
  quota_group: string;
  window_kind: string;
  severity: string;
  payload_json: string;
  emitted_at: string;
  acked: boolean;
  snoozed_until?: string | null;
}

export interface CapacityAlertState {
  fingerprint: string;
  acked: boolean;
  snoozed_until?: string | null;
  updated_at: string;
}

export type CostEvidenceState = 'EMPTY' | 'ATTRIBUTION_ONLY' | 'PARTIAL' | 'MEASURED';

export interface CostAnalyticsTotals {
  observation_count: number;
  invocations_total: number;
  invocations_failed: number;
  /** Fraction 0..1 of invocations classified as failed. Omitted when the
   * scope has no observations: missing evidence is never a fabricated 0. */
  error_rate?: number;
  /** Averages are present only when at least one observation actually
   * measured the evidence (duration_ms > 0; ttft_ms / output_tokens_per_second
   * NOT NULL). NULL evidence means "not measured", never zero. */
  avg_duration_ms?: number;
  avg_ttft_ms?: number;
  avg_output_tokens_per_second?: number;
  known_cost_observations: number;
  unknown_cost_observations: number;
  known_token_observations: number;
  reported_cost_observations: number;
  equivalent_cost_observations: number;
  inferred_cost_observations: number;
  known_cache_observations: number;
  cache_prompt_input_tokens: number;
  cache_prompt_read_tokens: number;
  input_tokens: number;
  cached_input_tokens: number;
  output_tokens: number;
  reasoning_tokens: number;
  duration_ms: number;
  provider_reported_microusd: number;
  api_equivalent_microusd: number;
  inferred_microusd: number;
  unclassified_microusd: number;
}

export interface DailyCostAnalytics extends CostAnalyticsTotals {
  date: string;
}

export interface ModelCostAnalytics extends CostAnalyticsTotals {
  provider: string;
  model: string;
}

export interface CostAnalyticsReport {
  since: string;
  generated_at: string;
  days: number;
  project_id?: string;
  source: 'LOCAL_COST_OBSERVATIONS';
  total: CostAnalyticsTotals;
  by_day: DailyCostAnalytics[];
  by_model: ModelCostAnalytics[];
  models_truncated: boolean;
}

export interface CostAggregate {
  observation_count: number;
  known_cost_observations: number;
  unknown_cost_observations: number;
  known_token_observations: number;
  cost_microusd: number;
  input_tokens: number;
  cached_input_tokens: number;
  output_tokens: number;
  reasoning_tokens: number;
  duration_ms: number;
}

export interface ProviderCostAggregate extends CostAggregate {
  provider: string;
  profile: string;
}

export interface ExternalClaudeSummary {
  source: 'LOCAL_DEFAULT_UNVERIFIED';
  unattributed: true;
  messages: number;
  input_tokens: number;
  output_tokens: number;
  cache_read_tokens: number;
  cache_write_tokens: number;
  last_observed_at?: string;
}

export interface ExternalClaudeScanResult {
  scan: { files: number; messages: number; bytes: number; has_more: boolean; unattributed: true };
  summary: ExternalClaudeSummary;
}

export interface CostPreferences {
  enabled: boolean;
  window_days: number;
  soft_limit_microusd: number;
  hard_limit_microusd: number;
  emergency_limit_microusd: number;
  anomaly_multiplier: number;
  anomaly_minimum_samples: number;
  anomaly_min_cost_microusd: number;
  updated_at: string;
}

export type CostBudgetLevel = 'DISABLED' | 'UNKNOWN' | 'OK' | 'SOFT' | 'HARD' | 'EMERGENCY';

export interface CostBudgetAssessment {
  level: CostBudgetLevel;
  /** Backward-compatible alias for provider-reported value, never estimates. */
  known_cost_microusd: number;
  reported_cost_microusd: number;
  api_equivalent_microusd: number;
  inferred_microusd: number;
  unclassified_microusd: number;
  reported_cost_observations: number;
  unverified_observations: number;
  /** Coverage of provider-reported observations, not invoices. */
  evidence: CostEvidenceState;
  active_limit_microusd?: number;
  requires_confirmation: boolean;
  message: string;
}

export type CostAnomalyStatus = 'UNKNOWN' | 'INSUFFICIENT' | 'NORMAL' | 'ANOMALY';

export interface CostAnomalyAssessment {
  status: CostAnomalyStatus;
  triggered: boolean;
  provider?: string;
  profile?: string;
  cost_kind?: CostObservation['cost_kind'];
  latest_cost_microusd?: number;
  baseline_median_microusd?: number;
  threshold_microusd?: number;
  historical_samples: number;
  observed_at?: string;
  message: string;
}

export interface CostSummary {
  since: string;
  generated_at: string;
  project_id?: string;
  aggregate: CostAggregate;
  by_provider: ProviderCostAggregate[];
  cost_evidence: CostEvidenceState;
  token_evidence: CostEvidenceState;
  preferences: CostPreferences;
  budget: CostBudgetAssessment;
  anomaly: CostAnomalyAssessment;
}

export interface CostObservation {
  id: string;
  project_id?: string;
  mission_run_id?: string;
  package_id?: string;
  invocation_id?: string;
  agent_id?: string;
  runtime_generation_id?: string;
  runtime_id?: string;
  provider?: string;
  profile?: string;
  model?: string;
  branch?: string;
  workspace_kind?: string;
  input_tokens?: number;
  cached_input_tokens?: number;
  output_tokens?: number;
  reasoning_tokens?: number;
  duration_ms: number;
  cost_microusd?: number;
  cost_kind: 'PROVIDER_REPORTED' | 'API_EQUIVALENT_ESTIMATE' | 'INFERRED' | 'UNKNOWN';
  source: string;
  status?: string;
  account_scope?: string;
  session_id?: string;
  /** Not-measured latency/throughput evidence stays absent, never zero. */
  ttft_ms?: number;
  output_tokens_per_second?: number;
  observed_at: string;
}

export interface CapabilityEvidence {
  status: 'SUPPORTED' | 'PARTIAL' | 'UNSUPPORTED' | 'UNKNOWN' | 'NOT_TESTED';
  provider_version?: string;
  mechanism?: string;
  reason?: string;
  tested: boolean;
}

export interface EffectiveCapabilities {
  process: CapabilityEvidence;
  terminal: CapabilityEvidence;
  attach: CapabilityEvidence;
  structured_events: CapabilityEvidence;
  sessions: CapabilityEvidence;
  resume: CapabilityEvidence;
  fork: CapabilityEvidence;
  submit_prompt: CapabilityEvidence;
  cancel_turn: CapabilityEvidence;
  approvals: CapabilityEvidence;
  native_ui_attach: CapabilityEvidence;
  headless: CapabilityEvidence;
  slash_control: CapabilityEvidence;
  /** Older Core versions may omit this newer capability. */
  read_only_review?: CapabilityEvidence;
  control_level: string;
}

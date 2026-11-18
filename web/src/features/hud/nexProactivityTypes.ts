export type NexProactivityMode = 'manual' | 'contextual' | 'proactive';

export interface NexProactivitySettings {
  mode: NexProactivityMode;
  quiet_start: string;
  quiet_end: string;
  timezone: string;
  min_interval_minutes: number;
}

export interface NexProactivityCandidate {
  fingerprint: string;
  title: string;
  description: string;
  target: 'missions' | 'usage';
  evidence: 'mission_needs_user' | 'hud_alert';
  observed_at: string;
}

export interface NexProactivityState {
  settings: NexProactivitySettings;
  candidates: NexProactivityCandidate[];
}

export interface NexProactivityClaim {
  claimed: boolean;
  suggestion?: NexProactivityCandidate;
}

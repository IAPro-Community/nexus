export type NexAssistantTarget =
  'missions' | 'agents' | 'usage' | 'nexus' | 'settings' | 'new-agent';

export interface NexAssistantSuggestion {
  id: string;
  title: string;
  description: string;
  target: NexAssistantTarget;
  record_id?: string;
}

export interface NexAssistantOverview {
  schema: 'nexus.nex.assistant.v1';
  generated_at: string;
  project_id?: string;
  agent_count: number;
  run_count: number;
  needs_user: number;
  alerts: number;
  suggestions: NexAssistantSuggestion[];
}

export interface NexChatTurn {
  role: 'user' | 'assistant';
  content: string;
}

export interface NexActionProposal extends NexAssistantSuggestion {
  risk: 'navigation';
  requires_confirmation: true;
}

export interface NexChatResponse {
  answer: string;
  source: 'nexus_snapshot' | 'configured_model' | 'stored';
  proposal?: NexActionProposal;
}

export interface NexConversationHistory {
  persisted: boolean;
  turns: Array<NexChatTurn & { sequence: number; source?: string; created_at: string }>;
}

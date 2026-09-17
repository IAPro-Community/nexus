/**
 * Nexus conversation domain types — independent of assistant-ui.
 * Events are the source of truth; ThreadMessageLike is a view projection.
 */

export type NexusConversationEventKind =
  | 'user'
  | 'assistant'
  | 'system'
  | 'status'
  | 'activity'
  | 'tool_call'
  | 'tool_result'
  | 'artifact'
  | 'attachment'
  | 'image'
  | 'approval'
  | 'warning'
  | 'error'
  | 'completion';

export type NexusActivityItemStatus = 'pending' | 'running' | 'done' | 'skipped' | 'error';

export type NexusActivityItem = {
  id: string;
  label: string;
  status: NexusActivityItemStatus;
  detail?: string;
  startedAt?: string;
  finishedAt?: string;
};

export type NexusActivityPayload = {
  title: string;
  currentId?: string;
  items: NexusActivityItem[];
  error?: string;
  expandable?: boolean;
};

export type NexusToolStatus = 'pending' | 'running' | 'success' | 'failure';

export type NexusToolPayload = {
  toolName: string;
  status: NexusToolStatus;
  summary?: string;
  detail?: string;
  args?: Record<string, unknown>;
};

export type NexusAttachmentRef = {
  id: string;
  name: string;
  mimeType: string;
  size: number;
  url?: string;
};

export type NexusConversationEvent = {
  id: string;
  sessionId: string;
  sequence: number;
  kind: NexusConversationEventKind;
  /** Plain text for user/assistant; optional for structured kinds. */
  content?: string;
  payload?: Record<string, unknown>;
  attachmentIds?: string[];
  createdAt: string;
  status?: 'complete' | 'running' | 'incomplete' | 'error';
};

export type NexusAttachment = {
  id: string;
  sessionId: string;
  name: string;
  mimeType: string;
  size: number;
  type: 'image' | 'file';
  /** Server-internal path; no longer exposed by the API (kept for compat). */
  storageRef?: string;
  metadata?: Record<string, unknown>;
  createdAt: string;
  /** Transient preview URL for pending uploads (revoked after submit). */
  previewUrl?: string;
};

export const IMAGE_ATTACHMENT_MIME = new Set(['image/png', 'image/jpeg', 'image/webp']);

export function isAllowedImageMime(mime: string): boolean {
  return IMAGE_ATTACHMENT_MIME.has(mime.toLowerCase());
}

import { describe, expect, it } from 'vitest';
import {
  buildNotificationTarget,
  prepareNotificationEnvelope,
  resolveNotificationTransport,
} from './notificationPolicy';

describe('resolveNotificationTransport', () => {
  it('returns IN_APP when Nexus is visible and focused', () => {
    expect(resolveNotificationTransport({
      hasActiveClients: true,
      visible: true,
      hasFocus: true,
    })).toBe('IN_APP');
  });

  it('returns BROWSER when Nexus is open but hidden', () => {
    expect(resolveNotificationTransport({
      hasActiveClients: true,
      visible: false,
      hasFocus: false,
    })).toBe('BROWSER');
  });

  it('returns WEB_PUSH when there are no active clients', () => {
    expect(resolveNotificationTransport({
      hasActiveClients: false,
      visible: false,
      hasFocus: false,
    })).toBe('WEB_PUSH');
  });

  it('can route background notifications to NATIVE_OS when requested explicitly', () => {
    expect(resolveNotificationTransport({
      hasActiveClients: true,
      visible: false,
      hasFocus: false,
      backgroundTransport: 'NATIVE_OS',
    })).toBe('NATIVE_OS');
  });
});

describe('prepareNotificationEnvelope', () => {
  it('deduplicates by attentionEventId and strips bounded sensitive content', () => {
    const envelope = prepareNotificationEnvelope(
      {
        runtimeId: 'rt-1',
        projectId: 'prj-1',
        projectName: 'Payments',
        reason: 'QUESTION',
        attentionEventId: 'evt-777',
        view: 'work',
        context: 'Bearer sk-live-secret transcript: user pasted credentials password=hunter2',
        summary: `Deploy blocked ${'x'.repeat(400)}`,
        credentials: { token: 'sk-live-secret' },
        transcript: 'full transcript should never escape',
      },
      {
        hasActiveClients: true,
        visible: false,
        hasFocus: false,
      },
    );

    expect(envelope.transport).toBe('BROWSER');
    expect(envelope.dedupeKey).toBe('evt-777');
    expect(envelope.body.length).toBeLessThanOrEqual(240);
    expect(envelope.body.toLowerCase()).not.toContain('transcript');
    expect(envelope.body.toLowerCase()).not.toContain('credentials');
    expect(envelope.body).not.toContain('sk-live-secret');
    expect(envelope.data).toEqual({
      attentionEventId: 'evt-777',
      projectId: 'prj-1',
      reason: 'QUESTION',
      runtimeId: 'rt-1',
      view: 'work',
    });
  });

  it('falls back to a stable legacy dedupe key when no attentionEventId exists yet', () => {
    const envelope = prepareNotificationEnvelope(
      {
        runtimeId: 'rt-legacy',
        projectName: 'Projeto',
        reason: 'APPROVAL',
        context: 'Approve deploy',
      },
      {
        hasActiveClients: true,
        visible: false,
        hasFocus: false,
      },
    );

    expect(envelope.dedupeKey).toBe('rt-legacy:APPROVAL:Approve deploy');
  });
});

describe('buildNotificationTarget', () => {
  it('encodes project and view into the Nexus URL', () => {
    expect(buildNotificationTarget('https://nexus.local/app', {
      projectId: 'prj-42',
      view: 'work',
    })).toBe('https://nexus.local/?project=prj-42&view=work');
  });
});

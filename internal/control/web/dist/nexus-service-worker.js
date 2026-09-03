(function () {
  const DEFAULT_BODY = 'Abra o Nexus para revisar a atenção pendente.';
  const DEFAULT_VIEW = 'overview';
  const MAX_TITLE_LENGTH = 120;
  const MAX_BODY_LENGTH = 240;

  function sanitizeText(value, fallback) {
    if (typeof value !== 'string') return fallback;
    const sanitized = value
      .replace(/\uFFFD+/g, ' ')
      .replace(/[\u0000-\u0008\u000B\u000C\u000E-\u001F\u007F]/g, ' ')
      .replace(/\s+/g, ' ')
      .trim();
    return sanitized || fallback;
  }

  function truncate(text, maxLength) {
    if (text.length <= maxLength) return text;
    return `${text.slice(0, maxLength - 1).trimEnd()}…`;
  }

  function containsSensitiveMarker(text) {
    return /\b(?:transcript|credentials?|password|token|secret|authorization|bearer|api[-_ ]?key)\b/i.test(text);
  }

  function redactInlineSecrets(text) {
    return text
      .replace(/\bBearer\s+[^\s]+/gi, '[redacted]')
      .replace(/\b(?:sk|pk|rk|ghp)_[A-Za-z0-9_-]+\b/g, '[redacted]')
      .replace(/\b(?:password|token|secret|authorization|api[-_ ]?key)\s*[:=]\s*[^\s]+/gi, '[redacted]');
  }

  function notificationTitle(projectName, reason, dynamicTitle) {
    const project = sanitizeText(projectName, 'Projeto');
    const dynamic = sanitizeText(dynamicTitle, '');
    if (dynamic && !containsSensitiveMarker(dynamic)) return truncate(`[Nexus | ${project}] ${dynamic}`, MAX_TITLE_LENGTH);
    if (reason === 'QUESTION') return `[Nexus | ${project}] ❓ Pergunta Requer Atenção`;
    if (reason === 'APPROVAL') return `[Nexus | ${project}] ⚠️ Aprovação Necessária`;
    if (reason === 'TASK_COMPLETED') return `[Nexus | ${project}] ✅ Tarefa Concluída!`;
    return `[Nexus | ${project}] Notificação`;
  }

  function resolveTransport(context) {
    if (!context.hasActiveClients) return 'WEB_PUSH';
    if (context.visible && context.hasFocus) return 'IN_APP';
    return context.backgroundTransport || 'BROWSER';
  }

  function buildBody(payload) {
    const candidates = [payload && payload.context, payload && payload.summary, payload && payload.dynamicTitle];
    for (const candidate of candidates) {
      const text = sanitizeText(candidate, '');
      if (!text) continue;
      if (containsSensitiveMarker(text)) continue;
      const redacted = redactInlineSecrets(text);
      if (containsSensitiveMarker(redacted)) continue;
      return truncate(redacted, MAX_BODY_LENGTH);
    }
    return DEFAULT_BODY;
  }

  function prepareEnvelope(payload, context) {
    const body = buildBody(payload || {});
    const eventId = sanitizeText(payload && payload.attentionEventId, '');
    const dedupeKey = eventId || `${payload && payload.runtimeId}:${payload && payload.reason}:${body}`;
    return {
      transport: resolveTransport(context),
      dedupeKey,
      title: notificationTitle(payload && payload.projectName, payload && payload.reason, payload && payload.dynamicTitle),
      body,
      tag: `nexus-${dedupeKey}`,
      data: {
        attentionEventId: eventId || undefined,
        projectId: sanitizeText(payload && payload.projectId, '') || undefined,
        reason: payload && payload.reason,
        runtimeId: payload && payload.runtimeId,
        view: sanitizeText(payload && payload.view, DEFAULT_VIEW) || DEFAULT_VIEW,
      },
    };
  }

  function buildTarget(origin, data) {
    const url = new URL(origin);
    url.pathname = '/';
    url.search = '';
    const projectId = sanitizeText(data && data.projectId, '');
    const view = sanitizeText(data && data.view, DEFAULT_VIEW) || DEFAULT_VIEW;
    if (projectId) url.searchParams.set('project', projectId);
    url.searchParams.set('view', view);
    return url.toString();
  }

  function isNexusClient(client) {
    if (!client || typeof client.url !== 'string') return false;
    try {
      return new URL(client.url).origin === self.location.origin;
    } catch {
      return false;
    }
  }

  self.addEventListener('push', (event) => {
    event.waitUntil((async () => {
      const payload = event.data && typeof event.data.json === 'function' ? event.data.json() : {};
      const clientList = await self.clients.matchAll({ type: 'window', includeUncontrolled: true });
      const hasActiveClients = clientList.some(isNexusClient);
      const envelope = prepareEnvelope(payload, {
        hasActiveClients,
        visible: false,
        hasFocus: false,
      });

      if (envelope.transport !== 'WEB_PUSH') {
        return;
      }

      await self.registration.showNotification(envelope.title, {
        body: envelope.body,
        data: envelope.data,
        icon: '/logo.png',
        tag: envelope.tag,
      });
    })());
  });

  self.addEventListener('notificationclick', (event) => {
    event.notification.close();
    event.waitUntil((async () => {
      const target = buildTarget(self.location.origin, event.notification && event.notification.data);
      const clientList = await self.clients.matchAll({ type: 'window', includeUncontrolled: true });
      const existingClient = clientList.find(isNexusClient);

      if (existingClient) {
        if (typeof existingClient.navigate === 'function') {
          await existingClient.navigate(target);
        }
        if (typeof existingClient.focus === 'function') {
          await existingClient.focus();
        }
        return;
      }

      await self.clients.openWindow(target);
    })());
  });

  self.addEventListener('pushsubscriptionchange', (event) => {
    event.waitUntil(Promise.resolve().then(() => {
      console.info('[NexusSW] pushsubscriptionchange received');
    }));
  });
})();

#!/usr/bin/env node
// Capacity Monitor API E2E (no browser required).
//
// Exercises the T6 surface against an isolated `nexus web` server:
// unauthenticated denial, CSRF negative, empty-not-healthy status,
// preferences validation + round-trip, ack/snooze lifecycle, history.
// Usage: node scripts/e2e-capacity.mjs (requires ./nexus binary: make build)
import assert from 'node:assert/strict';
import { startFixtureServer, terminateProcess } from './e2e-fixture.mjs';

let cookies = [];
let seenSetCookie = [];
function storeCookies(res) {
  const raw = res.headers.getSetCookie ? res.headers.getSetCookie() : [];
  for (const entry of raw) {
    seenSetCookie.push(entry);
    const pair = entry.split(';', 1)[0];
    const name = pair.split('=', 1)[0];
    cookies = cookies.filter((c) => !c.startsWith(`${name}=`));
    cookies.push(pair);
  }
}
let originHeader = '';
async function api(origin, method, path, { body, csrf } = {}) {
  const headers = { Accept: 'application/json' };
  // Browsers always send Origin: exercise the CheckOrigin path instead of
  // the lenient no-Origin path.
  headers.Origin = originHeader || origin;
  if (cookies.length > 0) headers.Cookie = cookies.join('; ');
  if (csrf) headers['X-CSRF-Token'] = csrf;
  let payload;
  if (body !== undefined) {
    headers['Content-Type'] = 'application/json';
    payload = JSON.stringify(body);
  }
  const res = await fetch(`${origin}${path}`, { method, headers, body: payload });
  storeCookies(res);
  let data = null;
  try {
    data = await res.json();
  } catch {}
  return { status: res.status, data };
}

async function main() {
  const { server, bootstrapUrl } = await startFixtureServer();
  try {
    const origin = new URL(bootstrapUrl).origin;
    originHeader = origin;

    // 1. Unauthenticated capacity endpoints are denied (auth before CSRF).
    {
      const res = await api(origin, 'GET', '/api/v1/capacity/status');
      assert.equal(res.status, 401, 'capacity status without session must be 401');
    }
    {
      const res = await api(origin, 'GET', '/api/v1/capacity/history');
      assert.equal(res.status, 401, 'capacity history without session must be 401');
    }
    {
      const res = await api(origin, 'POST', '/api/v1/capacity/ack', { body: { fingerprint: 'x' } });
      assert.equal(res.status, 401, 'capacity ack without session must be 401');
    }

    // Bootstrap exchange (same as the web client initSession).
    const fragment = new URL(bootstrapUrl).hash.replace(/^#/, '');
    const token = new URLSearchParams(fragment).get('nexus_bootstrap') || '';
    assert.ok(token, 'bootstrap URL must carry nexus_bootstrap fragment');
    {
      const res = await api(origin, 'POST', '/api/v1/auth/bootstrap', { body: { token } });
      assert.equal(res.status, 200, `bootstrap exchange: ${res.status}`);
    }
    const session = await api(origin, 'GET', '/api/v1/session');
    assert.equal(session.status, 200);
    const csrf = session.data?.csrf_token;
    assert.ok(csrf, 'session must issue a CSRF token');
    assert.ok(
      seenSetCookie.some((entry) => /httponly/i.test(entry)),
      'session cookies must be HttpOnly',
    );
    assert.ok(
      seenSetCookie.some((entry) => /samesite=strict/i.test(entry)),
      'session cookies must be SameSite=Strict',
    );
    // 2. CSRF negative: state-changing calls without token are rejected.
    {
      const res = await api(origin, 'POST', '/api/v1/capacity/ack', { body: { fingerprint: 'x' } });
      assert.equal(res.status, 403, 'ack without CSRF must be 403');
    }
    {
      const prefs = (await api(origin, 'GET', '/api/v1/capacity/preferences', { csrf })).data;
      const res = await api(origin, 'PUT', '/api/v1/capacity/preferences', { body: prefs });
      assert.equal(res.status, 403, 'preferences PUT without CSRF must be 403');
    }

    // 3. Fresh status: enabled defaults, empty resources (unknown, never
    // healthy), monitored count consistent.
    {
      const res = await api(origin, 'GET', '/api/v1/capacity/status', { csrf });
      assert.equal(res.status, 200);
      assert.equal(res.data.enabled, true);
      assert.deepEqual(res.data.resources, []);
      assert.equal(res.data.monitored_resources, 0);
      assert.ok(res.data.preferences, 'preferences embedded in status');
    }

    // 4. History starts empty.
    {
      const res = await api(origin, 'GET', '/api/v1/capacity/history?limit=10', { csrf });
      assert.equal(res.status, 200);
      assert.deepEqual(res.data, []);
    }

    // 5. Preferences reject hysteresis inversion, accept valid round-trip
    // with thresholds and background flag verified.
    {
      const current = (await api(origin, 'GET', '/api/v1/capacity/preferences', { csrf })).data;
      const bad = await api(origin, 'PUT', '/api/v1/capacity/preferences', {
        csrf,
        body: {
          ...current,
          warning_threshold: 5,
          critical_threshold: 10,
          exhausted_threshold: 0,
          recovery_threshold: 25,
        },
      });
      assert.equal(bad.status, 400, 'inverted thresholds must be 400');
      const ok = await api(origin, 'PUT', '/api/v1/capacity/preferences', {
        csrf,
        body: {
          ...current,
          warning_threshold: 20,
          critical_threshold: 10,
          exhausted_threshold: 0,
          recovery_threshold: 25,
          reminder_interval_seconds: 1800,
          background_enabled: false,
        },
      });
      assert.equal(ok.status, 200);
      assert.equal(ok.data.reminder_interval_seconds, 1800);
      assert.equal(ok.data.warning_threshold, 20);
      assert.equal(ok.data.critical_threshold, 10);
      assert.equal(ok.data.background_enabled, false);
    }

    // 6. Ack lifecycle echoes the fingerprint.
    {
      const missing = await api(origin, 'POST', '/api/v1/capacity/ack', { csrf, body: {} });
      assert.equal(missing.status, 400, 'ack without fingerprint must be 400');
      const acked = await api(origin, 'POST', '/api/v1/capacity/ack', {
        csrf,
        body: { fingerprint: 'alert-e2e-1' },
      });
      assert.equal(acked.status, 200);
      assert.equal(acked.data.acked, true);
      assert.equal(acked.data.fingerprint, 'alert-e2e-1');
    }

    // 7. Snooze lifecycle: past and malformed rejected, 30-day cap enforced,
    // future recorded, empty clears.
    {
      const past = new Date(Date.now() - 3600_000).toISOString();
      const bad = await api(origin, 'POST', '/api/v1/capacity/snooze', {
        csrf,
        body: { fingerprint: 'alert-e2e-1', until: past },
      });
      assert.equal(bad.status, 400, 'past snooze must be 400');
      const malformed = await api(origin, 'POST', '/api/v1/capacity/snooze', {
        csrf,
        body: { fingerprint: 'alert-e2e-1', until: 'tomorrow-ish' },
      });
      assert.equal(malformed.status, 400, 'malformed until must be 400');
      const tooFar = new Date(Date.now() + 31 * 86400_000).toISOString();
      const capped = await api(origin, 'POST', '/api/v1/capacity/snooze', {
        csrf,
        body: { fingerprint: 'alert-e2e-1', until: tooFar },
      });
      assert.equal(capped.status, 400, 'snooze past 30 days must be 400');
      const future = new Date(Date.now() + 3600_000).toISOString();
      const set = await api(origin, 'POST', '/api/v1/capacity/snooze', {
        csrf,
        body: { fingerprint: 'alert-e2e-1', until: future },
      });
      assert.equal(set.status, 200);
      assert.ok(set.data.snoozed_until, 'future snooze recorded');
      const cleared = await api(origin, 'POST', '/api/v1/capacity/snooze', {
        csrf,
        body: { fingerprint: 'alert-e2e-1', until: '' },
      });
      assert.equal(cleared.status, 200);
      assert.ok(cleared.data.snoozed_until == null, 'cleared snooze reads back null/absent');
    }

    console.log('capacity E2E: PASS (auth, CSRF, status, history, preferences, ack, snooze)');
  } finally {
    await terminateProcess(server);
  }
}

main().catch((error) => {
  console.error(`capacity E2E: FAIL: ${error?.stack || error}`);
  process.exit(1);
});

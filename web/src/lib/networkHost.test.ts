import { describe, expect, it } from 'vitest';
import { isLoopbackHostname, shouldAttachWebSocketQueryToken } from './networkHost';

describe('networkHost', () => {
  it('treats localhost and nexus.local as loopback', () => {
    expect(isLoopbackHostname('localhost')).toBe(true);
    expect(isLoopbackHostname('localhost:13000')).toBe(true);
    expect(isLoopbackHostname('nexus.local')).toBe(true);
    expect(isLoopbackHostname('127.0.0.1:13000')).toBe(true);
  });

  it('rejects private LAN and tunnel hosts', () => {
    expect(isLoopbackHostname('192.168.1.10')).toBe(false);
    expect(isLoopbackHostname('10.0.0.5:13000')).toBe(false);
    expect(isLoopbackHostname('nexus.local.evil.com')).toBe(false);
    expect(isLoopbackHostname('127.0.0.1.evil.com')).toBe(false);
    expect(shouldAttachWebSocketQueryToken('192.168.1.10:13000')).toBe(false);
    expect(shouldAttachWebSocketQueryToken('abc.trycloudflare.com')).toBe(false);
  });

  it('handles bracketed and bare IPv6 loopback', () => {
    expect(isLoopbackHostname('[::1]')).toBe(true);
    expect(isLoopbackHostname('[::1]:13000')).toBe(true);
    expect(isLoopbackHostname('::1')).toBe(true);
    expect(isLoopbackHostname('[::2]')).toBe(false);
    expect(shouldAttachWebSocketQueryToken('[::1]:13000')).toBe(true);
  });

  it('allows query tokens only on loopback', () => {
    expect(shouldAttachWebSocketQueryToken('127.0.0.1:13000')).toBe(true);
    expect(shouldAttachWebSocketQueryToken('localhost')).toBe(true);
  });
});

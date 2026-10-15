import { afterEach, describe, expect, it, vi } from 'vitest';
import { downloadAuthenticatedCsv } from './downloadAuthenticatedCsv';

const auth = vi.hoisted(() => ({ token: '' }));
vi.mock('../api', () => ({ getDesktopAuthToken: () => auth.token }));

afterEach(() => {
  auth.token = '';
  vi.unstubAllGlobals();
});

describe('downloadAuthenticatedCsv', () => {
  it('rejects external and non-API paths before network access', async () => {
    const fetcher = vi.fn();
    vi.stubGlobal('fetch', fetcher);
    await expect(
      downloadAuthenticatedCsv('https://attacker.example/collect', 'tokens.csv'),
    ).rejects.toThrow('Invalid CSV endpoint');
    await expect(downloadAuthenticatedCsv('/other/endpoint', 'tokens.csv')).rejects.toThrow(
      'Invalid CSV endpoint',
    );
    expect(fetcher).not.toHaveBeenCalled();
  });

  it('uses desktop bearer authentication and produces a download', async () => {
    auth.token = 'desktop-session';
    const click = vi.fn();
    const remove = vi.fn();
    const anchor = { href: '', download: '', click, remove };
    const appendChild = vi.fn();
    const createElement = vi.fn(() => anchor);
    const createObjectURL = vi.fn(() => 'blob:local-csv');
    const revokeObjectURL = vi.fn();
    const setTimeout = vi.fn();
    const fetcher = vi.fn(async () => new Response('provider,profile\\n', { status: 200 }));

    vi.stubGlobal('fetch', fetcher);
    vi.stubGlobal('document', { createElement, body: { appendChild } });
    vi.stubGlobal('window', { setTimeout });
    vi.stubGlobal('URL', { createObjectURL, revokeObjectURL });

    await downloadAuthenticatedCsv('/api/v1/capacity/pace.csv', 'nexus-capacity.csv');
    expect(fetcher).toHaveBeenCalledTimes(1);
    const [path, options] = fetcher.mock.calls[0] as [string, RequestInit];
    expect(path).toBe('/api/v1/capacity/pace.csv');
    expect(options.method).toBe('GET');
    expect(options.credentials).toBe('same-origin');
    const headers = options.headers as Headers;
    expect(headers.get('Authorization')).toBe('Bearer desktop-session');
    expect(headers.get('Accept')).toBe('text/csv');
    expect(anchor.download).toBe('nexus-capacity.csv');
    expect(anchor.href).toBe('blob:local-csv');
    expect(click).toHaveBeenCalledOnce();
    expect(remove).toHaveBeenCalledOnce();
    expect(setTimeout).toHaveBeenCalledOnce();
  });

  it('does not manufacture a download after an HTTP authorization failure', async () => {
    auth.token = '';
    const fetcher = vi.fn(async () => new Response('', { status: 401 }));
    const createElement = vi.fn();
    vi.stubGlobal('fetch', fetcher);
    vi.stubGlobal('document', { createElement });

    await expect(
      downloadAuthenticatedCsv('/api/v1/cost/observations.csv', 'costs.csv'),
    ).rejects.toThrow('HTTP 401');
    expect(createElement).not.toHaveBeenCalled();
    const options = fetcher.mock.calls[0]?.[1] as RequestInit | undefined;
    expect((options?.headers as Headers).has('Authorization')).toBe(false);
  });
});

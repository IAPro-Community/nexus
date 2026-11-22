/**
 * Formats an 8-character remote access code for display, grouping it as
 * ABCD-EFGH. Input is trimmed and uppercased; unexpected lengths are returned
 * unchanged so the user always sees exactly what the server minted.
 */
export function formatAccessCode(code: string): string {
  const clean = code.trim().toUpperCase();
  if (clean.length !== 8) return clean;
  return `${clean.slice(0, 4)}-${clean.slice(4)}`;
}

/**
 * Describes the stability of the tunnel's public origin.
 *
 * A Quick Tunnel mints a random hostname on every start. An installed PWA,
 * its service worker cache and its host-bound session cookie all point at
 * that hostname, so a restart silently breaks the phone. A Named Tunnel
 * always serves the same origin, which is what keeps the installed app
 * working across restarts.
 */
export type TunnelOriginKind = 'stable' | 'ephemeral';

export function tunnelOriginKind(stableUrl: boolean | undefined): TunnelOriginKind {
  return stableUrl === true ? 'stable' : 'ephemeral';
}

/**
 * Describes how the operator reaches the Nexus from another device.
 *
 * - `tunnel`: the Cloudflare transport, public and reachable from anywhere.
 * - `private`: a published private origin (tailnet or private DNS). It needs
 *   no Cloudflare tunnel, it is not exposed publicly, and it is stable.
 * - `none`: nothing is published yet.
 *
 * The tunnel wins when both exist: the server issues the invitation for the
 * live tunnel URL while a tunnel runs, so the address shown here must be the
 * one the invitation is actually bound to.
 */
export type RemoteAccessMode = 'none' | 'tunnel' | 'private';

export function remoteAccessMode(
  active: boolean | undefined,
  remoteOrigin: string | undefined,
): RemoteAccessMode {
  if (active) return 'tunnel';
  if (remoteOrigin && remoteOrigin.trim() !== '') return 'private';
  return 'none';
}

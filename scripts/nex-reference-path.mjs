import * as path from 'node:path';

// The NEX reference pack uses a fixed, trusted destination. Never accept
// arbitrary locations from a manifest, including absolute paths, '..' or
// Windows separators. Resolve containment with the *host platform's* APIs.
export function resolveNexReferenceAsset(root, asset, pathApi = path) {
  if (!asset || typeof asset.filename !== 'string' || typeof asset.path !== 'string') {
    return null;
  }
  if (!/^[a-zA-Z0-9][a-zA-Z0-9._-]*\.png$/.test(asset.filename)) return null;
  const expected = `docs/design/nex/reference-pack/2026-10-28/${asset.filename}`;
  if (asset.path !== expected || asset.path.includes('..') || asset.path.includes('\\')) {
    return null;
  }
  const absoluteRoot = pathApi.resolve(root);
  const candidate = pathApi.resolve(absoluteRoot, asset.path);
  const relativePath = pathApi.relative(absoluteRoot, candidate);
  if (
    !relativePath ||
    relativePath === '..' ||
    relativePath.startsWith('..' + pathApi.sep) ||
    pathApi.isAbsolute(relativePath)
  ) {
    return null;
  }
  return candidate;
}

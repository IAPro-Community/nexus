import { createHash } from 'node:crypto';

export function serviceWorkerBuildId(assets) {
  const fingerprint = [...assets]
    .toSorted((left, right) => left.name.localeCompare(right.name))
    .map(({ name, content }) => `${name}:${createHash('sha256').update(content).digest('hex')}`)
    .join('\n');
  return createHash('sha256').update(fingerprint).digest('hex').slice(0, 16);
}

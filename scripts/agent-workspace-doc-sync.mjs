// Keep the human-readable v1 acceptance table consistent with the canonical
// source-status/issue inventory. This is a documentation gate, not release QA.
export function verifyAgentWorkspaceAcceptanceDoc(features, markdown) {
  const errors = [];
  const expected = new Map(features.map((f) => [f.id, f]));
  const seen = new Set();
  let rowCount = 0;
  for (const line of markdown.split(/\r?\n/)) {
    const match = /^\s*\|\s*`([a-z][a-z0-9-]*)`\s*\|/.exec(line);
    if (!match) continue;
    rowCount++;
    const columns = line.split('|').slice(1, -1).map((cell) => cell.trim());
    const id = match[1];
    if (columns.length !== 5) {
      errors.push(`${id}: expected five acceptance-table columns, got ${columns.length}`);
      continue;
    }
    if (seen.has(id)) errors.push(`${id}: duplicate acceptance-table row`);
    seen.add(id);
    const feature = expected.get(id);
    if (!feature) {
      errors.push(`${id}: undocumented feature id in acceptance table`);
      continue;
    }
    const status = columns[2];
    if (status !== feature.sourceStatus) {
      errors.push(`${id}: sourceStatus drift (document=${status}, manifest=${feature.sourceStatus})`);
    }
    const issue = /^\[#(\d+)\]\(https:\/\/github\.com\/IAPro-Community\/nexus\/issues\/(\d+)\)$/.exec(columns[3]);
    if (!issue || issue[1] !== issue[2] || Number(issue[1]) !== feature.issue) {
      errors.push(`${id}: issue link differs from manifest #${feature.issue}`);
    }
  }
  if (rowCount !== features.length) errors.push(`Acceptance table has ${rowCount} rows; manifest has ${features.length}`);
  for (const feature of features) if (!seen.has(feature.id)) errors.push(`${feature.id}: missing from acceptance table`);
  return errors;
}

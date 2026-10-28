import test from 'node:test';
import assert from 'node:assert/strict';
import { posix, win32 } from 'node:path';
import { resolveNexReferenceAsset } from '../nex-reference-path.mjs';

const name = 'nex-visual-audit-2026-10-28.png';
const valid = {filename: name, path: `docs/design/nex/reference-pack/2026-10-28/${name}`};

for (const [label, pathApi, root] of [
  ['Linux', posix, '/home/runner/work/nexus'],
  ['Windows', win32, 'C:\\a\\nexus'],
]) {
  test(`${label}: known missing-but-registered NEX PNG is a safe path`, () => {
    assert.equal(resolveNexReferenceAsset(root, valid, pathApi), pathApi.resolve(root, valid.path));
  });
  test(`${label}: reject traversal and mismatched source/filename`, () => {
    for (const asset of [
      {...valid, path:'../secret.png'},
      {...valid, path:'/etc/passwd'},
      {...valid, path:'C:\\Windows\\system.ini'},
      {...valid, path:`docs/design/nex/reference-pack/2026-10-28/../${name}`},
      {...valid, path:`docs/design/nex/reference-pack/2026-10-28\\${name}`},
      {...valid, filename:'../../secret.png'},
      {...valid, filename:'.hidden.png'},
      {...valid, filename:'nex.svg'},
      {...valid, filename:'other.png'},
      {...valid, filename:''},
      {filename:name},
    ]) {
      assert.equal(resolveNexReferenceAsset(root, asset, pathApi), null, JSON.stringify(asset));
    }
  });
}

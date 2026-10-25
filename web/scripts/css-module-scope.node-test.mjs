import assert from 'node:assert/strict';
import test from 'node:test';
import { scopeCssModule } from './css-module-scope.mjs';

test('scopes CSS classes without rewriting asset URLs or declaration values', () => {
  const source = `.sprite, .root[data-mode='working'] .sprite:hover {
  background-image: url('/images/workspace-grid.svg');
  opacity: 0.5;
  content: '.literal';
}`;
  const result = scopeCssModule(source, 'src/components/Example.module.scss');

  assert.match(result.css, /\.sprite_[a-f0-9]{6}/);
  assert.match(result.css, /\.root_[a-f0-9]{6}\[data-mode='working'\]/);
  assert.match(result.css, /url\('\/images\/workspace-grid\.svg'\)/);
  assert.match(result.css, /opacity: 0\.5;/);
  assert.match(result.css, /content: '\.literal';/);
  assert.equal(Object.hasOwn(result.classMap, 'svg'), false);
  assert.equal(Object.hasOwn(result.classMap, 'literal'), false);
});

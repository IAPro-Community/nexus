import { readFile } from 'node:fs/promises';
import { execFileSync } from 'node:child_process';
import prettier from 'prettier';

for (const path of [
  'src/features/hud/NexAssistantPanel.tsx',
  'src/features/hud/NexAssistantActions.tsx',
  'src/features/hud/NexAssistantPanel.test.tsx',
  'src/nexus/api.ts',
  'src/features/hud/NexusHud.tsx',
]) {
  const original = await readFile(path, 'utf8');
  const formatted = await prettier.format(original, {
    ...(await prettier.resolveConfig(path)),
    filepath: path,
  });
  console.log('NEX_FORMAT:' + path + ':' + Buffer.from(formatted).toString('base64'));
}
for (const path of [
  'internal/control/web/handlers_nex_assistant.go',
  'internal/control/web/handlers_nex_assistant_test.go',
]) {
  const formatted = execFileSync('gofmt', ['../' + path]);
  console.log('NEX_GOFORMAT:' + path + ':' + formatted.toString('base64'));
}

import { readFile } from 'node:fs/promises';
import { execFileSync } from 'node:child_process';
import prettier from 'prettier';

for (const path of [
  'src/features/hud/NexAssistantPanel.tsx',
  'src/features/hud/nexAssistantTypes.ts',
]) {
  const content = await prettier.format(await readFile(path, 'utf8'), {
    ...(await prettier.resolveConfig(path)),
    filepath: path,
  });
  console.log('NEX_FORMAT:' + path + ':' + Buffer.from(content).toString('base64'));
}
for (const path of [
  'internal/control/web/nex_action_intents.go',
  'internal/control/web/nex_action_intents_test.go',
  'internal/control/web/handlers_nex_assistant.go',
]) {
  console.log('NEX_GOFORMAT:' + path + ':' +
    execFileSync('gofmt', ['../' + path]).toString('base64'));
}

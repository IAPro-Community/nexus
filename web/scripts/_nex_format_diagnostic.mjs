import { readFile } from 'node:fs/promises';
import prettier from 'prettier';
import { execFileSync } from 'node:child_process';

const files = [
  'src/features/hud/NexAssistantPanel.tsx',
  'src/features/hud/nexAssistantPreferences.ts',
  'src/features/hud/nexRigProjection.test.ts',
  'src/features/hud/nexRigProjection.ts',
  'src/features/hud/NexAssistantActions.tsx',
  'src/features/hud/NexAssistantPanel.module.scss',
];
for (const path of files) {
  const original = await readFile(path, 'utf8');
  const formatted = await prettier.format(original, {
    ...(await prettier.resolveConfig(path)),
    filepath: path,
  });
  console.log('NEX_FORMAT:' + path + ':' + Buffer.from(formatted).toString('base64'));
}

const goFiles = [
  'internal/nexus/intelligence/nex_chat.go',
  'internal/control/web/handlers_nex_assistant.go',
  'internal/control/web/handlers_nex_assistant_test.go',
  'internal/nexus/intelligence/nex_chat_test.go',
];
for (const path of goFiles) {
  const formatted = execFileSync('gofmt', ['../' + path]);
  console.log('NEX_GOFORMAT:' + path + ':' + formatted.toString('base64'));
}

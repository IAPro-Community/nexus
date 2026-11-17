import { readFile } from 'node:fs/promises';
import prettier from 'prettier';

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

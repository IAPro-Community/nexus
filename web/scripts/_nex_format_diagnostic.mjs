import { readFile } from 'node:fs/promises';
import prettier from 'prettier';
for (const path of [
  'src/features/hud/NexAssistantPanel.tsx',
  'src/features/hud/NexAssistantActions.tsx',
]) {
  const input = await readFile(path, 'utf8');
  const output = await prettier.format(input, {
    ...(await prettier.resolveConfig(path)),
    filepath: path,
  });
  console.log('NEX_FORMAT:' + path + ':' + Buffer.from(output).toString('base64'));
}

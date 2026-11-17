import { readFile } from 'node:fs/promises';
import { execFileSync } from 'node:child_process';
import prettier from 'prettier';
for (const path of ['src/features/hud/NexAssistantPanel.tsx']) {
  const output = await prettier.format(await readFile(path, 'utf8'), {
    ...(await prettier.resolveConfig(path)),
    filepath: path,
  });
  console.log('NEX_FORMAT:' + path + ':' + Buffer.from(output).toString('base64'));
}
for (const path of [
  'internal/nexus/store/nex_conversation.go',
  'internal/nexus/store/nex_conversation_test.go',
  'internal/control/web/handlers_nex_assistant.go',
  'internal/control/web/handlers_nex_conversation.go',
  'internal/control/web/handlers_nex_conversation_test.go',
]) {
  const output = execFileSync('gofmt', ['../' + path]);
  console.log('NEX_GOFORMAT:' + path + ':' + output.toString('base64'));
}

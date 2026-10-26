#!/usr/bin/env node
// Deterministic OpenAI-compatible Intelligence fixture. Never uses real tokens,
// network providers or the user's local configuration.
import { createServer } from 'node:http';

export async function startComposerIntelligenceMock() {
  const requests = [];
  const server = createServer(async (req, res) => {
    if (req.method !== 'POST' || req.url !== '/v1/chat/completions') {
      res.writeHead(404).end();
      return;
    }
    let body = '';
    for await (const part of req) {
      body += part.toString();
      if (body.length > 262144) {
        res.writeHead(413).end();
        return;
      }
    }
    let parsed;
    try { parsed = JSON.parse(body); } catch {
      res.writeHead(400).end();
      return;
    }
    const system = String(parsed.messages?.[0]?.content || '').toLowerCase();
    const user = String(parsed.messages?.[1]?.content || '');
    requests.push({ system: system.slice(0, 100), model: parsed.model });
    const plan = {
      discovery_complete: true,
      clarifications: [],
      recommended_approaches: [
        'Iniciar com uma solução incremental e preservar comportamento existente; verificar hipóteses antes de alterar contratos.',
      ],
      in_scope: user.includes('frontend') || user.includes('Frontend')
        ? ['Avaliar estrutura e componentização do frontend']
        : ['Organizar uma primeira entrega verificável'],
      out_of_scope: [],
      constraints_technical: [],
      constraints_security: [],
      risks: [],
      acceptance_criteria: ['A alteração atende ao objetivo e é verificável'],
      testing: ['Executar os testes existentes'],
      verification: [],
      prohibited_actions: [],
      deliverables: ['Plano revisável'],
      stack: [],
      existing_state: [],
    };
    const content = system.includes('extract') || system.includes('structured engineering brief') ||
      system.includes('product discovery and solution planning consultant')
      ? plan
      : system.includes('unknowns') || system.includes('ambiguities')
        ? { unknowns: [] }
        : system.includes('workpackage') || system.includes('workpackages')
          ? { packages: [] }
          : {
              intent: 'Atingir o objetivo informado sem ampliar o escopo arbitrariamente',
              scope: 'project', risk_level: 'medium',
              identified_goals: ['Resultado verificável'], constraints: [], assumptions: [],
            };
    res.writeHead(200, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify({
      id: 'chatcmpl-nexus-fixture',
      object: 'chat.completion',
      choices: [{ index: 0, message: { role: 'assistant', content: JSON.stringify(content) }, finish_reason: 'stop' }],
    }));
  });
  await new Promise((resolve, reject) => {
    server.once('error', reject);
    server.listen(0, '127.0.0.1', resolve);
  });
  const address = server.address();
  if (!address || typeof address === 'string') throw new Error('fixture listener unavailable');
  return {
    baseUrl: `http://127.0.0.1:${address.port}/v1`,
    requests,
    close: () => new Promise((resolve, reject) => server.close((error) => error ? reject(error) : resolve())),
  };
}

export async function configureComposerIntelligence(context, apiOrigin, csrfToken, projectId, mock) {
  const response = await context.request.put(
    `${apiOrigin}/api/v1/intelligence?project_id=${encodeURIComponent(projectId)}`,
    {
      headers: {
        'Accept': 'application/json',
        'Content-Type': 'application/json',
        'X-CSRF-Token': csrfToken,
      },
      data: {
        mode: 'OPENAI_COMPATIBLE',
        model: 'nexus-e2e-model',
        base_url: mock.baseUrl,
      },
    },
  );
  if (!response.ok()) {
    throw new Error(`Unable to configure isolated Intelligence mock: HTTP ${response.status()} ${await response.text()}`);
  }
  return response.json();
}

# Certificação offline assinada — Nexus release/v1.0.0

**Estado: NO_GO até prova nativa.** Este documento define como validar evidências externas sem exigir uma execução paga do GitHub Actions. **Não é um resultado de homologação**. O modo antigo via `--ci-run-id` permanece compatível.

## 1. Pré-condições inegociáveis

- Identificar os SHAs imutáveis de **Nexus e Maestro**. Checkout Nexus deve estar exatamente no SHA candidato, e não apenas no nome `release/v1.0.0`.
- Executar os 48 critérios do contrato `docs/reference/agent-workspace-v1.json` em **dispositivos físicos** Linux amd64, Windows amd64 e macOS arm64. CI, VMs e testes simulados não atestam comportamento nativo quando o critério pede dispositivo real.
- Guardar screenshots PNG, logs redigidos, versão de SO, hash SHA-256 do binário executado, identificação não reversível do dispositivo e nomes dos testes executados em diretório **fora do Git checkout**.
- Confirmar duas sessões autenticadas nos itens multiagent/remote, com resposta e cancelamento idempotentes, cross-project denied, restart, lease e divergência de versão como evidência negativa.
- Um responsável autorizado assina o pacote após revisar artefatos. **Não usar os testes unitários sintéticos como evidência de dispositivo.**

## 2. Âncora de confiança

O arquivo versionado `docs/reference/release-reviewers.json` lista **somente as chaves públicas Ed25519 de revisores autorizados**. Ele está vazio por padrão, fazendo o modo offline falhar fechado até o proprietário aprovar e adicionar uma chave pública por revisão de código. A chave privada deve permanecer fora do repositório e do CI.

Exemplo do formato após autorização real (substituir por chave pública legítima; NÃO adicionar o exemplo ao repositório):

```json
{"schemaVersion":1,"reviewers":[{"id":"release-owner","publicKeyPem":"-----BEGIN PUBLIC KEY-----\n...\n-----END PUBLIC KEY-----\n"}]}
```

## 3. Pacote de evidência externo

Arquivo JSON de metadados limitado a 2 MiB, `schemaVersion: 2`, com:

- `sourceSha` (Nexus), `maestroSha`, `issuedAtUtc` (UTC);
- `platforms`: **exatamente três** entradas `linux-amd64`, `windows-amd64`, `macos-arm64`; cada uma contém `sourceSha`, `maestroSha`, `binarySha256`, `deviceIdHash`, `osVersion`, `runnerType: "physical"`, `status: "passed"`, `reviewerId`, `reviewedAtUtc` e `artifacts` de screenshot PNG e log com SHA-256;
- `features`: as 48 IDs exatas, uma por item, cada uma com `status: "passed"`, `sourceSha`, `tests`, `platforms` e `clients >= 2` onde aplicável;
- `signature`: `{ "keyId": "...", "base64": "..." }` com assinatura Ed25519 da serialização JSON canônica do **payload sem assinatura**, com chaves de objeto ordenadas. O módulo `scripts/offline-release-evidence.mjs` exporta `canonicalEvidencePayload` para essa finalidade.

Todos os caminhos de `artifacts[].path` são relativos ao diretório externo do pacote e não podem escapar desse diretório; hashes de arquivos são verificados contra o conteúdo presente. Os screenshots precisam ser PNG com dimensões úteis. A assinatura sozinha **não prova** que um equipamento físico foi utilizado: o responsável precisa validar a origem dos artefatos.

## 4. Executar

```bash
git rev-parse HEAD
node --test scripts/verify-agent-workspace-v1.test.mjs
node scripts/verify-agent-workspace-v1.mjs --release --offline \
  --sha <SHA_COMPLETO_NEXUS> \
  --maestro-sha <SHA_COMPLETO_MAESTRO> \
  --evidence /caminho-externo/evidencia-v1/bundle.json
```

Para que o verificador libere GO, as 48 features do contrato também devem estar marcadas `sourceStatus: "present"` e `verification: "verified"`, **somente depois** da respectiva homologação. O manifest visual `docs/assets/screenshots/manifest.json` tem seu próprio gate e deve ser atualizado apenas após novas capturas reais, jamais pela troca isolada de SHA.

## 5. Limites

- O verificador não acessa dispositivos remotos: ele **valida artefatos e atestação** de testes realmente executados por operadores autorizados.
- Nenhum pacote foi entregue ou assinado nesta alteração, portanto o veredito permanece **NO_GO**.
- A verificação antiga com `--release --sha ... --ci-run-id ... --evidence ...` continua disponível.
- Este modo não habilita `runs.submit`, não amplia permissões de providers e não remove o requisito de executar E2E de dois clientes nem da homologação gráfica nativa.

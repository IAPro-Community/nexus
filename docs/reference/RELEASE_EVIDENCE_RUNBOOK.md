# Runbook — Evidência offline de release (caminho sem CI)

Gate fail-closed equivalente ao caminho CI, sem GitHub Actions. Produz o bundle
assinado exigido por `verify-agent-workspace-v1.mjs --release --offline`.

## Pré-requisitos

- Chave Ed25519 do revisor autorizado enrolada em
  `docs/reference/release-reviewers.json` (privada em
  `~/.local/share/nexus-release-keys/ed25519-private.pem`, 0600, **nunca
  versionada**; o owner deve rotacionar para chave pessoal antes do GO final).
- SHA de 40 chars do Nexus (`git rev-parse HEAD` no checkout exato) e do Maestro.
- As 48 features do contrato com `sourceStatus=present` e
  `verification=verified` — só depois da homologação real.

## 1. Certificar as três plataformas físicas

Para cada `linux-amd64`, `windows-amd64`, `macos-arm64`, no MESMO SHA:

1. Build/instalar o artefato distribuído; anotar `sha256` do binário.
2. Rodar a suíte de homologação da plataforma; salvar o log (`test-log`).
3. Capturar screenshot real ≥320×200 do app rodando (PNG).
4. Anotar versão do SO e hash do dispositivo (`deviceIdHash`, sha256 de um
   identificador estável — nunca o identificador cru).

## 2. Montar o payload (`payload.json`)

```jsonc
{
  "schemaVersion": 2,
  "sourceSha": "<40 chars>",
  "maestroSha": "<40 chars>",
  "issuedAtUtc": "2026-11-10T12:00:00Z",
  "platforms": [
    // exatamente 3, status "passed", runnerType "physical"
    {
      "platform": "linux-amd64",
      "status": "passed",
      "runnerType": "physical",
      "sourceSha": "<mesmo SHA>",
      "maestroSha": "<mesmo SHA>",
      "binarySha256": "<64 hex>",
      "deviceIdHash": "<64 hex>",
      "osVersion": "Ubuntu 24.04",
      "reviewedAtUtc": "2026-11-10T12:00:00Z",
      "reviewerId": "maestro-owner",
      "artifacts": [
        // mínimo: 1 binary + 1 screenshot + 1 test-log
        { "kind": "binary", "path": "linux-amd64/nexus", "sha256": "<64 hex>" },
        {
          "kind": "screenshot",
          "path": "linux-amd64/screen.png",
          "sha256": "<64 hex>",
        },
        {
          "kind": "test-log",
          "path": "linux-amd64/test.log",
          "sha256": "<64 hex>",
        },
      ],
    },
    // windows-amd64 e macos-arm64 no mesmo formato
  ],
  "features": [
    // exatamente 48, status "passed", 3 plataformas, testes não vazio
    {
      "id": "terminal-studio",
      "status": "passed",
      "sourceSha": "<SHA>",
      "tests": ["TestX", "e2e/y.ts"],
      "platforms": ["linux-amd64", "windows-amd64", "macos-arm64"],
    },
    // features multiagent/remote/two-client exigem "clients": 2
  ],
}
```

Hash de cada artefato: `node scripts/sign-offline-release-evidence.mjs --hash <arquivo>`.

## 3. Assinar e verificar

```bash
node scripts/sign-offline-release-evidence.mjs \
  --payload payload.json \
  --key ~/.local/share/nexus-release-keys/ed25519-private.pem \
  --key-id maestro-owner \
  --out nexus-release-evidence.json

node scripts/verify-agent-workspace-v1.mjs --release --offline \
  --sha <sourceSha> --maestro-sha <maestroSha> \
  --evidence nexus-release-evidence.json
```

Regras que o verificador impõe (fail-closed, sem bypass):

- Bundle **fora** do checkout do código-fonte.
- Screenshots PNG reais ≥320×200; binários/test-logs com SHA-256 conferidos.
- Timestamps UTC com no máximo 90 dias.
- Assinatura Ed25519 do `keyId` enrolado na trust store; payload canônico.
- Contrato com 48/48 `present` + `verified` antes da verificação.

## 4. Falhas comuns

| Erro                                            | Causa                                                            |
| ----------------------------------------------- | ---------------------------------------------------------------- |
| `No owner-authorized reviewers registered`      | Trust store sem chave — enrolar primeiro.                        |
| `Offline evidence attestation signature failed` | Payload alterado após assinar, ou chave não confere com `keyId`. |
| `Missing physical-device evidence`              | Plataforma faltando, `runnerType != physical` ou SHA divergente. |
| `Screenshot artifact must be a nontrivial PNG`  | Imagem sintética/pequena — usar captura real.                    |
| `release scope not certified`                   | Contrato ainda com `pending`/`partial` — homologar antes.        |

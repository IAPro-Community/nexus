# GO Ceremony Kits — sessões do maestro (execução mínima)

Kits prontos para as 5 sessões externas do GO. **Pré-requisito comum:** code
freeze no SHA final (Lane A concluída); todos os comandos nesse checkout exato.
Referência completa: `RELEASE_EVIDENCE_RUNBOOK.md`. Chaves em
`~/.local/share/nexus-release-keys/` (nunca versionadas).

---

## Kit 1 — AGY 2/2 (≈15 min)

Objetivo: 2/2 contas com quota 5h elegível, evidência medida.

```bash
nexus usage --refresh --json > DEV/validation/current/agy-quota-final.json
# conferir: as DUAS contas expõem janelas 5h nos dois grupos
python3 -c "import json;d=json.load(open('DEV/validation/current/agy-quota-final.json'));print(json.dumps(d,indent=1)[:2000])"
```

Se a 2ª conta seguir sem evidência 5h: abrir sessão da conta, aguardar janela,
repetir. Rotação por saldo: já implementada (`SelectNextBalancedProfile`,
fail-closed).

## Kit 2 — Chave de produção + pacote assinado (≈30 min)

```bash
# 1. gerar/rotacionar chave pessoal do owner (ed25519, 0600)
mkdir -p ~/.local/share/nexus-release-keys
openssl genpkey -algorithm ed25519 -out ~/.local/share/nexus-release-keys/ed25519-private.pem
openssl pkey -in ~/.local/share/nexus-release-keys/ed25519-private.pem -pubout \
  -out ~/.local/share/nexus-release-keys/ed25519-public.pem

# 2. build de produção com trust root embutida (hex da pubkey)
export NEXUS_UPDATE_PUBLIC_KEY=$(openssl pkey -in ~/.local/share/nexus-release-keys/ed25519-public.pem -pubout -outform DER | xxd -p | tr -d '\n')
make build && make build-desktop-wails && make build-hud-wails

# 3. payload + assinatura (ver runbook §2-3)
node scripts/sign-offline-release-evidence.mjs --payload payload.json \
  --key ~/.local/share/nexus-release-keys/ed25519-private.pem
```

Sem a key exportada, o updater fica fail-closed (correto, mas bloqueia GO).

## Kit 3 — Plataformas físicas Windows/macOS (≈45 min/plataforma)

No MESMO SHA em cada máquina (Windows amd64, macOS arm64; Linux já evidenciado
em `DEV/validation/current/runtime-qa-linux-wayland/`):

1. build/instalar artefato; `sha256sum` do binário
2. suíte de homologação da plataforma → `test.log`
3. screenshot real ≥320×200 do app rodando → `screen.png`
4. `deviceIdHash` = sha256 de identificador estável (nunca o cru)
5. anotar `osVersion`

Artefatos por plataforma: 1 binary + 1 screenshot + 1 test-log (runbook §1).

## Kit 4 — Two-client real, 2 dispositivos (≈60 min)

Pré-requisitos: túnel público + código de pareamento 8 chars (implementado:
`/api/v1/auth/pair`, `AccessCodeGate`, anti-força-bruta) + AGY/codex reais.

1. Dispositivo A: `nexus web` (ou desktop) logado com conta real; criar missão
2. Dispositivo B: parear via código de 8 caracteres pelo túnel público
3. Exercitar: atenção cross-device (pergunta do agent aparece nos 2), lease,
   conflito, handoff, browser compartilhado, reconexão após queda do B
4. Capturar: logs dos dois lados + screenshots (log `test-log`, imagens
   `screenshot` no payload, feature `two-client-real-e2e`, `clients: 2`)

## Kit 5 — Non-stop + aceite 48/48 + owner approval NEX (≈90 min)

1. Non-stop com 2 contas codex reais: missão ponta a ponta sem intervenção,
   registro de cada ciclo (`attention` real, nunca fictício)
2. Aceite das 48 capabilities no SHA final (varredura guiada pelo evidence
   map; critérios pendentes marcados na conversa)
3. **NEX owner approval**: abrir `docs/design/nex/hud-compact-v1/nex-compact-64-92-128-comparison.png`
   + `DEV/validation/current/companion-lab/captures/` e aprovar/ajustar
   (destrava `hud_compact`, `behavior_8` e o `owner_visual_approval` final
   após site integrado)

---

## Pós-sessões (eu executo)

- Recaptura de evidência visual sintética no SHA final (`npm --prefix web run docs:capture`) + `make docs-verify`
- `RELEASE_READINESS.json` 11/11 + payload assinado + `--require-ready` PASS
- `make quality-full` final + WORKLOG/VERIFY/HANDOFF + tag de release

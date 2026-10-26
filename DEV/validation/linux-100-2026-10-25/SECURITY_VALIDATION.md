# Validação de segurança — slice Linux local

## Revalidação das dependências Web — 2026-10-26

- O `bun audit` inicial reportou 23 advisories na árvore `web` (incluindo alertas críticos e altos) associados a dependências travadas, entre elas Vitest 3.2.7/Tinypool. Atualizei o intervalo do Vitest para 4.1.11+; o lockfile resolveu Vitest 4.1.12 e corrigiu as dependências transitivas afetadas.
- **PASS:** `bun install --frozen-lockfile` com Bun 1.3.9 preservou o lockfile; `bun audit --json` retornou zero advisories em `web/` e `docs-site/`. `bun run quality:full` passou com 109 arquivos, 675 testes e build de produção; E2E HUD e Mission Attention passaram em Chromium. Logs: `logs/web-bun-audit-after-fix-2026-10-26.json`, `logs/web-bun-audit-run-2026-10-26.txt`, `logs/web-quality-full-after-audit-fix-2026-10-26.log`, `logs/web-e2e-hud-after-audit-fix-2026-10-26.log` e `logs/web-e2e-mission-attention-after-audit-fix-2026-10-26.log`.
- A tentativa independente com OSV-Scanner v2.7.0 leu 626 dependências do lockfile Web, mas a chamada `api.osv.dev` foi interrompida por reset TCP após retries. Essa consulta é **BLOCKED**; seu JSON vazio não foi contado como ausência de vulnerabilidades. O gate `SEC-DEPS-001` passa apenas pela auditoria Bun da árvore travada; a matriz continua `NO_GO_LINUX` por outros bloqueios e pelo release estável ausente.
- O overlay de fontes desta execução é `751cfc1a160c38d950ac5b9807e1febb9746d03981c861c722e5bd965e1ef012` (1.428 arquivos; `logs/source-provenance-web-dependency-security-fix-2026-10-26.txt`).

## Identidade

- Branch: `release/v1.0.0`
- HEAD e remote consultados: `7a04f1b4cf5950b97b4ea1e156dda31ac513823b`
- Fontes avaliadas: fingerprint `268a9e9a0036e89e39bcff07eca880869534c9c75cc13765384bb3fa3fc30b32`, 1.421 fontes; manifesto em [proveniência](logs/p8-source-provenance.txt).
- Worktree modificada e sem commit congelado.

## Scanner Go

| Item | Resultado |
|---|---|
| Comando | `make security` |
| Configuração efetiva | `GO_TAGS=gtk3`; Go `1.26.10` via `GOTOOLCHAIN`; govulncheck `v1.7.0` baixado pelo alvo Make |
| Resultado observado | `No vulnerabilities found.`; exit code 0 |
| Evidência | [log do Make](logs/p8-security-make-security.txt) |

O resultado cobre somente findings de vulnerabilidades conhecidas detectáveis por govulncheck no código Go/build tags avaliados. Não representa auditoria arquitetural, revisão manual de segurança, pentest, verificação de segredos, nem comprova ausência geral de vulnerabilidades.

## Itens ainda sem aprovação

- P8 permanece parcial: ainda falta a revisão arquitetural independente prevista no plano.
- Assinatura de artefatos, manifesto/chave de produção, download/update interrompido e rollback seguem sem ciclo real, pois não existe release estável `v1.0.0` publicada.
- A homologação deste relatório é Linux AMD64; nenhum resultado aqui aprova Windows ou macOS.

Status deste gate: **PASS** para o scanner local; **NO_GO_LINUX** para a release como um todo.

## Reteste no overlay corrente — 2026-10-25

O comando `make security GO_TAGS=gtk3` foi repetido em `release/v1.0.0` no HEAD `7a04f1b4cf5950b97b4ea1e156dda31ac513823b` e no overlay de fontes `2c23ebb0c7726a200acbde9326c23ad6023d8fad0238c2adfef993677203f2fa` (1.425 fontes). **PASS:** `No vulnerabilities found.`, exit code 0. O Makefile usou govulncheck v1.7.0 via Go 1.26.10 e `GO_TAGS=gtk3`; diretório temporário ficou isolado em `/home`. Log: [scanner no overlay exato](logs/p8-current-overlay-make-security-2026-10-25.log). Isto atualiza o fingerprint da execução, mas não muda o limite do scanner descrito acima: revisão arquitetural, ciclo de release assinado e validação de plataformas fora do Linux continuam pendentes.

## Revisão local adicional — 2026-10-25

- **Go dependencies — PASS:** `make security` com `GO_TAGS=gtk3`, govulncheck v1.7.0 e Go 1.26.10 terminou com `No vulnerabilities found.` no overlay atual. Log: [scanner atual](logs/current-security-govulncheck-2026-10-25.txt).
- **Segredos no overlay — PASS com classificação:** Gitleaks v8.24.2 varreu `cmd`, `internal`, `web/src`, `.github` e manifests. Os 18 candidatos em `internal` foram classificados como fixtures de teste/sentinelas sintéticas, código SQL estático e sequência minificada; os demais escopos tiveram zero detecções. Relatórios redigidos: [resumo seguro](logs/current-gitleaks-review-summary-2026-10-25.txt) e `logs/current-gitleaks-working-tree-2026-10-25-*.json`.
- **Histórico Git — BLOCKED (P1):** a varredura de 1.906 commits encontrou 77 candidatos. Além de fixtures, chaves públicas, bundles gerados e um artefato patch removido, duas versões antigas de `DEV/HANDOFF.md` continham URLs de bootstrap local com token. A árvore atual não contém esses tokens e não há listener em `127.0.0.1:3000`; não há evidência neste checkout para provar expiração/revogação do estado antigo. Nenhum valor foi registrado no relatório. A classificação final depende de confirmação do responsável de que a instância/estado antigo foi encerrado e invalidado. O histórico não foi reescrito. Relatório redigido: [Gitleaks history](logs/current-gitleaks-git-history-2026-10-25.json).
- **Dependências Web — BLOCKED (P2):** `npm audit` não aceita `web/bun.lock` (ENOOLOCK), e `bun pm scan` informa ausência de scanner configurado. O mesmo `npm audit` passou para `docs-site/package-lock.json` com 0 vulnerabilidades em 1.298 dependências. Evidências: `logs/current-web-npm-audit-attempt-2026-10-25.txt`, `logs/current-web-bun-pm-scan-2026-10-25.txt`, `logs/current-docs-site-npm-audit-2026-10-25.json`, `logs/current-docs-site-bun-pm-scan-2026-10-25.txt`.

Esta revisão continua limitada ao Linux local e aos dados disponíveis. O histórico com os dois tokens de bootstrap antigos e a árvore de dependências Web sem scanner mantêm o gate geral em **NO_GO_LINUX**.

## Recheck após regressão do updater — 2026-10-25

O fingerprint corrente de fontes e testes agora é `abad854fffba2532c6f3c04bd4273f45747b80688833dddb24d0df418bdc73a8` (1.426 arquivos), manifesto em `logs/source-provenance-after-update-interruption-test-2026-10-25.txt`. A regressão de download interrompido passou com `-race`; `go test ./internal/update`, `go test -race ./internal/update` e `go vet ./internal/update` passaram. `make security` voltou a concluir com `No vulnerabilities found.` após essa alteração. Gitleaks foi repetido no overlay exato: manteve os mesmos 18 candidatos internos e nenhuma detecção em `cmd`, `web/src`, `.github` ou manifests. Evidências: `logs/current-update-interrupted-download-regression-2026-10-25.txt`, `logs/current-update-tests-2026-10-25.txt`, `logs/current-update-race-2026-10-25.txt`, `logs/current-update-vet-2026-10-25.txt`, `logs/current-security-after-update-test-2026-10-25.txt` e `logs/current-gitleaks-after-update-test-2026-10-25-*.json`.

## Reteste Gitleaks no overlay de importação host — 2026-10-26

O overlay atual é `9a66687715414fc8b42d612fd2b4a531370c7f8c4fbfa86338a81ba9728ddcd1` (1.428 fontes). Comparado ao manifesto `abad854fffba2532c6f3c04bd4273f45747b80688833dddb24d0df418bdc73a8`, as nove alterações de código estão em `web/src` e `web/scripts`. Executei Gitleaks v8.24.2 via `go run` em cache temporário, sem instalar binário global: **PASS**, zero detecções em `web/src` (~2,72 MB) e `web/scripts` (~299 KB). Logs redigidos: [web/src](logs/current-gitleaks-usage-host-import-overlay-2026-10-26.txt) e [scripts](logs/current-gitleaks-usage-host-import-scripts-2026-10-26.txt). O restante do código permanece coberto pelo scan anterior no overlay `abad`; os bloqueios de tokens históricos e advisory scan do `bun.lock` continuam.

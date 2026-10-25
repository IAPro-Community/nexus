# P1 — Intervenções duráveis e retomada

## Identificação

- Branch local: `release/v1.0.0`
- HEAD e remoto `origin/release/v1.0.0`: `bf6adf0c80ac3531d7291844512a14a017ab0002`
- Código executado: HEAD acima mais o overlay local desta homologação; a árvore permanece dirty e não foi commitada.
- Fingerprint do overlay testado nesta rodada: `ddad56697609de1374cdede949e794316e3263357b052f745defb64f11a4be5b` (1.412 fontes; [manifesto](logs/p1-e2e-source-provenance-current.txt)).
- Escopo: regressões do runner, persistência SQLite, handoff do worker e validação da requisição HTTP.

## Resultado

**P1 COMPLETE no overlay identificado acima.** O E2E combinado exercitou requisição HTTP autenticada, persistência em SQLite/outbox, encerramento e reabertura do servidor/store, recovery e estado terminal verificado. A duplicação da requisição não duplicou eventos nem efeitos externos; o contador de execução terminou exatamente em 1. O gate global continua `NO_GO_LINUX` por P2–P9 e outros bloqueios de release.

## Correções validadas

| ID | Área | Procedimento e esperado | Resultado observado | Status |
|---|---|---|---|---|
| P1-REPLAY-01 | Runner / despacho | Gerar bloqueio após coordenação insegura ou sem prova de replay; não oferecer `REPLAN_PACKAGE`, preservar ID do despacho e permitir somente verificação sem executar o provedor outra vez. | O campo durável `ReplayProhibited` é gravado; a ação de verificação mantém ID/runtime; tentativa forjada de replanejamento é recusada. Contagem de execução permanece 1. | PASS |
| P1-VERIFY-02 | Runner / verificação | Falhar verificação de despacho com replay proibido; manter bloqueio acionável e identidade, sem nova execução externa. | A Mission volta a `BLOCKED_NEEDS_USER`, oferece apenas nova verificação sem replay e conserva a identidade do despacho. | PASS |
| P1-RESUME-03 | Runner / replanejamento | Resolver retry/replan de falha pré-dispatch; passar novamente por Allocate e Compile antes de Execute, removendo artefato de prompt antigo. | O teste observa Allocate=1, Compile=1, Execute=1 e estado inicial `ALLOCATING`. | PASS |
| P1-DEPENDENCY-04 | Runner / dependências | Retomar uma etapa com pré-requisito ainda não verificado; não alocar a etapa antes do pré-requisito. | A etapa retomada fica `PENDING`, o pré-requisito progride primeiro e a etapa volta a `ALLOCATING` quando o pré-requisito fica `VERIFIED`. | PASS |
| P1-STORE-05 | SQLite / restart | Fechar e reabrir o store; manter `ReplayProhibited`, despacho concluído e resolução segura. | A marca e o despacho sobrevivem ao reopen; replanejamento não é oferecido e a retomada escolhida segue por `TESTING`. | PASS |
| P1-WORKER-06 | Worker / handoff | Fazer o worker antigo exceder o timeout de troca; depois que ele terminar, iniciar apenas uma geração substituta. | O handoff agendado iniciou a geração nova, que concluiu a Mission. `go test -race` focado passou. | PASS |
| P1-HTTP-07 | HTTP / validação | Enviar JSON malformado, JSON trailing, IDs vazios e versão ausente/zero/negativa; responder `400` antes do serviço de Mission. | Os cinco casos retornaram `400` com mensagem determinística. | PASS |
| P1-E2E-08 | HTTP → SQLite → restart | Resolver pela rota HTTP autenticada, repetir a requisição, reiniciar servidor/store e provar retomada terminal exatamente uma vez usando contador de efeito. | E2E integrado passou 10 execuções; teste com race passou 5. Middleware de sessão/CSRF real, dois eventos outbox idempotentes, estado persistido após reopen, resposta GET autenticada no servidor reiniciado e exatamente um efeito `Execute`. | PASS |

## Comandos e logs

- `go test ./internal/nexus/runner -count=1 -v` — passou; [log](logs/p1-mission-runner-regressions-2026-10-25.log).
- `go test ./internal/nexus -count=1` — passou; [log](logs/p1-nexus-worker-and-persistence-regressions-2026-10-25.log).
- `go test ./internal/control/web -count=1` — passou; [log](logs/p1-web-handler-regressions-2026-10-25.log).
- `go test -race ./internal/nexus/runner -count=1` — passou; [log](logs/p1-runner-race-2026-10-25.log).
- `go vet ./internal/control/web ./internal/nexus/...` — passou; [log](logs/p1-go-vet-2026-10-25.log).
- `go test ./internal/nexus/...` também passou após as alterações de worker e persistência; [log](logs/p1-nexus-subpackages-2026-10-25.log), SHA-256 `1e3ccf2d3174aa403ebe5920e39864e13eecc71a6b95c7e5d9c9325e3441d803`.
- `go test ./internal/control/web -run '^TestResolveInterventionHTTPDecisionSurvivesServerRestartExactlyOnce$' -count=10` — PASS; [log](logs/p1-http-restart-e2e-2026-10-25.log).
- `go test -race ./internal/control/web -run '^TestResolveInterventionHTTPDecisionSurvivesServerRestartExactlyOnce$' -count=5` — PASS; [log](logs/p1-http-restart-e2e-race-2026-10-25.log).
- `go test ./internal/control/web -count=1` e `go test ./internal/nexus/... -count=1` — PASS; [Web](logs/p1-web-integration-final-2026-10-25.log) e [Nexus](logs/p1-nexus-integration-e2e-suite-2026-10-25.log).
- `go vet ./internal/control/web ./internal/nexus/...` e `GOCACHE=/tmp/nexus-p1-gocache-2026-10-25 make lint-go` — PASS; [vet](logs/p1-integration-go-vet-final-2026-10-25.log) e [lint](logs/p1-integration-go-lint-isolated-2026-10-25.log).
- `make build build-desktop-wails build-hud-wails` e `make install-local` — PASS; [build](logs/p1-e2e-native-build-2026-10-25.log), [instalação/restart](logs/p1-e2e-local-install-restart-2026-10-25.log), [hashes](logs/p1-e2e-artifacts-2026-10-25.txt). CLI/Desktop/HUD são ELF x86-64 e `ldd` não reportou dependências ausentes.
- Build do overlay atual: `make web` e `make build build-desktop-wails build-hud-wails` — PASS. CLI, Desktop Wails e HUD Wails são ELF x86-64; `ldd` não encontrou dependências ausentes. CLI reporta `1.0.0-beta.10`, commit `bf6adf0c`, Go 1.25.14. [Log de compilação e hashes](logs/p1-native-build-2026-10-25.log); SHA-256 dos binários, na ordem CLI/Desktop/HUD: `656e3fc9e9a320665f9ed161af25bf37dbb78c19e7200806fb4f483d897c78bf`, `0d0a45427d107dfbade40c299dff92025a081fa442f3f91367a382bed6a59ad4`, `12bc1c8c9e53d0c677f1e6a4371513b3f256ac24b424eeea43be7e4722da7c4e`. Log: SHA-256 `e0986f287a10513baa89c7d0d8146e2ba00f4f5fe409f162dcc6653b9751e7e0`.
- `make install-local` instalou `/home/desenvolvedor/.local/bin/nexus` e reiniciou o servidor Web. `/api/v1/health` respondeu `{"status":"ok","version":"1.0.0-beta.10"}` no `127.0.0.1:36485`. Binário instalado SHA-256 `61e50e83681c6f7dec5c800fefaab9021760dbcd17c6aec29bdc5cbb3198e05f`; [log de instalação/restart](logs/p1-local-install-restart-2026-10-25.log), SHA-256 `77dc8ceac273fc1426c05b97715ca23c454014e6b911bcb7ea91084f59e9b5ed`.

## Risco pendente

A campanha P1 está fechada para o overlay testado. Ela não fecha os gates restantes de release; P2–P9, provedor real para atenção do HUD e outros itens da matriz mantêm o veredito global `NO_GO_LINUX`.

## Mascote

A decisão vigente é um único mascote NEX, muito bem acabado e funcional. `NexusHud → GuardianMascot → NexPixelSprite` é um caminho de renderização. A lógica de atenção/humor alimenta o mesmo NEX; a branch não tem segundo personagem nem modo Companion 2D separado. A regressão focada passou 25/25 e a [captura nativa](screenshots/native-hud-current-tree-mascot-xwayland.png) mostra um único NEX. A apresentação `Mascote` mostra o personagem único e foi preservada.

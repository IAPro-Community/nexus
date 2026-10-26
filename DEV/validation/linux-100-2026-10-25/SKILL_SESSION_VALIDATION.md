# Validação de execução de Skills em sessão Nexus

Data: 2026-10-25  
ID: `SKL-FIXTURE-001`  
Área: Skills / Runtime  
Status do caso de fixture: **PASS**  
Gate de provedor real `BLK-008`: **BLOCKED**

## Identidade e isolamento

- Branch: `release/v1.0.0`
- HEAD local/remoto: `7a04f1b4cf5950b97b4ea1e156dda31ac513823b`
- Fingerprint exato do overlay testado: `d773e38c495781f52cdb3fa208c6a80bd1aae88d5e14bfabee56ac486407e0e1` (1.423 arquivos; worktree sem commit imutável)
- Manifesto: [source provenance](logs/source-provenance-skill-session-fixture-2026-10-25.txt)
- Não foram usadas credenciais, conta autenticada ou rede externa. `HOME`, `NEXUS_REAL_HOME`, `NEXUS_CONFIG_DIR`, `AI_CLI_DATA_DIR`, `XDG_RUNTIME_DIR`, perfil e workspace foram isolados em diretórios temporários.
- O executável `codex` no `PATH` era um fixture shell local que respondeu `--version`, capturou argumentos e encerrou com uma saída determinística. O teste confirma que o runtime resolveu esse binário e que o perfil não recebeu `auth.json`.

## Critério e resultado

**Esperado:** skills explícitas e automáticas devem ser resolvidas no catálogo permitido, seus contratos devem chegar intactos ao próximo prompt numa sessão supervisionada do Nexus, o perfil de provider deve permanecer isolado e o Agent/runtime devem encerrar sem processo ativo.

**Observado:** passaram dois subcasos: seleção explícita de `testing` e seleção automática para revisão de segurança/verificação. `Nexus.executeAgentPrompt` criou um RuntimeGeneration real, iniciou o CLI fixture pelo `launcher`, passou o prompt compilado como um único argumento, incluiu `--sandbox read-only` e `--ask-for-approval never`, recebeu a saída do fixture e persistiu Agent e geração como `STOPPED`. O teste confirmou o `HOME`/`CODEX_HOME` temporário e ausência de credenciais.

- **PASS** — execução direcionada, 2 subcasos: [log](logs/skill-session-fixture-repeat-2026-10-25.log).
- **PASS** — cinco repetições normais (10 sessões): [log](logs/skill-session-fixture-repeat-2026-10-25.log).
- **PASS** — três repetições com `-race` (6 sessões): [log](logs/skill-session-fixture-race-2026-10-25.log).
- **PASS** — suíte completa `internal/nexus`: [log](logs/skill-session-internal-nexus-tests-2026-10-25.log).
- **PASS** — `go vet ./internal/nexus`: [log](logs/skill-session-internal-nexus-vet-2026-10-25.log).
- Código de regressão: [skill_session_fixture_test.go](../../../internal/nexus/skill_session_fixture_test.go).

## Reprodução

```bash
GOCACHE=/tmp/nexus-linux-100-2026-10-25/gocache-skill-fixture \
  go test ./internal/nexus -run '^TestSelectedSkillsReachAnIsolatedProviderSession$' -count=5

GOCACHE=/tmp/nexus-linux-100-2026-10-25/gocache-skill-fixture \
  go test -race ./internal/nexus -run '^TestSelectedSkillsReachAnIsolatedProviderSession$' -count=3

GOCACHE=/tmp/nexus-linux-100-2026-10-25/gocache-skill-fixture \
  go test -count=1 -timeout=20m ./internal/nexus

GOCACHE=/tmp/nexus-linux-100-2026-10-25/gocache-skill-fixture \
  go vet ./internal/nexus
```

## Limite do gate

O fixture verifica seleção, compilação, envio do contrato e ciclo de vida do runtime sem depender de serviço externo. Não verifica que um provedor real autenticado aceita o prompt nem seu comportamento de execução. Por isso, `BLK-008` continua **BLOCKED** até uma sessão real autorizada e segura; esta cobertura não transforma a falta de evidência externa em aprovação.

Na primeira tentativa, o executável fixture terminou antes de o handshake do host supervisionado abrir o endpoint. Mantive-o ativo por um segundo após a resposta; as repetições normal e com race passaram. Foi uma sincronização do fixture, sem alteração de código de produto.

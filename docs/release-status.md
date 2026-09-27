# Status público da release do Nexus v1

Atualizado em **27 de setembro de 2026**.

| Campo | Estado |
| --- | --- |
| Programa | estabilização do Nexus 1.0 beta |
| Versão do source candidato | `1.0.0-beta.4` |
| Última beta pública registrada | `v1.0.0-beta.1` |
| Matriz candidata da v1 | Linux amd64 + Windows amd64 + macOS arm64 |
| Release estável v1 | **ainda não publicada** |
| CI do SHA candidato | indisponível no momento por bloqueio de créditos da conta |
| Veredito atual | **beta.4 implementada no source; homologação/certificação exact-SHA pendentes** |

## Linha canônica de integração — 2026-09-27

A branch `release/v1.0.0-beta` é agora a única linha canônica de integração para os retestes da beta.4.

- PR #25 foi integrado por squash em `8fa37f4ca8051b2abce1ade3a0c83dd5b5be64d7`;
- essa integração contém a remediação Windows `ce52b09d...`, a convergência Linux/Windows, Direct/Mission OFF-by-default e o hardening Codex/ConPTY;
- a lista única de gaps e critérios de homologação está na issue #27;
- o PR #24 (Wails v3) foi retargetado para esta release, porém permanece Draft e não mergeável: deve ser rebaseado/portado seletivamente para não reverter contratos mais novos da beta;
- integração na branch beta não equivale a PASS, tag ou publicação.

Até o Wails v3 ser reconciliado, a linha canônica continua usando a implementação Wails atualmente integrada. A publicação permanece bloqueada pelos gates same-SHA e pelos blockers registrados na issue #27.

## Escopo consolidado da beta.4

A linha que antes estava planejada como 1.1 foi absorvida pela V1 enquanto a
primeira release estável ainda não foi publicada. A beta.4 concentra a conclusão
funcional desse escopo sem criar um segundo runtime, scheduler ou Mission engine.

O candidato inclui:

- Direct/Lead/Mission com os hardenings de admissão e isolamento já incorporados;
- Nexus HUD Web/Desktop;
- Nexus Voice Accessibility;
- OpenCode como provider independente de AGY;
- `ExecutionRole`: Build, Investigate, Review, Verify, Architect e Security;
- Agent Projection OpenCode version-aware;
- custom execution targets: local, Docker/VPN Workspace, Docker Compose, WSL,
  SSH e wrappers supervisionados;
- discovery nativo sem LLM;
- activation probe explícito;
- enforcement probe read-only separado e provider-backed;
- Windows/macOS implementados no source candidato, ainda sujeitos à evidência
  nativa exact-SHA antes de anúncio de suporte.

A `v1.0.0-beta.1` continua sendo evidência histórica da beta pública anterior.
Nenhuma tag antiga é reinterpretada retroativamente.

## Portabilidade dos ExecutionRoles

Na beta.4, `ExecutionRole` é um contrato do Nexus, não do provider:

- Codex, Claude, Cursor, Gemini e AGY continuam usando o mesmo selector e
  compiler por **Nexus overlay**;
- OpenCode é o adapter nativo de referência da beta.4;
- ausência de native adapter nunca impede role switch/overlay;
- ausência de prova nativa nunca aparece como primary/subagent/enforcement
  suportado;
- adapters nativos adicionais são expansão posterior e não bloqueiam o
  fechamento funcional da beta.4.

A Web também inclui:

- selector de role no terminal;
- roles do terminal ativo na Command Palette;
- `Ctrl/Cmd+Shift+A` para focar o selector;
- `Shift+Tab` continua pertencendo ao TUI hospedado e não é capturado pelo Nexus.

## Agent Interop / OpenCode — estado da beta.4

O lifecycle da V1 é:

```text
ExecutionRole
   ↓
Nexus overlay
   ↓
OpenCode projection
   ↓
filesystem/ownership verify
   ↓
native discovery
   ↓
explicit activation probe
   ↓
explicit read-only enforcement probe
   ↓
ENFORCED somente com evidência positiva
```

Regras:

- trocar role não cria `AgentRevision`;
- projection automática na UI não chama modelo;
- activation/enforcement exigem confirmação porque podem consumir quota/custo;
- `ENFORCED` exige:
  - receipt corrente;
  - projection verificada;
  - activation nativa verificada;
  - permission efetiva comprovada pelo próprio OpenCode;
  - **ou** write/edit/patch explicitamente rejeitado por permission/policy;
  - canário de escrita ausente em ambos os casos;
- mudar role, substituir ou remover projection invalida enforcement antigo;
- ausência de evidência permanece `BEST_EFFORT` / `UNVERIFIED`.

## Custom execution targets

Provider e target são dimensões separadas.

Exemplo válido:

```text
provider = opencode
target_kind = docker
command_template =
  docker exec -it -w "{target_cwd}" vpn-dev-workspace-terminal-1 {provider} {args}
target_probe_template =
  docker exec vpn-dev-workspace-terminal-1 {provider} {args}
```

Placeholders:

```text
{cwd}
{target_cwd}
{args}
{provider}
{profile}
{profile_home}
```

O launcher resolve argv diretamente e não entrega o template a um shell.

No VPN Dev Workspace:

- Linux/WSL pode usar filesystem/home compartilhado quando o mount realmente o
  expõe ao host;
- Windows usa `target_workspace=/workspace`;
- a home padrão Windows do container vive em Docker volume; o Nexus não fabrica
  uma `target_profile_home` host-visible;
- sem shared profile home a execução custom continua válida e degrada somente
  a projection nativa/enforcement.

## Voz / fala — estado da beta.4

**Implementada no source da V1.**

Fluxos incluídos:

- botão de voz no Nexus e no terminal;
- captura Web por `getUserMedia` / `MediaRecorder`;
- fallback desktop Linux por `pw-record` ou `arecord` quando disponível;
- integração WebView2 no Windows;
- integração WKWebView/macOS com descrição de uso do microfone;
- Groq / `whisper-large-v3-turbo` como provider STT primário;
- Cloudflare Workers AI / Whisper como fallback configurável;
- providers habilitáveis/reordenáveis em Settings;
- secrets permanecem no backend;
- fallback tipado para falhas transitórias;
- inserção da transcrição no PTY sem Enter automático;
- comandos globais de navegação tratados por gramática determinística;
- voz desligada por padrão e acionada explicitamente.

**Código concluído não equivale a microfone físico certificado.** Cada plataforma
anunciada ainda precisa do smoke same-SHA de permissão, captura, transcrição,
fallback e inserção PTY.

## Estado da validação

GitHub Actions continua indisponível por bloqueio de créditos da conta. Isso é
falha de infraestrutura, não PASS e não evidência funcional de FAIL.

Enquanto o CI remoto estiver indisponível:

Linux/macOS:

```bash
./scripts/homologate-beta.sh
```

Windows:

```powershell
.\scripts\homologate-beta.ps1
```

A evidência da beta.4 deve ser escrita em:

```text
DEV/validation/local-beta/1.0.0-beta.4/<exact-sha>/<plataforma>/
```

## Matriz candidata

| Plataforma | Estado no source | Estado de certificação |
| --- | --- | --- |
| Linux amd64 | integrado | homologação beta.4 exact-SHA pendente |
| Windows amd64 | integrado | certificação nativa exact-SHA pendente |
| macOS arm64 | integrado | certificação nativa exact-SHA pendente |
| Web | integrado | homologação beta.4 pendente |

Implementação não equivale a suporte certificado.

## Gates para `v1.0.0-beta.4`

Antes de criar/publicar a tag:

- documentação consistente com beta.4;
- frontend format/lint/styles/typecheck/unit/build;
- Go fmt/vet/test/race;
- regressões de terminal e roteamento Mission;
- control/runtime E2E;
- build CLI;
- `nexus version --json` = `1.0.0-beta.4`;
- `nexus doctor --json`;
- OpenCode visível em providers/Usage/Agents quando instalado;
- role switch provider-neutral por overlay; OpenCode → projection automática ou fallback explícito;
- Command Palette + `Ctrl/Cmd+Shift+A` operam o mesmo role lifecycle sem recriar PTY nem capturar Shift+Tab;
- materialização + ownership + receipt + verify;
- discovery nativo;
- activation probe com confirmação;
- enforcement probe read-only:
  - effective permission/toolset do OpenCode confirma mutação desabilitada **ou** uma chamada write/edit/patch é explicitamente negada;
  - o canário permanece ausente;
  - somente então `ENFORCED`;
- custom target smoke com provider ausente no host;
- VPN Dev Workspace com host→target cwd mapping;
- HUD smoke;
- voice smoke:
  - permissão de microfone;
  - captura;
  - transcrição;
  - fallback entre providers quando configurado;
  - texto no PTY sem Enter automático;
  - comando global não destrutivo;
- Direct/Lead/Mission regression smoke;
- build Desktop/HUD da plataforma homologada.

## Gates para `v1.0.0` estável

Além da beta.4 homologada, o SHA final precisa comprovar:

- CI exact-SHA completo;
- security/govulncheck;
- Browser E2E, Axe e visual smoke;
- Desktop/HUD;
- lifecycle install/version/doctor/update/uninstall;
- cadeia de assinatura/update aplicável a cada plataforma anunciada;
- provider smoke autenticado;
- Agent Interop OpenCode certificado no SHA final;
- custom target certificado no SHA final;
- voice same-SHA em plataforma anunciada;
- Mission smoke até `COMPLETED_VERIFIED`;
- recovery/restart;
- Stable Certification/soak;
- zero findings BLOCKER/HIGH relevantes.

## Regra

A beta.4 é considerada **implementada no source** quando os requisitos acima
existem no candidato. Ela só é considerada **homologada** quando os gates
automatizados e manuais forem registrados contra o mesmo SHA.

A `v1.0.0` estável continua fail-closed: ausência de CI, assinatura,
homologação nativa ou evidência real nunca é convertida em PASS.


A publicação exige manifesto assinado contendo `source_sha` igual ao SHA exato promovido.

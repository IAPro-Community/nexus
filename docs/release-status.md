> **Nota de atualização documental (18/10/2026):** HEAD observado em `release/v1.0.0`: `30f7e69bcf85446634144c315de76e621c221ea5`; arquivo `VERSION`: `1.0.0-beta.10`. O relatório detalhado abaixo contém snapshots históricos (inclusive de beta.7) e **não certifica este HEAD**. Sem evidência de homologação exact-SHA, o veredito de publicação estável permanece **NO-GO**. Nenhum instalador/URL de download foi validado nesta revisão documental.

# Status público da release do Nexus v1

Atualizado em 8 de outubro de 2026.

| Campo | Estado |
| --- | --- |
| Programa | estabilização do Nexus 1.0 beta |
| Versão-fonte candidata | 1.0.0-beta.10 (ainda não homologada) |
| Branch canônica | release/v1.0.0 |
| SHA validado nos gates automatizados Windows locais | d840797bbd975a5c7edc54ebe23f3dfef56ff947 |
| SHA originalmente solicitado para homologação | d14154d2490dfdd3966c7fddc990a1d9127edd39 |
| Remoto observado antes da continuidade | origin/release/v1.0.0 = 52f35c04181b85379219ab55fc3192c63a5440a8 |
| Candidato local | release/v1.0.0 em d840797bbd975a5c7edc54ebe23f3dfef56ff947; descendente do SHA KIVER d14154d2 |
| Estado do worktree durante os gates | limpo no início da execução exact-SHA; documentação atualizada depois dos gates |

| Última beta pública registrada (último estado conhecido) | v1.0.0-beta.1; inventário remoto atual não revalidado |
| Release estável v1 | ainda não publicada |
| Gate nativo Windows exact-SHA no GitHub Actions | não revalidado; `gh` ausente neste host |
| Veredito | NO-GO para beta.10: os gates Windows PASS se referem somente ao SHA anterior d840797b (beta.7); faltam CI exact-SHA, validação de túnel/Codex e demais smokes aplicáveis |

## Convergência beta.10 — PRs #45 e #46

As correções de homologação Windows da PR #45 e de engenharia da PR #46
foram conciliadas na PR #47 para incorporação única na `release/v1.0.0`.
As mudanças recentes de janela/título IA foram preservadas. O registro
histórico da homologação beta.8 não foi promovido indevidamente como
certificação do código beta.10, e as capturas de tela precisam ser
regeneradas contra o SHA final antes de serem declaradas atuais.

**Estado técnico:** o merge de código não representa aprovação de release.
A validação do exato SHA (Linux, Windows, Codex/ConPTY, túnel, browser,
Desktop/NSIS e assinatura) permanece pendente. Não criar tag, nem publicar
instalador ou manifest de atualização sem esses gates.

## Candidato atual — beta.10 (2026-10-08)

A branch canônica `release/v1.0.0` avançou de beta.9 para beta.10. Esta iteração
preserva as correções recentes de Codex/PTY/JSON e de autenticação/túnel e acrescenta
cancelamento de inicialização do Cloudflare Tunnel desde Settings, publicação
atômica do processo, proteção por geração contra canceladores antigos e testes
de cancelamento antes e durante readiness. O comando `scripts/release-beta.mjs`
passou a atualizar `VERSION` e `web/package.json` em conjunto, revertendo ambos
quando a compilação falha.

**Nenhuma evidência anterior certifica automaticamente a beta.10.** O último
Windows PASS documentado continua ancorado em `d840797b` (beta.7).
Antes de disponibilizar nova beta ou instalador: congelar o SHA final; obter
`go test -race` (quando suportado), frontend e browser E2E, smoke real de
túnel iniciado/cancelado e recuperação de sessão, Codex TUI/entrada e
certificação Windows/Linux aplicável ao exato SHA; não publicar tag apenas pelo bump.

## Continuidade Windows — gates exact-SHA locais — 2026-10-08

O candidato foi validado em `d840797bbd975a5c7edc54ebe23f3dfef56ff947`, na branch local `release/v1.0.0`, com `VERSION=1.0.0-beta.7`, Windows AMD64, Go 1.25.14, Bun 1.3.9 e Node 22.14.0 para Playwright.

PASS — sintaxe PowerShell; docs-verify (177 Markdown/33 assets); instalação Bun frozen; `bun run --cwd web quality:full` (82 arquivos/487 testes); Go formatting, vet, golangci-lint v2.12.2 (0 issues), govulncheck v1.7.0 (sem vulnerabilidades), contrato de versão e `go test -count=1 -v ./...` (65 pacotes PASS, zero FAIL). PASS — regressões control/runtime e Agent Interop/OpenCode. PASS — oito suítes browser E2E, mais tarefas de acessibilidade e visual. PASS — builds CLI/Desktop/HUD e smokes nativos isolados: janela Wails + WebView2 e perfil Desktop, HUD ativo por oito segundos e perfil próprio criado.

Os binários e ZIPs, com hashes SHA-256, estão em `DEV/validation/local-beta/1.0.0-beta.7/d840797bbd975a5c7edc54ebe23f3dfef56ff947/windows-AMD64/distribution-artifacts/`. Os executáveis estão `NotSigned`. O arquivo público `UPDATE_PUBLIC_KEY` está versionado e é válido; este host não dispõe de `signtool.exe`, certificado/secrets Authenticode, `makensis.exe`, `gh`, `NEXUS_UPDATE_PRIVATE_KEY` nem `NEXUS_UPDATE_KEY_ID`. O instalador final e CI nativo exact-SHA não foram validados. Os smokes interativos de provider/Mission/Voice e captura visual do render Desktop também não foram executados neste SHA. `nexus doctor --json` registrou alertas de host, incluindo Gemini CLI que invoca Node 18.12.1 e não inicia.

Consulta viva `git ls-remote` em 2026-10-08 confirmou inicialmente `release/v1.0.0` em 52f35c04; os commits locais foram enviados em fast-forward para f1928d0f e uma sincronização documental posterior avançou o remoto para `5a08a1807c4489395dca86ed198f268ea676d2e4`. O SHA de código validado `d840797b` é ancestral desses commits documentais. A tag `v1.0.0-beta.7` continua ausente. Não surgiu branch Beta 7 nova. A branch KIVER em d14154d2 é ancestral do candidato; nenhuma integração/cherry-pick foi necessária. O Actions API não autenticado retornou 404, `gh` não está instalado e o estado do workflow exact-SHA permanece não confirmado; o inventário de GitHub Releases também não foi consultado. Nenhum gate Linux foi executado nesta continuidade, conforme escopo Windows.

**Veredito:** gates automatizados locais Windows PASS no SHA registrado; release e publicação continuam **NO-GO** até fechar CI same-SHA, assinatura Authenticode, build/smoke do instalador e gates manuais aplicáveis.

## Continuidade Windows — perfil WebView2 e branches — 2026-10-07

Na branch release/v1.0.0, o HEAD-base local é 52f35c04181b85379219ab55fc3192c63a5440a8 e VERSION contém 1.0.0-beta.7. Os gates Go passaram no Windows após as alterações locais do perfil WebView2 e do isolamento de ID para os binários de smoke. O workflow e o smoke PowerShell passaram na análise de sintaxe.

O smoke Desktop confirmou janela Wails, controlador Chrome_WidgetWin_0 e diretório próprio em LOCALAPPDATA\IAPro\Nexus\desktop\WebView2 por 9 segundos. O HUD permaneceu ativo por 8 segundos e criou seu perfil próprio. O render da janela não foi capturado visualmente nesta sessão. Os IDs de produção não mudaram; os binários usados no smoke tinham IDs temporários, portanto não são pacotes de distribuição.

A branch fix/beta7-windows-KIVER-20261007 em d14154d2 é ancestral da release local em 52f35c04. A comparação local encontrou cinco commits posteriores em release e nenhum commit exclusivo em fix; não há merge de branch pendente. Os refs remotos são apenas o snapshot já presente no clone e não foram atualizados pela rede.

A homologação desta árvore ainda não é exact-SHA: o checkout está sujo e em 52f35c04, não em d14154d2. Assinatura Authenticode, trust root de atualização, rebuild e smoke final do instalador e CI nativo no SHA imutável permanecem pendentes. A release segue NO-GO; nenhum gate Linux foi executado nesta continuação.

## Homologação técnica Windows AMD64 — 2026-10-07

A branch release/v1.0.0 permanece no SHA-base 52f35c04181b85379219ab55fc3192c63a5440a8 e VERSION contém 1.0.0-beta.7. A matriz anterior passou nesse SHA imutável antes das correções locais mais recentes. A árvore atual inclui alterações sem commit para preservar HKCU Environment Path no NSIS, permitir o Host Wails exato e corrigir o ProjectHub; esta revalidação é de worktree sujo baseado no SHA 52f, não certificação exact-SHA do código modificado.

Nesta continuação Windows passaram go fmt, go vet, govulncheck, golangci-lint, go test -count=1, Bun frozen install e quality:full (Bun 1.3.9, Node 24.19.0, 82 arquivos e 487 testes). As oito suítes Playwright passaram no worktree Windows isolado com as mesmas alterações. Os builds CLI/Desktop/HUD foram gerados para validação e têm checksums nos artefatos da sessão. A captura reproduzível mostra o erro ocupando toda a largura e Adicionar Projeto em uma linha no viewport 775x483.

O HUD passou no smoke Windows. O binário Desktop oficial criou a janela Wails IAPro Nexus e o filho WebView2 Chrome_WidgetWin_0, permaneceu ativo por oito segundos com APPDATA e dados Nexus isolados e encerrou sem processos residuais. A janela foi criada em modo oculto; renderização visível permanece sem captura porque este host não fornece input desktop.

O run nativo 37666126714 foi observado como queued sem runner Windows compatível; o estado atual não foi revalidado porque gh não está instalado e github.com não resolve por DNS neste host. Faltam certificado/secrets Authenticode, signtool.exe e NEXUS_UPDATE_PUBLIC_KEY. Os artefatos novos são NotSigned e a árvore continua WORKTREE_FROZEN: NO. Não há tag local v1.0.0-beta.7; tag e GitHub Release remotas não foram revalidadas neste host. Veredito: gates locais de código Windows PASS com limitação no reteste Desktop; assinatura, tag e distribuição NO-GO. Nenhum gate Linux foi executado nesta continuação.

## Homologação Linux amd64 — 2026-09-30

A homologação automatizada de exact-SHA em Linux amd64 terminou **PASS** no
SHA `6bf1b9fef578822d2be9d542409a44a2e154ed96` (`scripts/homologate-beta.sh`,
19 gates: documentação, frontend, Go fmt/vet/lint/security, contrato de versão,
race suite, E2E de control/runtime, interop, build CLI, 10 tarefas de E2E de
browser, projeção OpenCode, desktop/HUD, snapshot GoReleaser e smoke do DEB).

Relatório completo e matriz de checks manuais pendentes:
`DEV/validation/BETA7_LINUX_FINAL_20260930.md` (documentação interna do
repositório).

Este PASS **não** certifica Windows, macOS, GitHub Actions nem os smokes
manuais interativos (Codex/AGY/OpenCode, HUD, Voice). O CI continua indisponível
porque o runner self-hosted `nexus-ci-dev-web-kiver` está offline — não por
créditos da conta. Nada foi publicado: não há tag beta.7 nova.

## Linha canônica de integração — 2026-09-27

A branch `release/v1.0.0` é a única linha canônica de integração da Nexus 1.0. As iterações beta continuam identificadas por `VERSION`/manifesto (`1.0.0-beta.N`), sem criar uma branch nova a cada beta.

- PR #25 foi integrado por squash em `8fa37f4ca8051b2abce1ade3a0c83dd5b5be64d7`;
- essa integração contém a remediação Windows `ce52b09d...`, a convergência Linux/Windows, Direct/Mission OFF-by-default e o hardening Codex/ConPTY;
- a lista única de gaps e critérios de homologação está na issue #27;
- Usage/Capacity V2 separa quota de assinatura, consumo/custo de API e score interno de roteamento; `quota_remaining` voltou a ter semântica literal 0..1;
- seleção multi-conta usa `ESTIMATED` apenas quando não existe `LIVE/CACHED`, mantendo a evidência fresca como tier superior;
- a workflow de release rejeita qualquer SHA que não seja exatamente o HEAD de `release/v1.0.0`, impedindo publicação por refs beta históricas;
- a migração Wails v3 do PR #24 foi portada seletivamente para a linha canônica, preservando os contratos mais novos de Agent Projection, ExecutionRole, Direct/Mission, OpenCode e homologação; o PR histórico não deve ser mergeado em bloco;
- a validação de diretório de workspace e a proveniência assinada `source_sha` foram recuperadas de branches históricas e incorporadas diretamente na linha canônica;
- integração na branch beta não equivale a PASS, tag ou publicação.

A linha canônica já usa a migração Wails v3 reconciliada. A publicação permanece bloqueada pelos gates same-SHA e pelos blockers registrados na issue #27.

## Escopo consolidado da beta.7

A linha que antes estava planejada como 1.1 foi absorvida pela V1 enquanto a
primeira release estável ainda não foi publicada. A beta.7 concentra a conclusão
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

Na beta.7, `ExecutionRole` é um contrato do Nexus, não do provider:

- Codex, Claude, Cursor, Gemini e AGY continuam usando o mesmo selector e
  compiler por **Nexus overlay**;
- OpenCode é o adapter nativo de referência da beta.7;
- ausência de native adapter nunca impede role switch/overlay;
- ausência de prova nativa nunca aparece como primary/subagent/enforcement
  suportado;
- adapters nativos adicionais são expansão posterior e não bloqueiam o
  fechamento funcional da beta.7.

A Web também inclui:

- selector de role no terminal;
- roles do terminal ativo na Command Palette;
- `Ctrl/Cmd+Shift+A` para focar o selector;
- `Shift+Tab` continua pertencendo ao TUI hospedado e não é capturado pelo Nexus.

## Agent Interop / OpenCode — estado da beta.7

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

## Voz / fala — estado da beta.7

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

GitHub Actions está atualmente **BLOCKED_EXTERNAL** por indisponibilidade de
créditos para executar os gates. O run exact-SHA do candidato permanece
pendente; ausência de execução não é PASS nem FAIL técnico.

A publicação continua fail-closed: PR CI, quando disponível, valida o merge
sintético, enquanto a release exige um run `push`/`workflow_dispatch` verde
no SHA exato já integrado em `release/v1.0.0`. A homologação local/nativa pode
produzir evidência complementar, mas não autoriza reinterpretar um gate não
executado como aprovado.

Para complementar a evidência remota ou quando um runner nativo não estiver disponível:

Linux/macOS:

```bash
./scripts/homologate-beta.sh
```

Windows:

```powershell
.\scripts\homologate-beta.ps1
```

A evidência da beta.7 deve ser escrita em:

```text
DEV/validation/local-beta/1.0.0-beta.7/<exact-sha>/<plataforma>/
```

## Matriz candidata

| Plataforma | Estado no source | Estado de certificação |
| --- | --- | --- |
| Linux amd64 | integrado | homologação beta.7 exact-SHA pendente |
| Windows amd64 | integrado | certificação nativa exact-SHA pendente |
| macOS arm64 | integrado | certificação nativa exact-SHA pendente |
| Web | integrado | homologação beta.7 pendente |

Implementação não equivale a suporte certificado.

## Gates para `v1.0.0-beta.7`

Antes de criar/publicar a tag:

- documentação consistente com beta.7;
- frontend format/lint/styles/typecheck/unit/build;
- Go fmt/vet/test/race;
- regressões de terminal e roteamento Mission;
- control/runtime E2E;
- build CLI;
- `nexus version --json` = `1.0.0-beta.7`;
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

Além da beta.7 homologada, o SHA final precisa comprovar:

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

A beta.7 é considerada **implementada no source** quando os requisitos acima
existem no candidato. Ela só é considerada **homologada** quando os gates
automatizados e manuais forem registrados contra o mesmo SHA.

A `v1.0.0` estável continua fail-closed: ausência de CI, assinatura,
homologação nativa ou evidência real nunca é convertida em PASS.


A publicação exige manifesto assinado contendo `source_sha` igual ao SHA exato promovido.

## Atualização Windows — smoke Desktop/WebView2 — 2026-10-07

O workflow native-windows agora inclui scripts/smoke-desktop-windows.ps1, com APPDATA isolado, verificação da janela Wails e do filho Chrome_WidgetWin_0 por oito segundos, e cleanup dos processos WebView2. O script passou em Windows; ci.yml é YAML válido. Essa mudança ainda está no worktree local sujo e não foi executada no runner exact-SHA.

A falha 0x800700AA ocorreu somente no perfil APPDATA padrão deste host; o binário oficial passou com perfil isolado. Isso valida a criação e estabilidade da janela/controller, não captura visual em desktop interativo. O host não expôs input desktop.

O SHA-base local continua 52f35c04181b85379219ab55fc3192c63a5440a8, e VERSION é 1.0.0-beta.7. O estado vivo do remoto, da fila 37666126714, da tag e do GitHub Release está não revalidado devido à ausência do gh e à falha de DNS. O trabalho segue sem commit; assinatura e trust root continuam ausentes. Escopo Windows-only; nenhum gate Linux foi executado.

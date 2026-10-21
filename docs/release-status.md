# Status público da release do Nexus v1

Atualizado em 8 de outubro de 2026.

| Campo | Estado |
| --- | --- |
| Programa | estabilização do Nexus 1.0 beta |
| Versão-fonte candidata | 1.0.0-beta.7 |
| Branch canônica | release/v1.0.0 |
| HEAD local/remoto | HEAD local `e56c091f5038a8c972c339f47a960167077e45d8`; ref remota cacheada `81dea07b3fba6d08fbf80c71c17c09518435aa64`; live não revalidado por DNS |
| SHA validado nos gates automatizados Windows locais | `81dea07b` — gates Go PASS; `quality:full` bruto FAIL |
| SHA originalmente solicitado para homologação | `d14154d2490dfdd3966c7fddc990a1d9127edd39` |
| Relação entre os SHAs | `d14154d2` está 18 commits atrás/0 à frente do snapshot de release `81dea07b`; HEAD local `e56c091f` está um commit atrás da ref cacheada |
| Relação com o código validado | Go validado em `81dea07b`; gate frontend bruto não passou no mesmo SHA |
| Worktree desta retomada | Checkout local dirty; worktree isolado 81dea07b contém dois ajustes locais de teste e evidências E2E geradas |

| Última beta pública conhecida | v1.0.0-beta.1; inventário remoto não revalidado nesta continuidade |
| Tag local v1.0.0-beta.7 | ausente |
| Release estável v1 | ainda não publicada |
| Veredito Windows | NO-GO: gates Go PASS, mas `quality:full` bruto falha no SHA 81dea07b |
| Bloqueios | corrigir e repetir quality:full em SHA limpo; CI exact-SHA, assinatura/trust root e instalador Windows final |

## Registro anterior — snapshot e56c091f (substituído pela atualização atual)

O checkout `release/v1.0.0` e a ref viva `origin/release/v1.0.0` estão em `e56c091f5038a8c972c339f47a960167077e45d8`; `VERSION` contém exatamente `1.0.0-beta.7`. O SHA de código validado `d840797bbd975a5c7edc54ebe23f3dfef56ff947` é ancestral, com nove caminhos documentais no intervalo commitado. A árvore local tem agora uma alteração frontend ainda não commitada.

No Windows passaram `bun run quality:full` com Node 24.19.0 (83 arquivos/490 testes/build), o teste do seletor de caminho (3/3) e `bun run test:e2e`: o cenário 403 `invalid origin` no viewport 775×483 manteve o alerta na largura do formulário e o CTA em uma linha; rolagem, Axe e criação válida depois da interceptação também passaram. O placeholder nos formulários agora usa caminho Windows/macOS/POSIX localizado. A captura nativa mostra o tema escuro e `C:\Projetos\meu-repositorio`; o mínimo de janela Wails ficou em 784×561.

O smoke Desktop passou com ID temporário exclusivo e diretórios de dados isolados, sem deixar processos de teste após cleanup. A tentativa inicial com o executável normal encontrou a instância interativa do usuário e saiu com código 0 devido ao ID singleton de produção; a instância foi preservada. O hash do binário de smoke é `49B629C43A58C5CCAFC5E4C93F3D50939F193382E9EF788E6F9C396F8C0F8D7A`; não é artefato de distribuição.

Os smokes autenticados de Codex/AGY/OpenCode, Mission e Voice permanecem pendentes: Nexus não tem perfis habilitados para Codex/AGY e OpenCode CLI não está instalado. O host segue sem `gh`, `signtool.exe`, `makensis.exe`, certificado/chave Authenticode e secrets `NEXUS_UPDATE_PRIVATE_KEY`/`NEXUS_UPDATE_KEY_ID`; os artefatos de distribuição anteriores são `NotSigned`.

A auditoria viva das branches não encontrou branch Beta 7 nova que precise ser vinculada à release. KIVER em d14154d2 é ancestral; protocol-p0, trace-diagnostics e final-boundary têm 4/24/3 commits exclusivos no grafo, mas as alterações relevantes estão cobertas; windows-autopilot tem um commit documental. beta8 e v1.1 são linhas separadas. gap-closure tem 38 commits exclusivos e ampla divergência; não integrar em bloco. A ref remota da tag `v1.0.0-beta.7` não foi encontrada; o inventário de GitHub Releases não foi consultado.

`docs-verify` também está bloqueado pelo manifesto visual, que exige SHA imutável e rejeita as três fontes frontend não commitadas.

Veredito: validações locais Windows PASS nos escopos e revisões registrados; worktree não congelado e **NO-GO** para tag/distribuição até atualizar o manifesto visual em SHA imutável, fechar CI exact-SHA, assinatura/trust root, instalador e smokes manuais. Nenhum gate Linux foi executado nesta continuidade.
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

## Atualização de continuidade Windows — snapshot 81dea07 — 2026-10-08

O checkout local está em `e56c091f5038a8c972c339f47a960167077e45d8`; a ref de rastreamento cacheada para `release/v1.0.0` está em `81dea07b3fba6d08fbf80c71c17c09518435aa64`. A tentativa de consulta viva falhou por DNS. `VERSION` contém exatamente `1.0.0-beta.7`.

No Windows AMD64, todos os gates Go e a instalação Bun frozen passaram no snapshot `81dea07b`. O `bun run quality:full` bruto falhou em formatação de `ptyLiveChrome.test.ts` e, após essa formatação local, em uma expectativa desatualizada de `taskbarHonesty.test.ts`. A execução passou com 82 arquivos/488 testes/build apenas após ajustes locais não commitados nos dois testes; por isso o gate do SHA remoto permanece FAIL. Oito suítes browser E2E, builds CLI/Desktop/HUD e smokes nativos Desktop/HUD passaram no worktree temporário com alterações de teste locais.

Nas refs cacheadas, KIVER `d14154d2` tem zero commits exclusivos frente à release `81dea07b`; não há integração pendente conhecida. O refresh remoto falhou por DNS, portanto branches criadas depois do último fetch não foram verificadas. A tag local beta.7 não existe; tag remota e GitHub Releases não foram reconsultados.

**Veredito: NO-GO para tag e distribuição.** Repetir o `quality:full` bruto após integrar os dois ajustes de teste em um SHA limpo e imutável; concluir os bloqueios de CI exact-SHA, assinatura/trust root e instalador Windows. Nenhum gate Linux foi executado nesta continuidade.

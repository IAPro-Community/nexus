# Homologação Linux — Nexus 1.0.0

## Decisão

**NO_GO_LINUX**

A release não atende aos gates de Companion 2D, E2E, publicação estável e validação visual/funcional nativa. A aplicação foi compilada e iniciada como Wails em GNOME/Wayland; isso não substitui a evidência de interação visual bloqueada pelo compositor.

## Identidade do código

- Branch obrigatória: `release/v1.0.0`.
- SHA remoto e local confirmado no início e no fetch final: `f62fdec0c571ac6a19a904164d3f580799da6bf3`.
- Estado da versão no código: `1.0.0-beta.10`; não é uma versão estável `1.0.0`.
- PR #55 (Companion), #57, #58, #59 e #61 continuavam abertas. PR #62 já fazia parte da branch. Nenhuma PR foi mesclada durante esta campanha.
- A reconciliação anterior em `DEV/validation/NEXUS_100_BRANCH_COMPANION_RECONCILIATION_20261022.md` é uma fotografia do SHA `9ff1889` e declarou explicitamente não ser certificação nativa. Revalidamos a branch e as PRs no HEAD atual; na consulta final, #55 ainda era draft aberta, apontava para `release/v1.0.0` e tinha conflito.
- Correções aplicadas localmente, sem commit: SHA-256 do patch `5557c3bccf606baffcdc8e3db1e44c7cd423d0da570baf69f9acb8744d749b10`. O SHA-base acima identifica a release avaliada; builds corrigidos são `f62fdec0 + FIXES.patch` e não são uma nova revisão publicada.

## Ambiente e execução

Ubuntu 25.10 AMD64, kernel `6.17.0-23-generic`, GNOME 49/Mutter, sessão Wayland, VMware SVGA II sem aceleração 3D. GTK3/WebKitGTK 4.1 e toolchain Wails do projeto foram detectados. Desktop e HUD foram compilados e iniciados; o Desktop serviu a home/onboarding via Core embutido HTTP 200, e criou um processo irmão `nexus-hud --linked`. AT-SPI confirmou a janela nativa e conteúdo acessível. Não foi possível capturar pixels nem enviar ações confiáveis à janela: o compositor negou as rotas de screenshot e a ação AT-SPI expirou.

## Resultado resumido

A matriz tem **59 registros**: **36 PASS, 9 FAIL, 10 BLOCKED e 4 NOT_RUN**. Os testes automáticos Go e frontend de qualidade passaram; o race detector também passou. Cinco suítes browser E2E falharam. Builds CLI/Desktop/HUD e snapshot local CLI/DEB passaram. Nenhuma instalação de sistema ou publicação ocorreu.

O gate crítico resultou em **`NO-GO — COMPANION_2D_NOT_INTEGRATED`**: no SHA testado, o HUD seleciona `GuardianMascot` antigo. `NexusGuardian.tsx`, `companionModel`, atlas de sprites e animações da PR #55 não existem na branch. Não substituímos o Companion por imagens do Guardian.

A distribuição oficial também não está pronta: a branch ainda é beta.10 e o GitHub não apresenta release/tag estável `v1.0.0` nem manifesto assinado. O instalador real falhou de modo seguro em HTTP 404 e não instalou arquivo algum. Um `.deb` local foi produzido em snapshot com a versão `1.0.0` forçada; não foi publicado e não muda o versionamento do código.

## Bloqueios e falhas que impedem GO

- **P1 — Companion 2D ausente.** PR #55 permanece aberta, draft e sem mergeabilidade limpa.
- **P1 — E2E browser:** `composer`, `surface`, `composer-spine`, `composer-vague-spine` e `composer-themes` falharam. Nos 20 temas, a configuração foi selecionada, mas a rota não apresentou `.nx-composer-surface`.
- **P1 — Certificação visual/funcional nativa incompleta.** GNOME bloqueou screenshots; AT-SPI não conseguiu executar ações. Barra/Mascote, transparência, click-through, foco, posição, persistência e preferências do HUD não foram homologados visualmente.
- **P1 — Release estável inexistente.** Fonte informa beta.10; preview atual não corresponde ao SHA homologado e é unsigned.
- **P1 — Provedores autenticados e continuidade live não validados.** Não foi possível confirmar credenciais de perfil para chamadas externas.
- **P2 — `docs-verify` falha** por manifesto visual desatualizado. Ele não foi regenerado artificialmente.
- **NOT_RUN:** sessão X11 separada, escala/segundo monitor, soak/crash-recovery/múltiplas instâncias e ShellCheck.

## Evidências

- [Matriz completa de testes](TEST_MATRIX.md) e [resumo JSON](LINUX_HOMOLOGATION_SUMMARY.json).
- [Ambiente](ENVIRONMENT.md), [runtime/agentes/terminais](RUNTIME_VALIDATION.md), [HUD](HUD_VALIDATION.md), [Companion](COMPANION_VALIDATION.md) e [distribuição](DISTRIBUTION_VALIDATION.md).
- [Correções locais](FIXES_APPLIED.md), [patch reproduzível](FIXES.patch), [logs](logs/) e [nota de screenshots](screenshots/README.md).

## Comandos para repetir

A partir de uma cópia Linux AMD64 do repositório:

```bash
git fetch origin --prune
git checkout release/v1.0.0
git rev-parse HEAD
git rev-parse origin/release/v1.0.0
bun install --cwd web --frozen-lockfile
bun run --cwd web quality:full
make format-check
GOTOOLCHAIN=go1.26.10 go test -tags=gtk3 -count=1 -timeout=25m ./...
GOTOOLCHAIN=go1.26.10 go test -race -tags=gtk3 -count=1 -timeout=25m ./...
GOTOOLCHAIN=go1.26.10 go vet -tags=gtk3 ./...
GOTOOLCHAIN=go1.26.10 make build
GOTOOLCHAIN=go1.26.10 make build-desktop-wails
GOTOOLCHAIN=go1.26.10 make build-hud-wails
```

Para reproduzir falhas E2E específicas depois de `make build`, execute `bun run --cwd web test:e2e:composer`, `test:e2e:surface`, `test:e2e:composer-spine`, `test:e2e:composer-vague-spine` e `test:e2e:composer-themes`. Para a instalação publicada, use `./install.sh --version=1.0.0 --no-path` em diretório temporário; no estado observado ela encerra em 404 porque não há manifesto estável. O smoke Wails requer sessão de desktop ativa e permissões que permitam captura/input para concluir a parte visual.

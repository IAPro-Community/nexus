# Homologação Linux AMD64 — Nexus

Início: 2026-10-24; atualização: 2026-10-25  
Resultado: **NO_GO_LINUX**  
Branch: release/v1.0.0  
HEAD local e remoto: bf6adf0c80ac3531d7291844512a14a017ab0002  
Fingerprint das fontes finais de produto compiladas: 84bef2759af6b975d73e0488ab80e5fced2df70f17855597d512dcdef1f29ec8  
Arquivos de produto no manifesto: 1.694  
Versão informada pelo binário: 1.0.0-beta.10

## Identidade do que foi testado

O fetch mais recente confirmou HEAD local e origin/release/v1.0.0 em bf6adf0c80ac3531d7291844512a14a017ab0002. A árvore continua modificada e não congelada. O manifesto final registra 1.694 arquivos de produto e fingerprint 84bef2759af6b975d73e0488ab80e5fced2df70f17855597d512dcdef1f29ec8. CLI, Desktop, HUD e pacotes foram reconstruídos com esse overlay. As screenshots anteriores pertencem ao overlay visual 6476b22e5ae0b69e6f7340d13915ad013e0151dd5119fe91142cbef69441b02e; a recaptura dos binários exatos finais passou em NATIVE-004, com imagens versionadas e hashes. O SHA Git é a base identificável, não substitui o fingerprint do overlay. Não houve commit, push, merge, tag ou publicação.

## Direção do mascote

O objetivo do produto é **um único mascote NEX**, bem acabado e funcional. Um Companion 2D separado e seu modo não fazem parte do produto nem da release. O seletor ativo oferece Barra, Mascote e Bandeja; Mascote exibe o NEX único. A lógica chamada companionModel no código trata estados e atenção do mesmo mascote; não renderiza uma segunda personagem. A PR #55 segue aberta, draft e conflitante; não foi incorporada.

## Resultado da homologação

| Status | Casos |
| --- | ---: |
| PASS | 40 |
| FAIL | 1 |
| BLOCKED | 10 |
| NOT_RUN | 9 |
| Total | 60 |

- **Backend Go:** quality-full, formatação, testes, race, vet, lint e govulncheck passaram nos gates executados.
- **Frontend:** quality:full passou: 103 arquivos e 633 testes; build de produção concluído. A suíte inclui a regressão que garante uma única arte NEX por renderização. O comando de login padrão foi corrigido para não duplicar o provedor.
- **E2E:** HUD passou; 20 temas foram verificados em três tamanhos de viewport, com persistência e Axe sem achados sérios ou críticos.
- **Build Linux:** CLI, Desktop Wails e HUD finais foram compilados para AMD64 (hashes em `logs/build-resilience-final-sha256.txt`); `ldd` não apontou dependências ausentes. CLI tar.gz, DEB, e tarballs Desktop/HUD foram gerados novamente depois das correções de resiliência, e o DEB foi instalado/removido em raiz temporária.
- **Desktop real:** screenshots XWayland do overlay visual anterior 6476b22e5ae0b69e6f7340d13915ad013e0151dd5119fe91142cbef69441b02e registram workspace, terminal, ANSI, Unicode, resize PTY, alternância Barra/Mascote e click-through. Depois das correções de resiliência, os binários exatos do overlay 84bef2759af6b975d73e0488ab80e5fced2df70f17855597d512dcdef1f29ec8 abriram sob o D-Bus da sessão real; Desktop 1280×800 e HUD 224×158 foram capturados, ambos com Core HTTP 200 e encerramento exit 0. O protocolo visual final passou em NATIVE-004.
- **Mascote:** o caminho ativo monta uma única `GuardianMascot` que renderiza `NexPixelSprite`; a screenshot exata do HUD final mostra uma arte NEX e nenhuma segunda personagem. Estados/atenção por runtime real permanecem BLOCKED.
- **Sessões locais:** Codex, Gemini, AGY e OpenCode foram importados em perfis temporários; arquivos de origem preservados. Codex CLI respondeu READY em smoke isolado. O arquivo Claude está sem tokens e corretamente retorna `NO_SESSION`. Cursor reporta autenticação no CLI, mas não oferece formato público de exportação/copiar credenciais; o importador Nexus permanece `UNSUPPORTED`.
- **Distribuição:** GoReleaser gerou CLI tar.gz e DEB de snapshot; Desktop/HUD foram empacotados em tar.gz e comparados aos executáveis. O DEB foi instalado/removido em um dpkg root temporário; validação de config e persistência em perfil isolado também passaram; nada foi instalado globalmente.
- **Logs:** 65 arquivos de texto foram examinados por padrões comuns de API key, JWT e Bearer; nenhum padrão foi encontrado. Isso é uma varredura heurística, não auditoria completa.

## Por que não recebe GO_LINUX

Há uma falha de release: a árvore ainda declara 1.0.0-beta.10 e não há release estável v1.0.0 nem assets Linux estáveis. O preview público mais recente é de 22/10 e anterior ao HEAD testado. O overlay ainda não foi congelado em commit; `make docs-verify` também permanece bloqueado pela política do manifesto visual para arquivos modificados sem SHA commitado.

A captura visual nativa exata passou em XWayland. Continuam sem homologação no runtime Nexus: roteamento/continuidade com agentes reais, skills em sessão, atenção originada por agente e importação Cursor. Wayland direto, alpha confiável, updater assinado/rollback e matriz de animação/GPU permanecem BLOCKED ou NOT_RUN. A máquina usa VMware SVGA II sem aceleração 3D utilizável; as capturas do HUD foram feitas em XWayland.

O resultado é **NO_GO_LINUX**. Não há P0 observado nos gates automatizados executados; ainda há um FAIL P1 de release e vários gates P1 sem validação. O status não se estende a Windows ou macOS.

## PRs abertas conferidas

- #55: draft e conflitante; propõe a personagem Companion 2D separada, que não corresponde ao objetivo de mascote único.
- #57: draft/mergeable; visibilidade de worktrees não homologada nesta release.
- #58: draft/mergeable; terminais divididos e comandos contextuais não homologados nesta release.
- #59: aberta, não draft e conflitante; correções HUD não incorporadas automaticamente.
- #61: draft/mergeable; portal de documentação.
- #64: direcionada a main; sem efeito direto na branch.
- Nenhuma PR foi mesclada por esta homologação; nenhuma tag ou release foi publicada.

## Correções locais realizadas

- **PTY:** resize do painel agora ajusta a grade do PTY sem resize da janela; observado 21×79 → 25×91 e testes de regressão passaram.
- **Mascote único:** o caminho ativo tem uma única `GuardianMascot` com `NexPixelSprite`; não há renderer/seleção de Companion 2D. O seletor de regressão limita apresentações a Barra, Mascote e Bandeja, e o componente exige uma arte NEX por renderização. PR #55 não foi integrada.
- **Login de provedor:** os dois fluxos da interface usam o mesmo formatador; o perfil padrão codex gera nexus login codex, enquanto perfis nomeados mantêm o identificador.

As alterações permanecem locais, não commitadas. Go efetivo no build final: 1.25.14 (go.mod declara 1.25.0). Detalhes e evidências em [FIXES_APPLIED.md](FIXES_APPLIED.md).

## Evidências

- [Matriz com procedimento, esperado, observado, status, severidade e commit-base por caso](TEST_MATRIX.md)
- [Resumo JSON estruturado](LINUX_HOMOLOGATION_SUMMARY.json)
- [Ambiente](ENVIRONMENT.md)
- [Validação do mascote único](COMPANION_VALIDATION.md)
- [Validação do HUD](HUD_VALIDATION.md)
- [Agentes e runtime](RUNTIME_VALIDATION.md)
- [Distribuição e atualização](DISTRIBUTION_VALIDATION.md)
- [Correções aplicadas](FIXES_APPLIED.md)
- [Varredura heurística de segredos dos logs](logs/credential-pattern-scan.md)
- [Manifesto e fingerprint final do código](logs/source-provenance-linux-final-2026-10-25.txt)
- [Captura Desktop exata do overlay final](screenshots/native-desktop-resilience-final-xwayland.png) (fingerprint 84bef2759af6b975d73e0488ab80e5fced2df70f17855597d512dcdef1f29ec8)
- [Captura HUD exata do overlay final — um NEX](screenshots/native-hud-resilience-final-xwayland.png) (fingerprint 84bef2759af6b975d73e0488ab80e5fced2df70f17855597d512dcdef1f29ec8)
- [Captura Desktop do overlay visual anterior](screenshots/native-desktop-exact-final-xwayland.png) (fingerprint 6476b22e5ae0b69e6f7340d13915ad013e0151dd5119fe91142cbef69441b02e)
- [Captura NEX do overlay visual anterior](screenshots/native-hud-exact-final-xwayland.png) ([hashes](logs/native-exact-final-visual-evidence.txt); fingerprint 6476b22e5ae0b69e6f7340d13915ad013e0151dd5119fe91142cbef69441b02e)
- [Hashes e startup das capturas exatas finais](logs/native-final-resilience-visual-evidence.txt)
- [Registro da tentativa inicial com D-Bus privado e reteste aprovado](logs/native-final-resilience-visual-blocked.md)
- [Captura do modo Mascote](screenshots/native-single-mascot-hud-final-xwayland.png)
- [Captura do modo Barra](screenshots/native-single-mascot-hud-bar-xwayland.png)
- [Captura dos controles](screenshots/native-single-mascot-hud-expanded-xwayland.png)
- [Capturas dos temas](screenshots/theme-e2e/e2e-theme-nexus-dark-desktop.png), [móvel](screenshots/theme-e2e/e2e-theme-paper-flat-mobile.png), [alto contraste escuro](screenshots/theme-e2e/e2e-theme-high-contrast-dark-desktop.png) e [alto contraste claro](screenshots/theme-e2e/e2e-theme-high-contrast-light-desktop.png). As capturas de temas são do E2E browser, não de GTK nativo.

## Reprodução principal

Na raiz, executar `GOCACHE=/tmp/nexus-linux-100-2026-10-24/gocache-native make quality-full`, `cd web && bun run quality:full`, `bun run test:e2e:hud`, `bun run test:e2e:composer-themes` e `make build build-desktop build-hud`. O GoReleaser snapshot final foi reproduzido com `/home/desenvolvedor/go/bin/goreleaser release --snapshot --clean --config /tmp/nexus-linux-100-2026-10-24/goreleaser-final-resilience-2026-10-25.yaml`. Os logs finais estão nos links da matriz.

Os binários e snapshots estão em /tmp/nexus-linux-100-2026-10-24. O DEB foi instalado e removido em uma raiz dpkg temporária; a instalação ativa do host não foi alterada. Para investigar a importação, consultar os casos BLK-002 a BLK-004 e executar o fluxo em perfil Nexus temporário; nunca registrar conteúdo das credenciais. O caso de porta ocupada foi reproduzido com dois Cores na porta 13241: a segunda instância retorna `address already in use`, e a primeira mantém `/api/v1/health` em HTTP 200. O caso de instâncias em portas distintas passou no build final: iniciar em 13241/13242, encerrar a segunda primeiro, confirmar que a primeira permanece saudável e verificar que seus arquivos de descoberta são isolados por PID. Logs em `core-multi-instance-*.log`. Para reproduzir o resize PTY, abrir o terminal, consultar stty size, redimensionar o painel sem redimensionar a janela e consultar stty size novamente.

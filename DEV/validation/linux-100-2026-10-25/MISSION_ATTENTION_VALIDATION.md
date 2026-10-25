# Validação do Attention Center e abertura de missões

- Data: 2026-10-25
- Branch: `release/v1.0.0`
- Base Git local/remota: `bf6adf0c80ac3531d7291844512a14a017ab0002`
- Fingerprint do overlay compilado: `7368531977b7422eccdf00e7c676f57d440e8d736d9e114bcdc071b9d5885fe6` (1.410 arquivos do escopo descrito no manifesto)
- Binário Desktop Wails SHA-256: `f6e1afcf2e7b0e10ed5d75a5d716b22dfe6f6ca894e056c64938599172ee7f15`

Esta é uma validação complementar ao relatório de homologação. O worktree continua sujo e sem commit imutável; estes resultados não mudam o veredito geral **NO_GO_LINUX**.

## Matriz desta rodada

| ID | Área | Procedimento | Resultado observado | Status | Severidade | Evidência |
| --- | --- | --- | --- | --- | --- | --- |
| ATTN-API-01 | Backend | `go test ./internal/nexus/runner ./internal/nexus ./internal/control/web -count=1` | Filtros por projeto e missão passaram nos testes do Attention Center e do handler. | PASS | — | `logs/go-mission-attention-regression.log` |
| ATTN-API-02 | Backend | `go vet ./internal/nexus/runner ./internal/nexus ./internal/control/web` | Sem diagnósticos; exit code 0. | PASS | — | `logs/go-mission-attention-vet.log` |
| ATTN-API-03 | Persistência | `go test ./internal/nexus -run '^TestGlobalAttentionProjectionSurvivesStoreReopen$' -count=1 -v` | Cinco missões determinísticas sobreviveram ao close/reopen do SQLite; A/D/E continuaram em Needs You/Completed/Failed, B/C em execução permaneceram fora e os filtros não cruzaram projetos. | PASS | — | `logs/go-mission-attention-persistence-reopen.log`; `logs/go-mission-attention-persistence-vet.log` |
| ATTN-API-04 | Autenticação HTTP | Teste do endpoint `/api/v1/attention` sem sessão e com cliente autenticado usando filtros sem correspondência | Sem autenticação, a API respondeu 401; com cliente autenticado, respondeu 200 e a projeção filtrada veio vazia. | PASS | — | `logs/go-mission-attention-auth-endpoint.txt`; `logs/go-mission-attention-auth-endpoint-vet.txt`; `logs/mission-attention-auth-test-provenance.txt` |
| ATTN-WEB-01 | Frontend | `cd web && bun run quality:full` | 108 arquivos/672 testes passaram; arquitetura de estilos, formatação, ESLint, Stylelint, TypeScript e build passaram. | PASS | — | `logs/web-quality-full-mission-attention-route-layout.log` |
| ATTN-NATIVE-01 | Desktop Linux | Iniciar o Wails em perfil isolado no GNOME Wayland/XWayland e abrir a Central de Atenção | A aba Missões mostrou a fixture controlada, contagem 1, filtros e estado “Precisa de você”. | PASS | — | `screenshots/mission-attention-drawer-native-xwayland.png` |
| ATTN-NATIVE-02 | Navegação | Clicar no cartão da fixture na janela Wails real | A aba “Flow Run” tornou-se ativa e mostrou a missão `BLOCKED_NEEDS_USER`, pergunta e opção disponível. Nenhuma chamada a provedor foi feita. | PASS | — | `screenshots/mission-attention-open-detail-native-xwayland.png`; `logs/native-mission-attention-route-open-retry-xwayland.log` |
| ATTN-NATIVE-03 | Layout | Inspecionar valores e rótulos do detalhe após aplicar SCSS Module | Resumo de estado e evidência agora usa colunas e espaçamento; pergunta e ações permanecem legíveis em 1280×800. | PASS | — | Screenshot acima; `web/src/features/work/FlowRunSurface.style.test.ts` |
| MASCOT-SINGLE-01 | Mascote | Inspecionar o HUD nativo e executar regressões focadas existentes | Há uma única arte NEX. Barra, Mascote e Bandeja são apresentações do mesmo HUD; não existe modo ou renderer Companion 2D separado. | PASS | — | `screenshots/native-nexus-hud-single-mascot-xwayland.png`; `MASCOT_VALIDATION.md` |

## Correções reproduzidas

- O clique no cartão de atenção fechava o drawer, mas não deixava uma aba de decisão ativa. O callback agora abre a superfície `flow-run` identificada no workspace e sincroniza a rota. A rota `missions/:id` é permitida, enquanto Composer, criação e histórico continuam sob o gate experimental.
- Os cartões `.nx-flow-run-status` não tinham estilo registrado; `strong` e `span` apareciam colados. O resumo usa `FlowRunSurface.module.scss` com grid, gap e quebra segura de texto.
- Foi preservada a decisão de produto: um mascote NEX único. Nenhum renderer ou modo Companion 2D adicional foi integrado, e a apresentação Mascote existente não foi removida.

## Artefato e ambiente

- Build: `make build-desktop-wails`; ELF Linux AMD64. `ldd` não reportou bibliotecas ausentes.
- Janela Wails real: 1280×800; GNOME em sessão Wayland, aplicação em XWayland. O processo foi iniciado com `G_DEBUG=fatal-criticals`, perfil de dados isolado e HUD spawn desativado.
- O log registra avisos de aceleração DRI/VMware do host, sem `GDK-CRITICAL` durante esta validação.
- Checksums do binário e capturas: `logs/mission-attention-and-single-mascot-sha256.txt`.
- Lista e fingerprint das fontes usadas: `logs/source-provenance-p2-mission-attention.txt`.

## Reprodução

```bash
cd /projetos/nexus
make build-desktop-wails
GDK_BACKEND=x11 G_DEBUG=fatal-criticals \
  NEXUS_DATA_DIR=/tmp/nexus-mission-attention-p2.zDpzg5/data \
  NEXUS_CONFIG_DIR=/tmp/nexus-mission-attention-p2.zDpzg5/config \
  NEXUS_STATE_DIR=/tmp/nexus-mission-attention-p2.zDpzg5/state \
  XDG_CACHE_HOME=/tmp/nexus-mission-attention-p2.zDpzg5/cache \
  NEXUS_HUD_SPAWN=0 ./cmd/nexus-desktop/build/bin/nexus-desktop
```

O perfil nativo contém uma fixture local controlada. Ela serve apenas para validar lista, filtros, navegação e renderização; não valida eventos reais de agentes, integração externa ou decisões de provedor. O close/reopen de SQLite e a rejeição HTTP sem autenticação têm testes determinísticos. A sequência HTTP/UI com reinício real do servidor, isolamento entre usuários/projetos autorizados e os estados vazio/loading/erro ainda está pendente em P2.

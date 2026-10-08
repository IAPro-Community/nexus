# ADR-2026-10-07 — Host virtual Wails na política de origem

Status: implementado e integrado à `release/v1.0.0` no commit `6bab37ea`; regressão Windows confirmada novamente no candidato beta.8.
Base: release/v1.0.0, 52f35c04181b85379219ab55fc3192c63a5440a8.

## Contexto

No Wails v3 para Windows, o AssetServer pode encaminhar requisições diretamente ao handler do Core embutido com Host wails.localhost e Origin http://wails.localhost. A classificação reconhecia a origem Wails, mas as validações do Host não incluíam wails.localhost. O bootstrap Desktop e a criação do primeiro projeto terminavam em HTTP 403 invalid origin.

## Decisão

Permitir o Host exato wails.localhost somente dentro dos ramos que já exigem origem desktop reconhecida. Manter validação de origem, autenticação e CSRF. Não aceitar sufixos, curingas ou hosts parecidos.

## Alternativas rejeitadas

- Aceitar qualquer host terminado em .localhost ampliaria a fronteira sem necessidade.
- Desabilitar validação de Origin para Desktop enfraqueceria a proteção contra requisições cross-site.
- Corrigir somente a mensagem do frontend ocultaria o 403 sem permitir criar o projeto.

## Consequências e segurança

O Desktop Wails pode usar seu host virtual reservado ao falar com o Core embutido. Hosts como wails.localhost.evil.com e origens externas continuam negados. A exceção usa igualdade exata e exige os sinais Desktop já reconhecidos.

## Evidência

- `TestWindowsWailsVirtualHostSupportsAuthenticatedProjectCreation` passou no Windows no HEAD `7cc03499`: bootstrap retorna 200 e criação de projeto retorna 201 com Host/Origin exatos `wails.localhost` e sessão autenticada com CSRF.
- O mesmo teste confirma que host lookalike e origens externas continuam em 403.
- `go test -count=1 ./internal/control/originpolicy ./internal/control/web ./internal/desktop ./cmd/nexus-desktop ./cmd/nexus-hud` passou no Windows no HEAD `7cc03499`.
- Smokes nativos posteriores confirmaram inicialização do Desktop/Wails/WebView2; inspeção visual e interação completa continuam pendentes antes da distribuição.

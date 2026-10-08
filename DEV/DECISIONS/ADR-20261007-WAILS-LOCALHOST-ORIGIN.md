# ADR-2026-10-07 — Host virtual Wails na política de origem

Status: aceito para correção local; integração num SHA imutável pendente.
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

- Teste HTTP Windows cria sessão Desktop, acessa desktop bootstrap e cria projeto com Host/Origin wails.localhost.
- Casos negativos validam hosts lookalike e origens externas.
- Os testes focados de originpolicy/web/desktop e go test -count=1 ./... passaram no checkout Windows.
- A correção ainda está sem commit; certificação exact-SHA e smoke Wails interativo permanecem necessários antes da distribuição.

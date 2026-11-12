# Revogação de sessão com terminal conectado — 2026-11-09

## Resultado local

- `AuthManager.WatchSession` emite revogação em logout, rotação e expiração observada.
- `TerminalHub` observa a revogação e encerra o WebSocket; cada frame recebido também renova a atividade somente enquanto a sessão continua válida.
- O teste integrado abre dois clientes autenticados ligados ao mesmo SessionHost. Depois que o cliente com lease `CONTROL` é revogado, seu socket fecha, o lease passa ao outro cliente e o processo supervisionado aceita input somente do cliente ainda autenticado.
- O teste revelou uma corrida em `events.Bus`: `Publish` enviava para o canal fora do lock enquanto `Unsubscribe` o fechava. Os envios agora são sincronizados com a remoção/fechamento.

## Evidência executada

```text
go test -race ./internal/control/events ./internal/control/web -run 'Test(UnsubscribeRacesWithPublish|AuthManagerSessionWatcher|TerminalHandoffAuthenticatedClientsShareProcessAndPersistIdentity)$' -count=1
PASS: internal/control/events
PASS: internal/control/web
```

## Limites

- Esta é evidência local de código-fonte em `release/v1.0.0`, baseada em SessionHost de teste. Não cobre app empacotado, túnel, dispositivo físico nem o SHA final homologado.
- A expiração durante uma conexão ativa é coberta pela sinalização testável de `SessionActive` e pelo monitor do handler; o E2E integrado cobre logout/revogação explícita.

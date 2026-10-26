# Mission notification delivery — P3

Data: 2026-10-25  
Branch: `release/v1.0.0`  
Base Git: `7a04f1b4cf5950b97b4ea1e156dda31ac513823b`  
Estado testado: base mais overlay local identificado em `logs/source-provenance-p3-notifications-2026-10-25.txt`; worktree não congelada.

## Resultado

P3 implementado e validado no nível de store, aplicação, retry e transporte nativo Linux. A Mission continua sendo a única fonte de verdade; a outbox armazena apenas cursor e estado de entrega. A homologação global permanece `NO_GO_LINUX` enquanto P2 e P4–P9 e os gates gerais não forem concluídos.

O Attention Center existente continua mostrando bloqueios, falhas e conclusões a partir do `MissionRun`. Entregas nativas são best-effort para bloqueios acionáveis e falhas terminais explicitamente escaladas. Conclusões permanecem no Attention Center. Nenhum prompt, mensagem do agente, credencial ou deep link é incluído no texto nativo; falhas de transporte são redigidas antes de persistir.

## Cobertura

| ID | Área | Procedimento | Resultado observado | Status | Evidência | Severidade |
|---|---|---|---|---|---|---|
| P3-001 | Outbox SQLite | Criar baseline, transicionar a blocker, observar repetidamente, fechar e reabrir o banco | Uma intenção e uma entrega pendente; repetição não duplica; janela de retry sobrevive ao reopen | PASS | `logs/go-test-p3-notifications.log`; `internal/nexus/store/mission_notifications_test.go` | — |
| P3-002 | Aplicação | Publicar `MISSION_STARTED`, persistir blocker e publicar `HUMAN_INTERVENTION_CREATED` pelo `RunApplicationService` | Evento cria uma única entrega nativa e persiste o recibo | PASS | `logs/go-test-p3-notifications.log`; `internal/nexus/mission_notifications_test.go` | — |
| P3-003 | Falha/retry | Falhar no transporte, reabrir SQLite e reconstruir o serviço | Entrega pendente é retomada depois do prazo; segundo envio registra `DELIVERED`; chamadas após sucesso são deduplicadas | PASS | `logs/go-test-p3-notifications.log`; `logs/go-race-p3-notifications.log` | — |
| P3-004 | Capacidade nativa ausente | Fixture do notifier informa indisponibilidade; avançar relógio controlado e disponibilizar transporte | Não tenta envio nem incrementa tentativas enquanto indisponível; entrega quando a capacidade aparece | PASS | `logs/go-test-p3-notifications.log` | — |
| P3-005 | Filtro de ruído | Exercitar `QUOTA_EXHAUSTED`, `RATE_LIMITED`, retry, remediação, execução normal, falha sem escalonamento e conclusão | Estados automáticos e conclusão não geram toast nativo; conclusão continua projetada no Attention Center; falha escalonada é elegível | PASS | `logs/go-test-p3-notifications.log` | — |
| P3-006 | Privacidade | Inspecionar payload e erro de transporte com cabeçalho Bearer | Payload é genérico, sem URL, conteúdo de intervenção ou segredo; segredo é removido do erro persistido | PASS | `logs/go-test-p3-notifications.log` | — |
| P3-007 | Throttle compartilhado | Falhar uma chamada, repetir dentro do cooldown e concluir a segunda | A primeira falha não fica marcada como enviada; retry chega ao SO, e somente o sucesso ativa o cooldown | PASS | `logs/go-test-p3-notifications.log` | — |
| P3-008 | Linux nativo | `NEXUS_ENABLE_NATIVE_NOTIFICATION_SMOKE=1 go test ./internal/control/notify -run '^TestDesktopNotificationSmoke$' -count=1 -v` em GNOME/Wayland | `notify-send` disponível; transporte D-Bus aceitou texto de homologação sem dados de Mission | PASS | `logs/native-notification-smoke-p3-2026-10-25.log` | — |

Os testes de capacidade indisponível usam notifier controlado e comprovam a política de capability-gate. O smoke real exercita o transporte Linux disponível nesta sessão, mas não captura pixels do toast nem prova sua apresentação visual em outro ambiente GNOME.

## Implementação

- Migração `0031_mission_notification_delivery.sql` separa cursor, intenção e tentativas por canal.
- `MissionNotificationService` reconcilia execuções na inicialização, deduplica por `AttentionDedupKey`, agenda retry exponencial limitado a cinco minutos e recupera entregas pendentes.
- `RunApplicationService` observa eventos publicados depois da gravação do estado durável. Erros de transporte não alteram o estado da Mission.
- O throttle do notifier só registra cooldown após o transporte retornar sucesso.
- Os textos nativos não contêm links. O usuário abre o Attention Center pela interface do Nexus.

## Reprodução

```bash
go test ./internal/nexus/... ./internal/control/notify -count=1
go test -race ./internal/control/notify ./internal/nexus/store ./internal/nexus -run 'Test(Throttle|IsAvailableDelegates|MissionNotification|MissionNotifications|RunApplicationPublishesMissionBlocker)' -count=5
go vet ./internal/control/notify ./internal/nexus/...
NEXUS_ENABLE_NATIVE_NOTIFICATION_SMOKE=1 go test ./internal/control/notify -run '^TestDesktopNotificationSmoke$' -count=1 -v
```

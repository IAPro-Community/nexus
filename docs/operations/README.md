# Operação e suporte

## Operação local

| Necessidade | Comando | Observação |
| --- | --- | --- |
| diagnóstico | `nexus doctor` | use `--json` / `--bundle` para evidence sanitizada |
| providers | `nexus providers status --json` | UNKNOWN não é disponibilidade |
| runtimes | `nexus ps --json` | confira antes de recuperar/encerrar |
| profile/quota | `nexus status [provider:profile]` | pode ser LIVE/CACHED/UNKNOWN |
| segurança | `nexus security --json` | revise boundaries antes de corrigir |
| versão | `nexus version --json` | registre em reports/issues |

## Operação de Missions

Para execução planejada, observe no Flow Run:

- estado da Mission/package;
- Agent e recurso selecionado;
- attempts;
- dependências;
- coordination requests;
- WorkReceipts;
- blockers/intervenções;
- verificação global.

Não force retry de um dispatch com outcome desconhecido.

## Recovery

Quando uma sessão ou Mission parar:

1. diagnostique o estado real;
2. diferencie Agent identity de RuntimeGeneration;
3. diferencie PAUSED de BLOCKED_NEEDS_USER;
4. reconcilie side effects externos antes de retry;
5. use a ação explícita de retomada oferecida pelo runtime.

## Limites de operação

Nexus é uma workstation local. A API Web é autenticada e destinada às
superfícies do próprio produto; não a exponha como serviço público.

Tunnel/acesso remoto exige decisão consciente de risco.

## Referências

- [Troubleshooting](troubleshooting.md)
- [Segurança PT-BR](../security.pt-BR.md)
- [Guardrails](../architecture/guardrails.md)
- [Suporte de plataformas](platform-support.md)
- [Release/certificação](release.md)
- [Missions](../product/missions.md)

# Segurança e isolamento

[🇬🇧 English](security.md) | **🇧🇷 Português** | [🇪🇸 Español](security.es.md)

O Nexus é local-first e executa ferramentas capazes de modificar código. A
segurança é tratada como parte do runtime, não apenas como documentação.

## Princípios

- fail closed quando não há prova suficiente;
- menor privilégio;
- profiles/credenciais isolados por provider quando suportado;
- worktrees para escrita autônoma;
- reviewers/investigators read-only quando o provider comprova o modo;
- push/deploy sem autorização implícita;
- redaction de evidence/logs;
- UNKNOWN não equivale a seguro ou disponível.

## Isolamento de credenciais

Profiles usam diretórios/config stores específicos do provider. O Nexus evita
tratar credenciais de um profile como compartilháveis por padrão.

Autenticação continua pertencendo ao CLI/provider oficial.

## Workspace não é uma VM

Mesmo com isolamento de profile e worktree, processos podem rodar com o mesmo
usuário do sistema operacional. Não confunda isso com um sandbox de máquina
virtual.

O `AutonomyContract` e o isolamento real do provider definem o que pode ser
automatizado.

## Side effects

Operações sensíveis dependem de policy explícita:

- Git destrutivo;
- push;
- deploy/publicação;
- rede;
- secrets;
- serviços pagos.

Quando uma continuação não pode provar segurança, o Nexus bloqueia em vez de
repetir a operação.

Veja [Guardrails](architecture/guardrails.md).

## Web

O servidor usa loopback por padrão e sessão autenticada. A URL de bootstrap é
sensível e temporária. Não exponha a Web/API diretamente à internet.

Para acesso remoto, trate túnel/VPN como uma decisão explícita de segurança.

## Diagnóstico

```bash
nexus security --json
nexus doctor --json
```

Nunca publique tokens, cookies, chaves, arquivos de credential ou dados
privados em bundles/issues.

Para vulnerabilidades, siga o [SECURITY.md raiz](../SECURITY.md).
